package distribution

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.distribution: "

var (
	// ErrInvalidStdDev 表示标准差非正或非有限
	ErrInvalidStdDev = errors.New(_ERROR_PREFIX + "standard deviation must be positive and finite")
	// ErrInvalidRange 表示区间端点非有限或未满足 lower < upper
	ErrInvalidRange = errors.New(_ERROR_PREFIX + "range must satisfy lower < upper with finite endpoints")
	// ErrInvalidRate 表示指数分布的速率非正或非有限
	ErrInvalidRate = errors.New(_ERROR_PREFIX + "rate must be positive and finite")
	// ErrInvalidMean 表示泊松分布的均值非正或非有限
	ErrInvalidMean = errors.New(_ERROR_PREFIX + "mean must be positive and finite")
	// ErrInvalidProbability 表示概率不在 (0, 1) 内
	ErrInvalidProbability = errors.New(_ERROR_PREFIX + "probability must be within (0, 1)")
	// ErrNotConverged 表示分位数搜索在项数上限内未达到目标概率
	ErrNotConverged = errors.New(_ERROR_PREFIX + "failed to reach the requested probability within the term limit")
	// ErrNonFinite 表示参数或求值点出现 NaN / Inf
	ErrNonFinite = errors.New(_ERROR_PREFIX + "encountered a non-finite value")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
