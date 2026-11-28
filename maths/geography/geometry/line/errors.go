package line

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.geography.geometry.line: "

var (
	// ErrTooFewPoints 表示构建线所需要的点不足
	ErrTooFewPoints = errors.New(_ERROR_PREFIX + "points are too few, at least two are needed")
	// ErrIndexOutOfRange 表示点索引超出有效范围
	ErrIndexOutOfRange = errors.New(_ERROR_PREFIX + "index out of range")
)

// wrapIndexError 构造索引越界的错误，保留可用于 errors.Is 的哨兵错误
func wrapIndexError(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrIndexOutOfRange}, args...)...)
}
