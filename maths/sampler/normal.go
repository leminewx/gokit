package sampler

import "math"

// SampleNormal 从均值为 mean、标准差为 stdDev 的正态分布抽取一个样本
// 采用 Box-Muller 变换，每次消耗两个均匀随机数。
func (own *Sampler) SampleNormal(mean, stdDev float64) (float64, error) {
	if !isFinite(mean) {
		return 0, wrapError(ErrNonFinite, "mean %g", mean)
	}

	if !isFinite(stdDev) || stdDev <= 0 {
		return 0, wrapError(ErrInvalidStdDev, "%g", stdDev)
	}

	first, err := own.draw()
	if err != nil {
		return 0, err
	}

	second, err := own.draw()
	if err != nil {
		return 0, err
	}

	// 1 - first 落在 (0, 1]，避免对数在 0 处发散
	radius := math.Sqrt(-2 * math.Log(1-first))
	return mean + stdDev*radius*math.Cos(2*math.Pi*second), nil
}
