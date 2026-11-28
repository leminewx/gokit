package distribution

import "math"

// Exponential 指数分布，以速率 λ 参数化
// 构造后不可变，可并发调用。
type Exponential struct {
	Rate float64 // 速率 λ，必须为正，均值为 1/λ
}

// NewExponential 构造指数分布，要求速率有限且为正
func NewExponential(rate float64) (*Exponential, error) {
	if !isFinite(rate) || rate <= 0 {
		return nil, wrapError(ErrInvalidRate, "%g", rate)
	}

	return &Exponential{Rate: rate}, nil
}

// GetMean 返回均值 1/λ
func (own *Exponential) GetMean() float64 {
	return 1 / own.Rate
}

// GetVariance 返回方差 1/λ²
func (own *Exponential) GetVariance() float64 {
	return 1 / (own.Rate * own.Rate)
}

// CalculatePDF 返回概率密度函数值，负半轴为 0
func (own *Exponential) CalculatePDF(x float64) (float64, error) {
	if !isFinite(x) {
		return 0, wrapError(ErrNonFinite, "evaluation point %g", x)
	}

	if x < 0 {
		return 0, nil
	}

	return own.Rate * math.Exp(-own.Rate*x), nil
}

// CalculateCDF 返回累积分布函数值 P(X <= x)，负半轴为 0
func (own *Exponential) CalculateCDF(x float64) (float64, error) {
	if !isFinite(x) {
		return 0, wrapError(ErrNonFinite, "evaluation point %g", x)
	}

	if x <= 0 {
		return 0, nil
	}

	return -math.Expm1(-own.Rate * x), nil
}

// CalculateQuantile 返回满足 P(X <= x) = probability 的分位数 x
// probability 必须位于 (0, 1)。
func (own *Exponential) CalculateQuantile(probability float64) (float64, error) {
	if err := validateProbability(probability); err != nil {
		return 0, err
	}

	return -math.Log1p(-probability) / own.Rate, nil
}
