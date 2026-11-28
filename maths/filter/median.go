package filter

import "sort"

// CalculateMedian 对序列做中值滤波
//   - window 必须为不小于 3 的奇数且不超过序列长度；
//   - 两端不足半窗的点保持原值，避免窗口偏移造成边界失真；
//   - 单个尖峰最多影响半窗范围，阶跃边缘不会被抹平。
func CalculateMedian(values []float64, window int) (Series, error) {
	if err := validateValues(values); err != nil {
		return Series{}, err
	}

	if err := validateOddWindow(values, window); err != nil {
		return Series{}, err
	}

	half := window / 2
	buffer := make([]float64, window)
	result := append([]float64(nil), values...)

	for index := half; index+half < len(values); index++ {
		copy(buffer, values[index-half:index+half+1])
		sort.Float64s(buffer)
		result[index] = buffer[half]
	}

	return Series{Values: result}, nil
}
