package interpolator

import "math"

// CubicSpline 为三次样条插值器，构造后不可变，可并发求值
type CubicSpline struct {
	xs         []float64 // 采样横坐标，严格递增
	ys         []float64 // 采样纵坐标
	moments    []float64 // 各节点的二阶导数
	policy     Extrapolation
	leftSlope  float64 // 左端点一阶导数，供线性外推使用
	rightSlope float64 // 右端点一阶导数，供线性外推使用
}

// NewCubicSpline 由采样点与边界条件构造三次样条
//   - xs 必须严格递增且与 ys 等长，至少 2 个采样点；
//   - BOUNDARY_CLAMPED 需配合 WithClampedDerivatives，否则返回 ErrMissingDerivative；
//   - BOUNDARY_PERIODIC 要求首末函数值一致且至少 4 个采样点（含重复的周期端点）；
//   - BOUNDARY_NOT_A_KNOT 至少需要 4 个采样点；
//   - 越界策略默认 EXTRAPOLATION_REJECT，EXTRAPOLATION_WRAP 仅周期边界可用。
func NewCubicSpline(xs, ys []float64, condition BoundaryCondition, options ...Option) (*CubicSpline, error) {
	if err := validateSamples(xs, ys); err != nil {
		return nil, err
	}

	own := newConfig(options)
	if err := validateBoundary(condition, xs, ys, own); err != nil {
		return nil, err
	}

	moments, err := calculateMoments(xs, ys, condition, own)
	if err != nil {
		return nil, err
	}

	spline := &CubicSpline{
		xs:      append([]float64(nil), xs...),
		ys:      append([]float64(nil), ys...),
		moments: moments,
		policy:  own.policy,
	}

	spline.leftSlope = spline.evaluateDerivative(spline.xs[0])
	spline.rightSlope = spline.evaluateDerivative(spline.xs[len(spline.xs)-1])

	return spline, nil
}

// Calculate 返回 x 处的插值结果
//   - 落在 [xs[0], xs[n-1]] 内按段求值；
//   - 越界行为由构造时的 Extrapolation 策略决定，默认返回 ErrOutOfRange。
func (own *CubicSpline) Calculate(x float64) (float64, error) {
	if !isFinite(x) {
		return 0, wrapError(ErrNonFinite, "%g", x)
	}

	if own.isOutside(x) {
		return own.calculateOutside(x)
	}

	return own.evaluate(x), nil
}

// CalculateDerivative 返回 x 处的一阶导数
//   - 越界时：线性外推返回端点斜率，常值延拓返回 0，三次外推沿用端段导数；
//   - 拒绝外推时返回 ErrOutOfRange。
func (own *CubicSpline) CalculateDerivative(x float64) (float64, error) {
	if !isFinite(x) {
		return 0, wrapError(ErrNonFinite, "%g", x)
	}

	if !own.isOutside(x) {
		return own.evaluateDerivative(x), nil
	}

	first, last := own.xs[0], own.xs[len(own.xs)-1]
	switch own.policy {
	case EXTRAPOLATION_LINEAR:
		if x < first {
			return own.leftSlope, nil
		}

		return own.rightSlope, nil
	case EXTRAPOLATION_CUBIC:
		return own.evaluateDerivative(x), nil
	case EXTRAPOLATION_CLAMP:
		return 0, nil
	case EXTRAPOLATION_WRAP:
		wrapped, err := own.wrap(x)
		if err != nil {
			return 0, err
		}

		return own.evaluateDerivative(wrapped), nil
	default:
		return 0, wrapError(ErrOutOfRange, "%g, range [%g, %g]", x, first, last)
	}
}

// CalculateSecondDerivative 返回 x 处的二阶导数
//   - 越界时：线性外推与常值延拓返回 0，三次外推与周期环绕沿用端段二阶导数。
func (own *CubicSpline) CalculateSecondDerivative(x float64) (float64, error) {
	if !isFinite(x) {
		return 0, wrapError(ErrNonFinite, "%g", x)
	}

	if !own.isOutside(x) {
		return own.evaluateSecondDerivative(x), nil
	}

	first, last := own.xs[0], own.xs[len(own.xs)-1]
	switch own.policy {
	case EXTRAPOLATION_LINEAR, EXTRAPOLATION_CLAMP:
		return 0, nil
	case EXTRAPOLATION_CUBIC:
		return own.evaluateSecondDerivative(x), nil
	case EXTRAPOLATION_WRAP:
		wrapped, err := own.wrap(x)
		if err != nil {
			return 0, err
		}

		return own.evaluateSecondDerivative(wrapped), nil
	default:
		return 0, wrapError(ErrOutOfRange, "%g, range [%g, %g]", x, first, last)
	}
}

// isOutside 判断 x 是否落在样本区间之外
func (own *CubicSpline) isOutside(x float64) bool {
	return x < own.xs[0] || x > own.xs[len(own.xs)-1]
}

// calculateOutside 按越界策略返回插值结果
func (own *CubicSpline) calculateOutside(x float64) (float64, error) {
	first, last := own.xs[0], own.xs[len(own.xs)-1]
	right := len(own.ys) - 1

	switch own.policy {
	case EXTRAPOLATION_LINEAR:
		if x < first {
			return own.ys[0] + own.leftSlope*(x-first), nil
		}

		return own.ys[right] + own.rightSlope*(x-last), nil
	case EXTRAPOLATION_CUBIC:
		return own.evaluate(x), nil
	case EXTRAPOLATION_CLAMP:
		if x < first {
			return own.ys[0], nil
		}

		return own.ys[right], nil
	case EXTRAPOLATION_WRAP:
		wrapped, err := own.wrap(x)
		if err != nil {
			return 0, err
		}

		return own.evaluate(wrapped), nil
	default:
		return 0, wrapError(ErrOutOfRange, "%g, range [%g, %g]", x, first, last)
	}
}

// wrap 把区间外的点按周期映射回样本区间
func (own *CubicSpline) wrap(x float64) (float64, error) {
	first := own.xs[0]
	period := own.xs[len(own.xs)-1] - first
	if period <= 0 {
		return 0, wrapError(ErrInvalidBoundary, "period %g", period)
	}

	position := math.Mod(x-first, period)
	if position < 0 {
		position += period
	}

	return first + position, nil
}

// evaluate 在样本区间内按段求值
func (own *CubicSpline) evaluate(x float64) float64 {
	index := findSegment(own.xs, x)
	offset := x - own.xs[index]
	step := own.xs[index+1] - own.xs[index]
	left, right := own.moments[index], own.moments[index+1]
	slope := (own.ys[index+1]-own.ys[index])/step - step*(2*left+right)/6

	return own.ys[index] + offset*(slope+offset*(left/2+offset*(right-left)/(6*step)))
}

// evaluateDerivative 在样本区间内按段求一阶导数
func (own *CubicSpline) evaluateDerivative(x float64) float64 {
	index := findSegment(own.xs, x)
	offset := x - own.xs[index]
	step := own.xs[index+1] - own.xs[index]
	left, right := own.moments[index], own.moments[index+1]
	slope := (own.ys[index+1]-own.ys[index])/step - step*(2*left+right)/6

	return slope + offset*(left+offset*(right-left)/(2*step))
}

// evaluateSecondDerivative 在样本区间内按段求二阶导数
func (own *CubicSpline) evaluateSecondDerivative(x float64) float64 {
	index := findSegment(own.xs, x)
	offset := x - own.xs[index]
	step := own.xs[index+1] - own.xs[index]

	return own.moments[index] + offset*(own.moments[index+1]-own.moments[index])/step
}
