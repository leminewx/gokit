package filter

// CalculateMovingAverage 对序列做滑动平均，窗口内每个样本权重相同
//   - window 满足 1 <= window <= len(values)；
//   - 两端不足半窗时按实际可用样本收缩窗口，输出长度与输入一致；
//   - 窗口越大平滑越强，但阶跃与峰值也会被更大范围地摊平。
func CalculateMovingAverage(values []float64, window int) (Series, error) {
	if err := validateValues(values); err != nil {
		return Series{}, err
	}

	if window < 1 || window > len(values) {
		return Series{}, wrapError(ErrInvalidWindow, "%d over %d", window, len(values))
	}

	half := window / 2
	result := make([]float64, len(values))

	for index := range values {
		low, high := max(0, index-half), min(len(values), index+half+1)
		sum := 0.0
		for position := low; position < high; position++ {
			sum += values[position]
		}

		result[index] = sum / float64(high-low)
	}

	return Series{Values: result}, nil
}
