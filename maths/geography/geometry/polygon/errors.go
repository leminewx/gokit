package polygon

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.geography.geometry.polygon: "

var (
	// ErrTooFewVertices 表示构建多边形所需的顶点不足 3 个
	ErrTooFewVertices = errors.New(_ERROR_PREFIX + "vertices are too few, at least 3 are required")
)

// wrapVertexError 构造顶点不足的错误，保留可用于 errors.Is 的哨兵错误
func wrapVertexError(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrTooFewVertices}, args...)...)
}
