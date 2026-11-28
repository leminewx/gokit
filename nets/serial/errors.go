package serial

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.nets.serial: "

var (
	// ErrUnsupported 表示当前平台不支持串口。
	ErrUnsupported = errors.New(_ERROR_PREFIX + "serial ports are not supported on this platform")
	// ErrInvalidConfig 表示串口配置非法。
	ErrInvalidConfig = errors.New(_ERROR_PREFIX + "invalid serial config")
	// ErrOpenFailed 表示打开串口失败。
	ErrOpenFailed = errors.New(_ERROR_PREFIX + "failed to open port")
	// ErrReadFailed 表示从串口读取失败。
	ErrReadFailed = errors.New(_ERROR_PREFIX + "failed to read from port")
	// ErrWriteFailed 表示向串口写入失败。
	ErrWriteFailed = errors.New(_ERROR_PREFIX + "failed to write to port")
	// ErrCloseFailed 表示关闭串口失败。
	ErrCloseFailed = errors.New(_ERROR_PREFIX + "failed to close port")
	// ErrTermios 表示读取或设置 termios 失败。
	ErrTermios = errors.New(_ERROR_PREFIX + "failed to access termios")
	// ErrPollFailed 表示轮询串口超时或失败。
	ErrPollFailed = errors.New(_ERROR_PREFIX + "failed to poll port")
	// ErrCommState 表示读取或设置 Windows 串口状态失败。
	ErrCommState = errors.New(_ERROR_PREFIX + "failed to access comm state")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
