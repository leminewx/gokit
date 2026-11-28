package codec

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.nets.codec: "

var (
	// ErrInvalidLength 表示帧长度非法（负数或非正缓冲区大小）
	ErrInvalidLength = errors.New(_ERROR_PREFIX + "invalid message length")
	// ErrMessageTooLarge 表示帧长度超过 maxMessageSize 限制
	ErrMessageTooLarge = errors.New(_ERROR_PREFIX + "message too large")
	// ErrUnsupportedLengthField 表示长度字段宽度不受支持
	ErrUnsupportedLengthField = errors.New(_ERROR_PREFIX + "unsupported length field size")
	// ErrInvalidVarint 表示 uvarint 长度字段编码非法
	ErrInvalidVarint = errors.New(_ERROR_PREFIX + "invalid uvarint")
	// ErrReadFailed 表示从连接读取失败
	ErrReadFailed = errors.New(_ERROR_PREFIX + "failed to read message")
	// ErrWriteFailed 表示向连接写入失败
	ErrWriteFailed = errors.New(_ERROR_PREFIX + "failed to write message")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}

// wrapReadError 构造读取失败的包装错误，保留哨兵与底层错误链
func wrapReadError(codec string, err error) error {
	return fmt.Errorf("%w: %s: %w", ErrReadFailed, codec, err)
}

// wrapWriteError 构造写入失败的包装错误，保留哨兵与底层错误链
func wrapWriteError(codec string, err error) error {
	return fmt.Errorf("%w: %s: %w", ErrWriteFailed, codec, err)
}
