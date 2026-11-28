package optimizer

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.optimizer: "

var (
	// ErrNilFunction 表示未提供目标函数
	ErrNilFunction = errors.New(_ERROR_PREFIX + "function must not be nil")
	// ErrEmptyInitial 表示未提供初始点
	ErrEmptyInitial = errors.New(_ERROR_PREFIX + "initial point must not be empty")
	// ErrInvalidRange 表示搜索区间端点非有限或未满足 lower < upper
	ErrInvalidRange = errors.New(_ERROR_PREFIX + "range must satisfy lower < upper with finite endpoints")
	// ErrInvalidTolerance 表示容差非正或非有限
	ErrInvalidTolerance = errors.New(_ERROR_PREFIX + "tolerance must be positive and finite")
	// ErrInvalidIterations 表示最大迭代次数非正
	ErrInvalidIterations = errors.New(_ERROR_PREFIX + "max iterations must be positive")
	// ErrNotConverged 表示在迭代上限内未满足收敛条件
	ErrNotConverged = errors.New(_ERROR_PREFIX + "failed to converge within the iteration limit")
	// ErrNonFinite 表示初值、目标函数值或中间结果出现 NaN / Inf
	ErrNonFinite = errors.New(_ERROR_PREFIX + "encountered a non-finite value")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
