package filter

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.filter: "

var (
	// ErrEmptyInput 表示输入序列为空
	ErrEmptyInput = errors.New(_ERROR_PREFIX + "values must not be empty")
	// ErrInvalidWindow 表示窗口长度超出支持范围或与序列长度不匹配
	ErrInvalidWindow = errors.New(_ERROR_PREFIX + "window must be an odd number within the sequence length")
	// ErrInvalidThreshold 表示 Hampel 阈值非正或非有限
	ErrInvalidThreshold = errors.New(_ERROR_PREFIX + "threshold must be positive and finite")
	// ErrInvalidAlpha 表示指数平滑系数不在 (0, 1] 内
	ErrInvalidAlpha = errors.New(_ERROR_PREFIX + "alpha must be within (0, 1]")
	// ErrNonFinite 表示输入出现 NaN / Inf
	ErrNonFinite = errors.New(_ERROR_PREFIX + "encountered a non-finite value")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
