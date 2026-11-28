//go:build windows

package serial

import (
	"errors"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	_GENERIC_READ          = 0x80000000 // 指定对设备的读权限
	_GENERIC_WRITE         = 0x40000000 // 指定对设备的写权限
	_OPEN_EXISTING         = 3          // 指示必须打开已存在的设备（串口必须存在）
	_FILE_ATTRIBUTE_NORMAL = 0x80       // 默认文件属性
	_FILE_FLAG_OVERLAPPED  = 0x40000000 // 异步 I/O 标志，代码中未使用（同步模式），仅作为注释说明
)

const (
	_F_BINARY              = 0x0001 // 二进制模式（必须为 1）
	_F_PARITY              = 0x0002 // 是否启用奇偶校验
	_F_OUTX_CTS_FLOW       = 0x0004 // CTS 输出流控制
	_F_OUTX_DSR_FLOW       = 0x0008 // DSR 输出流控制
	_F_DTR_CTRL            = 0x0030 // DTR 控制（2 位)
	_F_DSR_SENSITIVITY     = 0x0040 // DSR 敏感度
	_F_TX_CONTINUE_ON_XOFF = 0x0080 // XOFF 后继续发送
	_F_OUT_X               = 0x0100 // XON/XOFF 输出流控
	_F_IN_X                = 0x0200 // XON/XOFF 输入流控
	_F_ERROR_CHAR          = 0x0400 // 是否替换错误字符
	_F_NULL                = 0x0800 // 是否丢弃 NULL 字符
	_F_RTS_CTRL            = 0x3000 // RTS 控制（2 位)
	_F_ABORT_ON_ERROR      = 0x4000 // 出错时中止读写)
)

/*
fDtrControl/fRtsControl 控制位的值（需移位后合并到 Flags）：
+-------+----------------------------------
| value | description
+-------+----------------------------------
| 0     | 禁用（Disable）
| 1     | 启用（Enable）
| 2     | 握手（Handshake）
| 3     | RTS 切换模式（仅用于 RTS，极少使用）
+-------+----------------------------------
*/

const (
	_DTR_CTRL_DISABLE   = 0 << 4 // 值 0 左移 4 位
	_DTR_CTRL_ENABLE    = 1 << 4 // 值 1 左移 4 位
	_DTR_CTRL_HANDSHAKE = 2 << 4 // 值 2 左移 4 位

	_RTS_CTRL_DISABLE   = 0 << 12 // 值 0 左移 12 位
	_RTS_CTRL_ENABLE    = 1 << 12 // 值 1 左移 12 位
	_RTS_CTRL_HANDSHAKE = 2 << 12 // 值 2 左移 12 位
	_RTS_CTRL_TOGGLE    = 3 << 12 // 值 3 左移 12 位（切换模式）
)

// port 串口对象
type port struct {
	config       Config         // 串口配置
	handle       windows.Handle // Windows 文件句柄，代表打开的串口
	readTimeout  time.Duration  // 读超时时间（0表示无限/非阻塞，>0表示有限超时）
	writeTimeout time.Duration  // 写超时时间（0表示无限/非阻塞，>0表示有限超时）
}

// open 打开 windows 串口
func open(config Config) (Port, error) {
	// 转换路径：COM1x 及以上需要格式 "\\\\.\\COM1x"
	path, err := syscall.UTF16PtrFromString("\\\\.\\" + config.Name)
	if err != nil {
		return nil, wrapError(ErrOpenFailed, "port name %q: %w", config.Name, err)
	}

	// 创建文件句柄
	handle, err := windows.CreateFile(
		path,                         //转换后的 UTF-16 字符串指针
		_GENERIC_READ|_GENERIC_WRITE, // 可读可写
		0,                            // 独占访问（不允许其他进程共享）
		nil,                          // 默认安全属性
		_OPEN_EXISTING,               // 必须存在
		_FILE_ATTRIBUTE_NORMAL,       // 同步 I/O 模式（未使用 _FILE_FLAG_OVERLAPPED）
		0,                            // 模板文件句柄（未使用）
	)
	if err != nil {
		return nil, wrapError(ErrOpenFailed, "%q: %w", config.Name, err)
	}

	// 设置串口配置
	p := &port{handle: handle, config: config}
	if err := p.SetConfig(config); err != nil {
		_ = windows.CloseHandle(handle)
		return nil, err
	}

	return p, nil
}

// SetConfig 设置串口配置
func (own *port) SetConfig(config Config) error {
	// 获取操作系统 DCB 控制块
	var dcb windows.DCB
	dcb.DCBlength = uint32(unsafe.Sizeof(dcb))
	if err := windows.GetCommState(own.handle, &dcb); err != nil {
		return wrapError(ErrCommState, "get: %w", err)
	}

	// 设置基本参数
	dcb.BaudRate = uint32(config.Baud)
	dcb.ByteSize = byte(config.DataBits)

	// 设置停止位
	switch config.StopBits {
	case 1:
		dcb.StopBits = 0 // 1 位
	case 2:
		dcb.StopBits = 2 // 2 位
	default:
		return syscall.EINVAL
	}

	// 设置校验位
	switch config.Parity {
	case 'N':
		dcb.Parity = 0 // 无校验
	case 'O':
		dcb.Parity = 1 // 奇校验
	case 'E':
		dcb.Parity = 2 // 偶校验
	default:
		return syscall.EINVAL
	}

	// 清除所有流控相关位
	dcb.Flags &^= _F_OUTX_CTS_FLOW | _F_OUTX_DSR_FLOW | _F_DTR_CTRL | _F_RTS_CTRL | _F_OUT_X | _F_IN_X

	// 强制设置二进制模式（必须）
	dcb.Flags |= _F_BINARY

	// 设置默认的 DTR/RTS 为启用（非握手）
	dcb.Flags |= _DTR_CTRL_ENABLE | _RTS_CTRL_ENABLE

	// 如果启用了奇偶校验，设置 _F_PARITY 位
	if config.Parity != 'N' {
		dcb.Flags |= _F_PARITY
	}

	// 根据配置修改流控制
	switch config.FlowControl {
	// RTS/CTS 硬件流控
	case FLOW_CONTROL_RTS_CTS:
		dcb.Flags |= _F_OUTX_CTS_FLOW    // 启用 CTS 流控
		dcb.Flags &^= _RTS_CTRL_ENABLE   // 清除之前设置的 RTS 启用位
		dcb.Flags |= _RTS_CTRL_HANDSHAKE // 设置为握手模式

	// DTR/DSR 硬件流控
	case FLOW_CONTROL_DTR_DSR:
		dcb.Flags |= _F_OUTX_DSR_FLOW    // 启用 DSR 流控
		dcb.Flags &^= _DTR_CTRL_ENABLE   // 清除之前设置的 DTR 启用位
		dcb.Flags |= _DTR_CTRL_HANDSHAKE // 设置为握手模式

	// XON/XOFF 软件流控
	case FLOW_CONTROL_XON_XOFF:
		dcb.Flags |= _F_OUT_X | _F_IN_X // 启用软件流控
		dcb.XonChar = 0x11
		dcb.XoffChar = 0x13
		dcb.XonLim = 100
		dcb.XoffLim = 100
		// 硬件流控保持默认启用（非握手），已设置 _DTR_CTRL_ENABLE | _RTS_CTRL_ENABLE

	case FLOW_CONTROL_NONE:
		// 无流控：保持默认（DTR/RTS 启用但不用于握手）
	}

	// 应用 DCB
	if err := windows.SetCommState(own.handle, &dcb); err != nil {
		return wrapError(ErrCommState, "set: %w", err)
	}

	// 设置超时
	return own.SetTimeouts(config.ReadTimeout, 0)
}

/*
SetTimeouts 设置串口超时
  - readTimeout > 0: 设置总超时常量为指定的毫秒数，而字符间隔超时设为 0（表示不等待字符间隔），这种配置使得 ReadFile 在指定时间内未读到任何数据时返回 0
  - readTimeout == 0，设置 ReadIntervalTimeout = 0xFFFFFFFF，这是 Windows 的特殊值，表示读操作应立即返回（若缓冲区有数据则返回，否则返回 0），相当于非阻塞读
  - 写超时未设置（保持 0），意味着写操作将无限等待
*/
func (own *port) SetTimeouts(readTimeout, writeTimeout time.Duration) error {
	timeouts := &windows.CommTimeouts{}

	// 设置读超时
	if readTimeout > 0 { // 有超时
		timeouts.ReadIntervalTimeout = 0
		timeouts.ReadTotalTimeoutConstant = uint32(readTimeout / time.Millisecond)
	} else { // 无超时
		timeouts.ReadIntervalTimeout = 0xFFFFFFFF
		timeouts.ReadTotalTimeoutConstant = 0
	}
	timeouts.ReadTotalTimeoutMultiplier = 0

	// 设置写超时
	if writeTimeout > 0 {
		ms := writeTimeout / time.Millisecond
		timeouts.WriteTotalTimeoutConstant = uint32(ms)
	} else {
		// 如果 writeTimeout <= 0，设置为 0 表示无限等待
		timeouts.WriteTotalTimeoutConstant = 0
	}
	timeouts.WriteTotalTimeoutMultiplier = 0

	// 应用超时
	if err := windows.SetCommTimeouts(own.handle, timeouts); err != nil {
		return wrapError(ErrCommState, "timeouts: %w", err)
	}

	// 设置成功后保存当前超时值
	own.readTimeout = readTimeout
	own.writeTimeout = writeTimeout

	return nil
}

// Read 读取数据
func (own *port) Read(data []byte) (int, error) {
	var done uint32
	if err := windows.ReadFile(own.handle, data, &done, nil); err != nil {
		// 检查错误是否是因为超时
		if errors.Is(err, windows.ERROR_TIMEOUT) {
			return 0, ErrTimeout
		}

		return 0, wrapError(ErrReadFailed, "%w", err)
	}

	// 如果读取 0 字节且没有错误，检查是否是因为超时
	// 当前设置了有限读超时，且未读到数据，视为超时
	if done == 0 && own.readTimeout > 0 {
		return 0, ErrTimeout
	}

	return int(done), nil
}

// Write 写数据到串口
func (own *port) Write(data []byte) (int, error) {
	var done uint32
	if err := windows.WriteFile(own.handle, data, &done, nil); err != nil {
		// 检查错误是否是因为超时
		if errors.Is(err, windows.ERROR_TIMEOUT) {
			return 0, ErrTimeout
		}

		return 0, wrapError(ErrWriteFailed, "%w", err)
	}

	// 写操作可能因超时而部分写入，但 WriteFile 通常会在超时时返回错误
	// 这里只需依赖系统错误，但为了完整性，也可类似处理
	// 如果设置了写超时且未写入任何字节，视为超时
	if done == 0 && own.writeTimeout > 0 {
		return 0, ErrTimeout
	}

	return int(done), nil
}

// Close 关闭串口
func (own *port) Close() error {
	if err := windows.CloseHandle(own.handle); err != nil {
		return wrapError(ErrCloseFailed, "%w", err)
	}

	return nil
}
