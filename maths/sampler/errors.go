package sampler

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.sampler: "

var (
	// ErrNilSource 表示未提供随机源
	ErrNilSource = errors.New(_ERROR_PREFIX + "source must not be nil")
	// ErrNilRandom 表示未提供伪随机数生成器
	ErrNilRandom = errors.New(_ERROR_PREFIX + "random generator must not be nil")
	// ErrInvalidSource 表示随机源返回了 [0, 1) 之外的值
	ErrInvalidSource = errors.New(_ERROR_PREFIX + "source returned a value outside [0, 1)")
	// ErrInvalidRange 表示区间端点非有限或未满足 lower < upper
	ErrInvalidRange = errors.New(_ERROR_PREFIX + "range must satisfy lower < upper with finite endpoints")
	// ErrInvalidStdDev 表示标准差非正或非有限
	ErrInvalidStdDev = errors.New(_ERROR_PREFIX + "standard deviation must be positive and finite")
	// ErrInvalidRate 表示指数分布的速率非正或非有限
	ErrInvalidRate = errors.New(_ERROR_PREFIX + "rate must be positive and finite")
	// ErrInvalidMean 表示泊松分布的均值非正或非有限
	ErrInvalidMean = errors.New(_ERROR_PREFIX + "mean must be positive and finite")
	// ErrInvalidCount 表示请求的样本数为非正
	ErrInvalidCount = errors.New(_ERROR_PREFIX + "count must be positive")
	// ErrNonFinite 表示输入参数为 NaN 或无穷
	ErrNonFinite = errors.New(_ERROR_PREFIX + "input must be finite")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
