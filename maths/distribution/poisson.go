package distribution

import "math"

// Poisson 泊松分布，以均值 λ 参数化
// 构造后不可变，可并发调用。
type Poisson struct {
	Mean float64 // 均值 λ，必须为正，方差同为 λ
}

// NewPoisson 构造泊松分布，要求均值有限且为正
func NewPoisson(mean float64) (*Poisson, error) {
	if !isFinite(mean) || mean <= 0 {
		return nil, wrapError(ErrInvalidMean, "%g", mean)
	}

	return &Poisson{Mean: mean}, nil
}

// GetMean 返回均值
func (own *Poisson) GetMean() float64 {
	return own.Mean
}

// GetVariance 返回方差，泊松分布的方差与均值相等
func (own *Poisson) GetVariance() float64 {
	return own.Mean
}

// CalculatePMF 返回概率质量函数值 P(X = k)，k 为负时返回 0
func (own *Poisson) CalculatePMF(k int) float64 {
	if k < 0 {
		return 0
	}

	// 在对数空间计算，避免阶乘与幂次溢出
	logarithm, _ := math.Lgamma(float64(k) + 1)
	return math.Exp(-own.Mean + float64(k)*math.Log(own.Mean) - logarithm)
}

// CalculateCDF 返回累积分布函数值 P(X <= k)，k 为负时返回 0
// 当 k 超过均值 40 个标准差以上时直接返回 1，尾部概率低于 1e-14。
func (own *Poisson) CalculateCDF(k int) float64 {
	if k < 0 {
		return 0
	}

	boundary := own.Mean + _POISSON_TAIL*math.Sqrt(own.Mean) + _POISSON_TAIL
	if float64(k) >= boundary {
		return 1
	}

	total := 0.0
	for index := 0; index <= k; index++ {
		total += own.CalculatePMF(index)
	}

	return math.Min(1, total)
}

// CalculateQuantile 返回满足 P(X <= k) >= probability 的最小非负整数 k
// probability 必须位于 (0, 1)，项数超过上限时返回 ErrNotConverged。
func (own *Poisson) CalculateQuantile(probability float64) (int, error) {
	if err := validateProbability(probability); err != nil {
		return 0, err
	}

	limit := int(own.Mean + _POISSON_TAIL*math.Sqrt(own.Mean) + _POISSON_TAIL)
	if limit > _MAX_POISSON_ITERATION {
		limit = _MAX_POISSON_ITERATION
	}

	cumulative := 0.0
	for index := 0; index <= limit; index++ {
		cumulative += own.CalculatePMF(index)
		if cumulative >= probability {
			return index, nil
		}
	}

	return 0, wrapError(ErrNotConverged, "probability %g over %d terms", probability, limit)
}
