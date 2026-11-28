package differentiator

import "math"

// Sampler 持有等距或非等距的采样序列，可在区间内求一阶、二阶导数
// 构造后不可变，可并发调用。
type Sampler struct {
	xs []float64
	ys []float64
}

// NewSampler 由横纵坐标构造采样器，要求点数不少于 3、横坐标严格递增且全部有限
func NewSampler(xs, ys []float64) (*Sampler, error) {
	if len(xs) != len(ys) {
		return nil, wrapError(ErrLengthMismatch, "%d != %d", len(xs), len(ys))
	}

	if len(xs) < _MIN_SAMPLE {
		return nil, wrapError(ErrTooFewSamples, "%d < %d", len(xs), _MIN_SAMPLE)
	}

	for index := range xs {
		if !isFinite(xs[index]) || !isFinite(ys[index]) {
			return nil, wrapError(ErrNonFinite, "sample[%d]", index)
		}

		if index > 0 && xs[index] <= xs[index-1] {
			return nil, wrapError(ErrNotStrictlyIncreasing, "%g then %g", xs[index-1], xs[index])
		}
	}

	return &Sampler{
		xs: append([]float64(nil), xs...),
		ys: append([]float64(nil), ys...),
	}, nil
}

// CalculateDerivative 计算求值点处的一阶导数
func (own *Sampler) CalculateDerivative(x float64) (Derivative, error) {
	return own.CalculateDerivativeOrder(x, 1)
}

// CalculateDerivativeOrder 计算求值点处的指定阶导数，order 仅支持 1 与 2
// 两侧模板都可用时取平均，并以两者之差的一半作为误差指示。
func (own *Sampler) CalculateDerivativeOrder(x float64, order int) (Derivative, error) {
	if order != 1 && order != 2 {
		return Derivative{}, wrapError(ErrInvalidOrder, "%d, want 1 or 2", order)
	}

	if !isFinite(x) {
		return Derivative{}, wrapError(ErrNonFinite, "evaluation point %g", x)
	}

	last := len(own.xs) - 1
	if x < own.xs[0] || x > own.xs[last] {
		return Derivative{}, wrapError(ErrOutOfRange, "%g over [%g, %g]", x, own.xs[0], own.xs[last])
	}

	segment := findSegment(own.xs, x)
	hasLeft := segment >= 1
	hasRight := segment+2 <= last

	left, leftErr := 0.0, error(nil)
	if hasLeft {
		left, leftErr = own.calculateTriplet(segment-1, x, order)
		if leftErr != nil {
			return Derivative{}, leftErr
		}
	}

	right, rightErr := 0.0, error(nil)
	if hasRight {
		right, rightErr = own.calculateTriplet(segment, x, order)
		if rightErr != nil {
			return Derivative{}, rightErr
		}
	}

	switch {
	case hasLeft && hasRight:
		return Derivative{
			Value:         (left + right) / 2,
			ErrorEstimate: math.Abs(right-left) / 2,
			Window:        3,
		}, nil
	case hasRight:
		return Derivative{Value: right, Window: 3}, nil
	default:
		return Derivative{Value: left, Window: 3}, nil
	}
}

// calculateTriplet 用三个连续样本构造二次插值多项式，在 x 处求指定阶导数
func (own *Sampler) calculateTriplet(start int, x float64, order int) (float64, error) {
	x0, x1, x2 := own.xs[start], own.xs[start+1], own.xs[start+2]
	y0, y1, y2 := own.ys[start], own.ys[start+1], own.ys[start+2]

	if order == 1 {
		value := y0*(2*x-x1-x2)/((x0-x1)*(x0-x2)) +
			y1*(2*x-x0-x2)/((x1-x0)*(x1-x2)) +
			y2*(2*x-x0-x1)/((x2-x0)*(x2-x1))

		if !isFinite(value) {
			return 0, wrapError(ErrNonFinite, "derivative at %g", x)
		}

		return value, nil
	}

	value := 2*y0/((x0-x1)*(x0-x2)) +
		2*y1/((x1-x0)*(x1-x2)) +
		2*y2/((x2-x0)*(x2-x1))

	if !isFinite(value) {
		return 0, wrapError(ErrNonFinite, "second derivative at %g", x)
	}

	return value, nil
}

// findSegment 返回满足 xs[index] <= x <= xs[index+1] 的区间下标
func findSegment(xs []float64, x float64) int {
	low, high := 0, len(xs)-1
	for high-low > 1 {
		middle := (low + high) / 2
		if x < xs[middle] {
			high = middle
		} else {
			low = middle
		}
	}

	return low
}
