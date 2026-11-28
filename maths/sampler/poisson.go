package sampler

import "math"

// SamplePoisson 从均值为 mean 的泊松分布抽取一个样本
//   - mean 小于 50 时用 Knuth 乘积法，期望消耗 mean+1 个均匀随机数；
//   - mean 不小于 50 时用正态近似后取整并截断为非负数，尾部精度低于精确方法。
func (own *Sampler) SamplePoisson(mean float64) (int, error) {
	if !isFinite(mean) || mean <= 0 {
		return 0, wrapError(ErrInvalidMean, "%g", mean)
	}

	if mean >= _POISSON_EXACT_LIMIT {
		value, err := own.SampleNormal(mean, math.Sqrt(mean))
		if err != nil {
			return 0, err
		}

		sample := int(math.Round(value))
		if sample < 0 {
			sample = 0
		}

		return sample, nil
	}

	threshold := math.Exp(-mean)
	weight := 1.0
	count := 0

	for weight > threshold {
		value, err := own.draw()
		if err != nil {
			return 0, err
		}

		weight *= value
		count++
	}

	return count - 1, nil
}
