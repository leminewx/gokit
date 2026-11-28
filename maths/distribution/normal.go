package distribution

import "math"

// Normal 正态分布
// 构造后不可变，可并发调用。
type Normal struct {
	Mean   float64 // 均值 μ
	StdDev float64 // 标准差 σ，必须为正
}

// NewNormal 构造正态分布，要求均值有限且标准差为正
func NewNormal(mean, stdDev float64) (*Normal, error) {
	if !isFinite(mean) {
		return nil, wrapError(ErrNonFinite, "mean %g", mean)
	}

	if !isFinite(stdDev) || stdDev <= 0 {
		return nil, wrapError(ErrInvalidStdDev, "%g", stdDev)
	}

	return &Normal{Mean: mean, StdDev: stdDev}, nil
}

// GetMean 返回均值
func (own *Normal) GetMean() float64 {
	return own.Mean
}

// GetVariance 返回方差
func (own *Normal) GetVariance() float64 {
	return own.StdDev * own.StdDev
}

// CalculatePDF 返回概率密度函数值
func (own *Normal) CalculatePDF(x float64) (float64, error) {
	if !isFinite(x) {
		return 0, wrapError(ErrNonFinite, "evaluation point %g", x)
	}

	standardized := (x - own.Mean) / own.StdDev
	return math.Exp(-0.5*standardized*standardized) / (own.StdDev * _SQRT_TWO_PI), nil
}

// CalculateCDF 返回累积分布函数值 P(X <= x)
func (own *Normal) CalculateCDF(x float64) (float64, error) {
	if !isFinite(x) {
		return 0, wrapError(ErrNonFinite, "evaluation point %g", x)
	}

	standardized := (x - own.Mean) / (own.StdDev * math.Sqrt2)
	return 0.5 * (1 + math.Erf(standardized)), nil
}

// CalculateQuantile 返回满足 P(X <= x) = probability 的分位数 x
// probability 必须位于 (0, 1)，极端概率下结果可能超出 float64 表示范围。
func (own *Normal) CalculateQuantile(probability float64) (float64, error) {
	if err := validateProbability(probability); err != nil {
		return 0, err
	}

	value := own.Mean + own.StdDev*math.Sqrt2*math.Erfinv(2*probability-1)
	if !isFinite(value) {
		return 0, wrapError(ErrNonFinite, "quantile at probability %g", probability)
	}

	return value, nil
}
