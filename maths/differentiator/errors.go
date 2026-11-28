package differentiator

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.differentiator: "

var (
	// ErrNilFunction 表示未提供目标函数
	ErrNilFunction = errors.New(_ERROR_PREFIX + "function must not be nil")
	// ErrEmptyInput 表示采样序列为空
	ErrEmptyInput = errors.New(_ERROR_PREFIX + "values must not be empty")
	// ErrLengthMismatch 表示横纵坐标长度不一致
	ErrLengthMismatch = errors.New(_ERROR_PREFIX + "abscissas and ordinates must have the same length")
	// ErrTooFewSamples 表示样本数不足以构造模板
	ErrTooFewSamples = errors.New(_ERROR_PREFIX + "not enough samples")
	// ErrNotStrictlyIncreasing 表示横坐标未严格递增
	ErrNotStrictlyIncreasing = errors.New(_ERROR_PREFIX + "abscissas must be strictly increasing")
	// ErrInvalidOrder 表示导数阶数或多项式阶数不受支持
	ErrInvalidOrder = errors.New(_ERROR_PREFIX + "order is not supported")
	// ErrInvalidWindow 表示窗口长度为偶数或超出支持范围
	ErrInvalidWindow = errors.New(_ERROR_PREFIX + "window must be an odd number within the supported range")
	// ErrOutOfRange 表示求值点超出采样区间
	ErrOutOfRange = errors.New(_ERROR_PREFIX + "evaluation point is outside the sampled range")
	// ErrNonFinite 表示输入或中间结果出现 NaN / Inf
	ErrNonFinite = errors.New(_ERROR_PREFIX + "encountered a non-finite value")
	// ErrSingularSystem 表示局部多项式拟合的法方程奇异
	ErrSingularSystem = errors.New(_ERROR_PREFIX + "normal equations are singular")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
