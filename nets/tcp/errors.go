package tcp

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.nets.tcp: "

var (
	// ErrDialFailed 表示拨号建立连接失败
	ErrDialFailed = errors.New(_ERROR_PREFIX + "failed to dial")
	// ErrConnClosed 表示连接已被关闭
	ErrConnClosed = errors.New(_ERROR_PREFIX + "connection closed")
	// ErrNotConnected 表示当前没有可用连接
	ErrNotConnected = errors.New(_ERROR_PREFIX + "connection is not established")
	// ErrReconnectDisabled 表示连接已断开且未启用自动重连
	ErrReconnectDisabled = errors.New(_ERROR_PREFIX + "connection lost and auto reconnect is disabled")
	// ErrReconnectFailed 表示重连达到最大次数仍失败
	ErrReconnectFailed = errors.New(_ERROR_PREFIX + "failed to reconnect")
	// ErrReconnectInterrupted 表示重连过程被 Close 中断
	ErrReconnectInterrupted = errors.New(_ERROR_PREFIX + "reconnect interrupted")
	// ErrSetDeadline 表示设置读写超时失败
	ErrSetDeadline = errors.New(_ERROR_PREFIX + "failed to set deadline")
	// ErrCloseFailed 表示关闭底层连接失败
	ErrCloseFailed = errors.New(_ERROR_PREFIX + "failed to close connection")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
