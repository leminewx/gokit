package ratelimiter

import (
	"errors"
	"fmt"
)

// _ERROR_PREFIX 定义本包错误消息的统一前缀。
const _ERROR_PREFIX = "gokit.ratelimiter: "

var (
	// ErrInvalidConfig 表示构造限流器时传入的速率或容量非法。
	ErrInvalidConfig = errors.New(_ERROR_PREFIX + "rate must not be negative and capacity must be positive")
	// ErrUnattainable 表示请求的令牌数永远无法被满足（超过容量或速率为 0）。
	ErrUnattainable = errors.New(_ERROR_PREFIX + "requested tokens can never be satisfied")
)

// wrapError 为错误补充包前缀并保留错误链，便于调用方使用 errors.Is。
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}

	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
