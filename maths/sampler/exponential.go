package sampler

import "math"

// SampleExponential 从速率为 rate 的指数分布抽取一个样本，均值为 1/rate
// 采用逆变换法，每次消耗一个均匀随机数。
func (own *Sampler) SampleExponential(rate float64) (float64, error) {
	if !isFinite(rate) || rate <= 0 {
		return 0, wrapError(ErrInvalidRate, "%g", rate)
	}

	value, err := own.draw()
	if err != nil {
		return 0, err
	}

	return -math.Log(1-value) / rate, nil
}
