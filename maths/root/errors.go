package root

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.root: "

var (
	// ErrNilFunction 表示未提供目标函数
	ErrNilFunction = errors.New(_ERROR_PREFIX + "function must not be nil")
	// ErrInvalidInterval 表示求根区间非法
	ErrInvalidInterval = errors.New(_ERROR_PREFIX + "interval must satisfy a < b with finite endpoints")
	// ErrInvalidInitial 表示初值非有限
	ErrInvalidInitial = errors.New(_ERROR_PREFIX + "initial value must be finite")
	// ErrDuplicateInitial 表示 Muller 法的三个初值存在重复
	ErrDuplicateInitial = errors.New(_ERROR_PREFIX + "initial values must be pairwise distinct")
	// ErrNoSignChange 表示端点函数值同号或两点函数值相等，无法确定根的区间
	ErrNoSignChange = errors.New(_ERROR_PREFIX + "function values do not bracket a root")
	// ErrInvalidTolerance 表示自变量容差非正或非有限
	ErrInvalidTolerance = errors.New(_ERROR_PREFIX + "tolerance must be positive and finite")
	// ErrInvalidResidual 表示残差容差为负或非有限
	ErrInvalidResidual = errors.New(_ERROR_PREFIX + "residual tolerance must be non-negative and finite")
	// ErrInvalidIterations 表示最大迭代次数非正
	ErrInvalidIterations = errors.New(_ERROR_PREFIX + "max iterations must be positive")
	// ErrZeroDerivative 表示导数或迭代分母已退化
	ErrZeroDerivative = errors.New(_ERROR_PREFIX + "derivative is zero")
	// ErrNonFinite 表示计算过程中出现 NaN 或 Inf
	ErrNonFinite = errors.New(_ERROR_PREFIX + "encountered a non-finite value")
	// ErrNotConverged 表示达到迭代上限仍未满足收敛条件
	ErrNotConverged = errors.New(_ERROR_PREFIX + "failed to converge within the iteration limit")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
