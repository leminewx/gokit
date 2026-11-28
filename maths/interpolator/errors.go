package interpolator

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.interpolator: "

var (
	// ErrTooFewSamples 表示采样点少于该方法要求的下限
	ErrTooFewSamples = errors.New(_ERROR_PREFIX + "not enough samples")
	// ErrLengthMismatch 表示横纵坐标长度不一致
	ErrLengthMismatch = errors.New(_ERROR_PREFIX + "xs and ys must have the same length")
	// ErrNotStrictlyIncreasing 表示横坐标未严格递增
	ErrNotStrictlyIncreasing = errors.New(_ERROR_PREFIX + "abscissas must be strictly increasing")
	// ErrOutOfRange 表示求值点落在样本区间之外且未启用外推
	ErrOutOfRange = errors.New(_ERROR_PREFIX + "evaluation point is outside the sample range")
	// ErrInvalidBoundary 表示边界条件与数据不匹配
	ErrInvalidBoundary = errors.New(_ERROR_PREFIX + "boundary condition is incompatible with the samples")
	// ErrMissingDerivative 表示夹持边界缺少端点导数
	ErrMissingDerivative = errors.New(_ERROR_PREFIX + "clamped boundary requires endpoint derivatives")
	// ErrNonFinite 表示输入或中间结果出现 NaN 或 Inf
	ErrNonFinite = errors.New(_ERROR_PREFIX + "encountered a non-finite value")
	// ErrTooManySamples 表示构造多项式的采样点过多，展开系数代价过高
	ErrTooManySamples = errors.New(_ERROR_PREFIX + "too many samples for polynomial construction")
	// ErrSingularSystem 表示样条的三对角方程组奇异
	ErrSingularSystem = errors.New(_ERROR_PREFIX + "spline system is singular")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
