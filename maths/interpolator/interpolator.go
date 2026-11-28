// Package interpolator 提供一维采样数据的插值方法
//   - Lagrange 与 Newton：整体多项式插值，返回可求值与求导的 Polynomial；
//   - 三次样条：分段三次多项式，C² 连续，支持自然 / 夹持 / 周期 / 非扭结四种边界；
//   - 分段三次 Hermite：给定节点值与一阶导数，逐段构造三次多项式。
//
// 设计目标：
//   - 零第三方依赖，三对角与循环三对角方程组自带追赶法求解；
//   - 统一返回 (值, error)，越界行为显式可选，绝不静默外推；
//   - 输入校验前置：长度、严格递增、有限性在构造阶段一次性检查；
//   - 构造后不可变，所有求值方法可并发调用。
//
// 越界求值约定：
//   - 三次样条默认拒绝外推（ErrOutOfRange），可显式选择线性 / 三次 / 常值 / 周期环绕策略；
//   - Hermite 与多项式插值不提供外推开关，越界一律返回 ErrOutOfRange
//     （多项式的定义域天然是全体实数，此处仅对样条与 Hermite 检查区间）。
//
// 典型用法：
//
//	spline, err := interpolator.NewCubicSpline(xs, ys, interpolator.BOUNDARY_NATURAL)
//	if err == nil {
//	    value, err := spline.Calculate(1.5)
//	}
//
// 已知限制：
//   - 多项式插值对高次情形存在 Runge 现象，采样点超过 64 个时直接拒绝；
//   - 样条要求横坐标严格递增，重复横坐标需调用方先聚合；
//   - 周期边界要求首末函数值一致且至少 4 个采样点，否则返回 ErrInvalidBoundary。
package interpolator

import "math"

// BoundaryCondition 定义三次样条在两端使用的边界条件
type BoundaryCondition uint8

const (
	// BOUNDARY_NATURAL 自然边界：两端二阶导数为 0
	BOUNDARY_NATURAL BoundaryCondition = iota
	// BOUNDARY_CLAMPED 夹持边界：给定两端一阶导数，需配合 WithClampedDerivatives
	BOUNDARY_CLAMPED
	// BOUNDARY_PERIODIC 周期边界：要求首末函数值相等，越界可选周期环绕
	BOUNDARY_PERIODIC
	// BOUNDARY_NOT_A_KNOT 非扭结边界：首末两个内部节点的三阶导数连续，至少需要 4 个采样点
	BOUNDARY_NOT_A_KNOT
)

// Extrapolation 定义三次样条在样本区间之外的求值策略
type Extrapolation uint8

const (
	// EXTRAPOLATION_REJECT 越界返回 ErrOutOfRange（默认）
	EXTRAPOLATION_REJECT Extrapolation = iota
	// EXTRAPOLATION_LINEAR 沿端点切线线性外推
	EXTRAPOLATION_LINEAR
	// EXTRAPOLATION_CUBIC 沿用端段三次多项式外推
	EXTRAPOLATION_CUBIC
	// EXTRAPOLATION_CLAMP 常值延拓，返回端点值、导数 0
	EXTRAPOLATION_CLAMP
	// EXTRAPOLATION_WRAP 按周期环绕，仅 BOUNDARY_PERIODIC 可用
	EXTRAPOLATION_WRAP
)

const (
	// _MIN_SAMPLE 为插值所需的最少采样点数
	_MIN_SAMPLE = 2
	// _NOT_A_KNOT_MIN_SAMPLE 为非扭结边界所需的最少采样点数
	_NOT_A_KNOT_MIN_SAMPLE = 4
	// _PERIODIC_MIN_SAMPLE 为周期边界所需的最少采样点数（含重复的周期端点）
	_PERIODIC_MIN_SAMPLE = 4
	// _MAX_POLYNOMIAL_SAMPLE 为整体多项式插值的采样点上限
	_MAX_POLYNOMIAL_SAMPLE = 64
	// _PERIODIC_TOLERANCE 为周期边界首末函数值的一致性容差
	_PERIODIC_TOLERANCE = 1e-10
	// _PIVOT_TOLERANCE 为三对角求解的主元下界
	_PIVOT_TOLERANCE = 1e-14
)

// Option 配置插值对象的选项
type Option func(*config)

type config struct {
	policy             Extrapolation
	leftDerivative     float64
	rightDerivative    float64
	hasClampedBoundary bool
}

// WithExtrapolation 设置三次样条的越界求值策略，默认 EXTRAPOLATION_REJECT
func WithExtrapolation(policy Extrapolation) Option {
	return func(own *config) { own.policy = policy }
}

// WithClampedDerivatives 为 BOUNDARY_CLAMPED 提供两端一阶导数
func WithClampedDerivatives(left, right float64) Option {
	return func(own *config) {
		own.leftDerivative = left
		own.rightDerivative = right
		own.hasClampedBoundary = true
	}
}

// newConfig 应用选项并填充默认值
func newConfig(options []Option) *config {
	own := &config{policy: EXTRAPOLATION_REJECT}

	for _, option := range options {
		option(own)
	}

	return own
}

// validateSamples 校验采样数据：长度一致、至少两点、横坐标严格递增、全部有限
func validateSamples(xs, ys []float64) error {
	if len(xs) != len(ys) {
		return wrapError(ErrLengthMismatch, "xs=%d, ys=%d", len(xs), len(ys))
	}

	if len(xs) < _MIN_SAMPLE {
		return wrapError(ErrTooFewSamples, "%d, want >= %d", len(xs), _MIN_SAMPLE)
	}

	for index := range xs {
		if !isFinite(xs[index]) || !isFinite(ys[index]) {
			return wrapError(ErrNonFinite, "sample %d: (%g, %g)", index, xs[index], ys[index])
		}

		if index > 0 && xs[index] <= xs[index-1] {
			return wrapError(ErrNotStrictlyIncreasing, "xs[%d]=%g, xs[%d]=%g", index-1, xs[index-1], index, xs[index])
		}
	}

	return nil
}

// findSegment 返回满足 xs[index] <= x < xs[index+1] 的段号，末段索引不超过 len(xs)-2
func findSegment(xs []float64, x float64) int {
	low, high := 0, len(xs)-2
	for low < high {
		middle := (low + high + 1) / 2
		if xs[middle] <= x {
			low = middle
		} else {
			high = middle - 1
		}
	}

	return low
}

// isFinite 判断浮点数是否为有限值
func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
