package activator

import "math"

// CalculateSoftmax 对多维度输入归一化为概率分布
//   - 输入为空时返回空切片
//   - 减去最大值以避免指数溢出
func CalculateSoftmax(x []float64) []float64 {
	if len(x) == 0 {
		return nil
	}

	maxVal := x[0]
	for _, val := range x[1:] {
		if val > maxVal {
			maxVal = val
		}
	}

	expVals := make([]float64, len(x))
	expSum := 0.0
	for i, val := range x {
		expVals[i] = math.Exp(val - maxVal)
		expSum += expVals[i]
	}

	// 输入全为 -Inf/NaN 时指数和会退化为 0，此时返回均匀分布
	if expSum == 0 || math.IsNaN(expSum) {
		uniform := 1.0 / float64(len(x))
		for i := range expVals {
			expVals[i] = uniform
		}

		return expVals
	}

	for i := range expVals {
		expVals[i] /= expSum
	}

	return expVals
}

// CalculateSoftmaxBackward 结合交叉熵损失计算梯度
//   - label 越界时返回全零梯度
func CalculateSoftmaxBackward(probes []float64, label int) []float64 {
	grad := make([]float64, len(probes))
	if label < 0 || label >= len(probes) {
		return grad
	}

	for i := range grad {
		if i == label {
			grad[i] = probes[i] - 1
			continue
		}

		grad[i] = probes[i]
	}

	return grad
}
