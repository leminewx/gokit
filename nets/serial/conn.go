package serial

import (
	"net"
	"sync"
	"time"
)

// 编译期断言：LocalAddr/RemoteAddr/SetDeadline/Read/Write/Close 是 net.Conn 钉死的方法名
var _ net.Conn = (*Connection)(nil)

// Connection 串口连接对象
type Connection struct {
	port       Port     // 串口对象
	closed     bool     // 是否已关闭
	localAddr  net.Addr // 通常为串口名
	remoteAddr net.Addr // 可设为一个固定值或与 local 相同

	readDeadline     time.Time     // 读超时（0表示无限/非阻塞，>0表示有限超时）
	writeDeadline    time.Time     // 写超时（0表示无限/非阻塞，>0表示有限超时）
	lastReadTimeout  time.Duration // 上次读超时时间
	lastWriteTimeout time.Duration // 上次写超时时间

	lock sync.Mutex
}

// NewConnection 创建新的串口连接
func NewConnection(cfg Config) (net.Conn, error) {
	port, err := Open(cfg)
	if err != nil {
		return nil, err
	}

	return &Connection{
		port:             port,
		localAddr:        addr(cfg.Name),
		remoteAddr:       addr(""),
		lastReadTimeout:  -1,
		lastWriteTimeout: -1,
	}, nil
}

// SetDeadline 设置读写超时
func (own *Connection) SetDeadline(deadline time.Time) error {
	own.lock.Lock()
	own.readDeadline = deadline
	own.writeDeadline = deadline
	own.lock.Unlock()
	return nil
}

// SetReadDeadline 设置读超时
func (own *Connection) SetReadDeadline(deadline time.Time) error {
	own.lock.Lock()
	own.readDeadline = deadline
	own.lock.Unlock()
	return nil
}

// SetWriteDeadline 设置写超时
func (own *Connection) SetWriteDeadline(deadline time.Time) error {
	own.lock.Lock()
	own.writeDeadline = deadline
	own.lock.Unlock()
	return nil
}

func (own *Connection) LocalAddr() net.Addr {
	return own.localAddr
}

func (own *Connection) RemoteAddr() net.Addr {
	return own.remoteAddr
}

// Read 读取数据
func (own *Connection) Read(data []byte) (int, error) {
	own.lock.Lock()
	defer own.lock.Unlock()

	// 检查读 deadline 是否已过期
	if !own.readDeadline.IsZero() && time.Now().After(own.readDeadline) {
		return 0, &timeoutError{}
	}

	// 计算读超时
	readTimeout := computeTimeout(own.readDeadline)
	// 计算写超时（保持写 deadline 的语义）
	writeTimeout := computeTimeout(own.writeDeadline)

	// 设置超时（同时影响后续读写）
	if err := own.updateTimeouts(readTimeout, writeTimeout); err != nil {
		return 0, err
	}

	// 读取数据
	n, err := own.port.Read(data)
	if err != nil {
		return n, err
	}

	return n, nil
}

// Write 写数据
func (own *Connection) Write(data []byte) (int, error) {
	own.lock.Lock()
	defer own.lock.Unlock()

	// 检查写超时
	if !own.writeDeadline.IsZero() && time.Now().After(own.writeDeadline) {
		return 0, ErrTimeout
	}

	// 计算写超时
	writeTimeout := computeTimeout(own.writeDeadline)
	// 计算读超时（保持读 deadline 的语义）
	readTimeout := computeTimeout(own.readDeadline)

	// 设置超时（同时影响后续读和写）
	if err := own.updateTimeouts(readTimeout, writeTimeout); err != nil {
		return 0, err
	}

	// 写数据
	n, err := own.port.Write(data)
	if err != nil {
		return n, err
	}

	return n, nil
}

// updateTimeouts 更新超时时间
func (own *Connection) updateTimeouts(read, write time.Duration) error {
	if read == own.lastReadTimeout && write == own.lastWriteTimeout {
		return nil
	}

	if err := own.port.SetTimeouts(read, write); err != nil {
		return err
	}

	own.lastReadTimeout, own.lastWriteTimeout = read, write
	return nil
}

// Close 关闭串口连接
func (own *Connection) Close() error {
	own.lock.Lock()
	defer own.lock.Unlock()

	if own.closed {
		return nil
	}

	// 标记已关闭，避免重复 Close 对同一句柄二次关闭
	own.closed = true
	return own.port.Close()
}

// computeTimeout 根据 deadline 返回应该设置的超时时间
//   - 如果 deadline 为零值，返回 0（表示无限等待，具体根据你的库定义）
//   - 如果 deadline 已过期，返回一个小于等于 0 的值，调用者应直接返回超时错误
func computeTimeout(deadline time.Time) time.Duration {
	if deadline.IsZero() {
		return 0 // 0 表示无限，可根据你的需要调整（如 -1 表示无限）
	}

	remaining := time.Until(deadline)
	if remaining < 0 {
		return -1 // 负数表示已过期
	}

	return remaining
}
