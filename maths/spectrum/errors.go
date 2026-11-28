package spectrum

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.spectrum: "

var (
	// ErrEmptyInput 表示输入序列为空
	ErrEmptyInput = errors.New(_ERROR_PREFIX + "values must not be empty")
	// ErrInvalidLength 表示序列长度不是 2 的幂或窗长度非正
	ErrInvalidLength = errors.New(_ERROR_PREFIX + "length must be a positive power of two")
	// ErrInvalidWindow 表示窗函数类型不受支持
	ErrInvalidWindow = errors.New(_ERROR_PREFIX + "window type is not supported")
	// ErrInvalidSampleRate 表示采样率非正或非有限
	ErrInvalidSampleRate = errors.New(_ERROR_PREFIX + "sample rate must be positive and finite")
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
