package distribution

// Uniform 连续均匀分布
// 构造后不可变，可并发调用。
type Uniform struct {
	Lower float64 // 下界
	Upper float64 // 上界，必须大于 Lower
}

// NewUniform 构造均匀分布，要求两端点有限且满足 lower < upper
func NewUniform(lower, upper float64) (*Uniform, error) {
	if !isFinite(lower) || !isFinite(upper) || lower >= upper {
		return nil, wrapError(ErrInvalidRange, "[%g, %g]", lower, upper)
	}

	return &Uniform{Lower: lower, Upper: upper}, nil
}

// GetMean 返回均值
func (own *Uniform) GetMean() float64 {
	return 0.5 * (own.Lower + own.Upper)
}

// GetVariance 返回方差
func (own *Uniform) GetVariance() float64 {
	width := own.Upper - own.Lower
	return width * width / 12
}

// CalculatePDF 返回概率密度函数值，区间外为 0
func (own *Uniform) CalculatePDF(x float64) (float64, error) {
	if !isFinite(x) {
		return 0, wrapError(ErrNonFinite, "evaluation point %g", x)
	}

	if x < own.Lower || x > own.Upper {
		return 0, nil
	}

	return 1 / (own.Upper - own.Lower), nil
}

// CalculateCDF 返回累积分布函数值 P(X <= x)
func (own *Uniform) CalculateCDF(x float64) (float64, error) {
	if !isFinite(x) {
		return 0, wrapError(ErrNonFinite, "evaluation point %g", x)
	}

	switch {
	case x <= own.Lower:
		return 0, nil
	case x >= own.Upper:
		return 1, nil
	default:
		return (x - own.Lower) / (own.Upper - own.Lower), nil
	}
}

// CalculateQuantile 返回满足 P(X <= x) = probability 的分位数 x
// probability 必须位于 (0, 1)。
func (own *Uniform) CalculateQuantile(probability float64) (float64, error) {
	if err := validateProbability(probability); err != nil {
		return 0, err
	}

	return own.Lower + probability*(own.Upper-own.Lower), nil
}
