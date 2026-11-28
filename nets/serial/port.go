package serial

import (
	"io"
	"strings"
	"time"
	"unicode"
)

const (
	// _DEFAULT_BAUD 定义未指定波特率时使用的默认值。
	_DEFAULT_BAUD = 9600
	// _DEFAULT_DATA_BITS 定义未指定数据位时使用的默认值。
	_DEFAULT_DATA_BITS = 8
	// _DEFAULT_STOP_BITS 定义未指定停止位时使用的默认值。
	_DEFAULT_STOP_BITS = 1
	// _DEFAULT_PARITY 定义未指定校验位时使用的默认值。
	_DEFAULT_PARITY rune = 'N'
)

// Config 包含串口配置参数。
type Config struct {
	Name        string          // 串口名：/dev/ttyS0、COM1
	Baud        int             // 波特率：9600、115200 等
	DataBits    int             // 数据位：5、6、7、8
	StopBits    int             // 停止位：1、2
	Parity      rune            // 校验位：N-无校验；O-奇校验；E-偶校验
	ReadTimeout time.Duration   // 读超时：0-表示阻塞
	FlowControl FlowControlType // 控制流：None、RTSCTS、DTRDSR、XONXOFF
}

// FlowControlType 定义串口流控类型。
type FlowControlType int

const (
	// FLOW_CONTROL_NONE 表示无流控。
	FLOW_CONTROL_NONE FlowControlType = iota
	// FLOW_CONTROL_RTS_CTS 表示 RTS/CTS 硬件流控。
	FLOW_CONTROL_RTS_CTS
	// FLOW_CONTROL_DTR_DSR 表示 DTR/DSR 硬件流控（仅 Windows 支持）。
	FLOW_CONTROL_DTR_DSR
	// FLOW_CONTROL_XON_XOFF 表示 XON/XOFF 软件流控。
	FLOW_CONTROL_XON_XOFF
)

// String 返回流控类型的可读名称。
func (own FlowControlType) String() string {
	switch own {
	case FLOW_CONTROL_NONE:
		return "none"
	case FLOW_CONTROL_RTS_CTS:
		return "rts-cts"
	case FLOW_CONTROL_DTR_DSR:
		return "dtr-dsr"
	case FLOW_CONTROL_XON_XOFF:
		return "xon-xoff"
	default:
		return "unknown"
	}
}

// Normalize 返回一份填充默认值、校验位统一为大写后的配置副本。
//
// 只处理可以安全推断的零值；非法取值保留原样，交由 Validate 报错。
func (own Config) Normalize() Config {
	config := own
	config.Name = strings.TrimSpace(config.Name)
	config.Parity = unicode.ToUpper(config.Parity)

	if config.Baud <= 0 {
		config.Baud = _DEFAULT_BAUD
	}
	if config.DataBits == 0 {
		config.DataBits = _DEFAULT_DATA_BITS
	}
	if config.StopBits == 0 {
		config.StopBits = _DEFAULT_STOP_BITS
	}
	if config.Parity == 0 {
		config.Parity = _DEFAULT_PARITY
	}

	return config
}

// Validate 校验配置是否合法。
func (own Config) Validate() error {
	if own.Name == "" {
		return wrapError(ErrInvalidConfig, "port name is empty")
	}

	if own.Baud <= 0 {
		return wrapError(ErrInvalidConfig, "baud rate must be positive, got %d", own.Baud)
	}

	if own.DataBits < 5 || own.DataBits > 8 {
		return wrapError(ErrInvalidConfig, "data bits must be 5-8, got %d", own.DataBits)
	}

	if own.StopBits != 1 && own.StopBits != 2 {
		return wrapError(ErrInvalidConfig, "stop bits must be 1 or 2, got %d", own.StopBits)
	}

	switch own.Parity {
	case 'N', 'O', 'E':
	default:
		return wrapError(ErrInvalidConfig, "parity must be N, O or E, got %q", own.Parity)
	}

	switch own.FlowControl {
	case FLOW_CONTROL_NONE, FLOW_CONTROL_RTS_CTS, FLOW_CONTROL_DTR_DSR, FLOW_CONTROL_XON_XOFF:
	default:
		return wrapError(ErrInvalidConfig, "unknown flow control %d", own.FlowControl)
	}

	if own.ReadTimeout < 0 {
		return wrapError(ErrInvalidConfig, "read timeout must not be negative, got %s", own.ReadTimeout)
	}

	return nil
}

// Port 定义串口的接口类型。
type Port interface {
	// ReadWriteCloser 继承 io.ReadWriteCloser
	io.ReadWriteCloser

	// SetConfig 修改串口配置（如波特率）
	SetConfig(config Config) error
	// SetTimeouts 设置串口的读写超时
	SetTimeouts(readTimeout, writeTimeout time.Duration) error
}

// Open 打开并配置串口。
//
// 打开前会先对配置做归一化（补默认值）与合法性校验；失败时返回的错误
// 可通过 errors.Is 与 ErrInvalidConfig / ErrUnsupported 比较。
func Open(config Config) (Port, error) {
	config = config.Normalize()
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return open(config)
}
