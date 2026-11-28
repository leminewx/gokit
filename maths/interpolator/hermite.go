package interpolator

// PiecewiseHermite 为分段三次 Hermite 插值器，构造后不可变，可并发求值
type PiecewiseHermite struct {
	xs          []float64
	ys          []float64
	derivatives []float64
}

// NewHermite 由节点值与其一阶导数构造分段三次 Hermite 插值器
//   - xs 严格递增，ys 与 derivatives 必须与 xs 等长；
//   - 每个区段都是三次多项式，整体保证 C¹ 连续；
//   - 不提供外推开关，越界一律返回 ErrOutOfRange。
func NewHermite(xs, ys, derivatives []float64) (*PiecewiseHermite, error) {
	if err := validateSamples(xs, ys); err != nil {
		return nil, err
	}

	if len(derivatives) != len(xs) {
		return nil, wrapError(ErrLengthMismatch, "derivatives=%d, xs=%d", len(derivatives), len(xs))
	}

	for index := range derivatives {
		if !isFinite(derivatives[index]) {
			return nil, wrapError(ErrNonFinite, "derivative %d: %g", index, derivatives[index])
		}
	}

	return &PiecewiseHermite{
		xs:          append([]float64(nil), xs...),
		ys:          append([]float64(nil), ys...),
		derivatives: append([]float64(nil), derivatives...),
	}, nil
}

// Calculate 返回 x 处的插值结果
func (own *PiecewiseHermite) Calculate(x float64) (float64, error) {
	index, err := own.locate(x)
	if err != nil {
		return 0, err
	}

	step := own.xs[index+1] - own.xs[index]
	offset := (x - own.xs[index]) / step
	square := offset * offset
	cube := square * offset

	value := (2*cube-3*square+1)*own.ys[index] +
		(cube-2*square+offset)*step*own.derivatives[index] +
		(-2*cube+3*square)*own.ys[index+1] +
		(cube-square)*step*own.derivatives[index+1]

	return value, nil
}

// CalculateDerivative 返回 x 处的一阶导数
func (own *PiecewiseHermite) CalculateDerivative(x float64) (float64, error) {
	index, err := own.locate(x)
	if err != nil {
		return 0, err
	}

	step := own.xs[index+1] - own.xs[index]
	offset := (x - own.xs[index]) / step
	square := offset * offset

	return (6*square-6*offset)*own.ys[index]/step +
		(3*square-4*offset+1)*own.derivatives[index] +
		(-6*square+6*offset)*own.ys[index+1]/step +
		(3*square-2*offset)*own.derivatives[index+1], nil
}

// locate 校验求值点并返回所在段号
func (own *PiecewiseHermite) locate(x float64) (int, error) {
	if !isFinite(x) {
		return 0, wrapError(ErrNonFinite, "%g", x)
	}

	first, last := own.xs[0], own.xs[len(own.xs)-1]
	if x < first || x > last {
		return 0, wrapError(ErrOutOfRange, "%g, range [%g, %g]", x, first, last)
	}

	return findSegment(own.xs, x), nil
}
