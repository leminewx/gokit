package filter

import (
	"math"
	"sort"
)

// CalculateHampel 用 Hampel 准则检测并替换野值
//   - window 必须为不小于 3 的奇数且不超过序列长度，threshold 为判定倍数，通常取 3；
//   - 局部中位数为 m，绝对偏差的中位数为 MAD，判定尺度取 1.4826×MAD（正态分布下与标准差一致）；
//   - |x - m| > threshold×尺度 时用 m 替换 x 并计入 Replaced；
//   - MAD 为 0 时尺度为 0，任何不等于局部中位数的点都会被替换；
//   - 两端不足半窗的点保持原值，窗口统计始终基于原始样本。
func CalculateHampel(values []float64, window int, threshold float64) (Series, error) {
	if err := validateValues(values); err != nil {
		return Series{}, err
	}

	if err := validateOddWindow(values, window); err != nil {
		return Series{}, err
	}

	if !isFinite(threshold) || threshold <= 0 {
		return Series{}, wrapError(ErrInvalidThreshold, "%g", threshold)
	}

	half := window / 2
	buffer := make([]float64, window)
	deviations := make([]float64, window)
	result := append([]float64(nil), values...)
	replaced := 0

	for index := half; index+half < len(values); index++ {
		copy(buffer, values[index-half:index+half+1])
		sort.Float64s(buffer)
		median := buffer[half]

		for offset, value := range buffer {
			deviations[offset] = math.Abs(value - median)
		}

		sort.Float64s(deviations)
		scale := _MAD_SCALE * deviations[half]

		if math.Abs(values[index]-median) > threshold*scale {
			result[index] = median
			replaced++
		}
	}

	return Series{Values: result, Replaced: replaced}, nil
}
