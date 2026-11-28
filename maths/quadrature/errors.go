package quadrature

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.quadrature: "

var (
	// ErrNilFunction 表示未提供被积函数
	ErrNilFunction = errors.New(_ERROR_PREFIX + "integrand must not be nil")
	// ErrInvalidInterval 表示积分区间非法
	ErrInvalidInterval = errors.New(_ERROR_PREFIX + "interval must satisfy a < b with finite endpoints")
	// ErrInvalidTolerance 表示容差非正或非有限
	ErrInvalidTolerance = errors.New(_ERROR_PREFIX + "tolerance must be positive and finite")
	// ErrInvalidIterations 表示迭代次数或自适应深度非正
	ErrInvalidIterations = errors.New(_ERROR_PREFIX + "max iterations must be positive")
	// ErrInvalidOrder 表示 Newton-Cotes 或 Gauss-Legendre 的节点数超出支持范围
	ErrInvalidOrder = errors.New(_ERROR_PREFIX + "order must be within the supported range")
	// ErrInvalidPanels 表示分区数非正
	ErrInvalidPanels = errors.New(_ERROR_PREFIX + "panels must be positive")
	// ErrNonFinite 表示计算过程中出现 NaN 或 Inf
	ErrNonFinite = errors.New(_ERROR_PREFIX + "encountered a non-finite value")
	// ErrNotConverged 表示达到迭代上限仍未满足精度要求
	ErrNotConverged = errors.New(_ERROR_PREFIX + "failed to reach the requested accuracy")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
