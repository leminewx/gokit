package differentiator

import "math"

// CalculateSavitzkyGolay 对等距样本做 Savitzky-Golay 平滑，并给出单位间隔上的一阶导数
//   - 窗口长度为奇数，局部拟合 order 次多项式；order 为 0 时只平滑，导数全为 0；
//   - 平滑结果对不超过 order 次的多项式精确（不改变原值）；
//   - 两端不足半窗的点保持原值、导数置 0，避免边界外推；
//   - 导数以样本间隔 1 计算，真实间隔为 dx 时调用方需自行除以 dx。
func CalculateSavitzkyGolay(values []float64, window, order int) (Series, error) {
	if len(values) == 0 {
		return Series{}, ErrEmptyInput
	}

	if window < _MIN_WINDOW || window > _MAX_WINDOW || window%2 == 0 {
		return Series{}, wrapError(ErrInvalidWindow, "%d", window)
	}

	if order < 0 || order > _MAX_ORDER || order >= window {
		return Series{}, wrapError(ErrInvalidOrder, "%d with window %d", order, window)
	}

	if len(values) < window {
		return Series{}, wrapError(ErrTooFewSamples, "%d < %d", len(values), window)
	}

	for index, value := range values {
		if !isFinite(value) {
			return Series{}, wrapError(ErrNonFinite, "values[%d]", index)
		}
	}

	smoothing, derivative, err := calculateWeights(window, order)
	if err != nil {
		return Series{}, err
	}

	half := window / 2
	result := Series{
		Values:      append([]float64(nil), values...),
		Derivatives: make([]float64, len(values)),
		Window:      window,
		Order:       order,
	}

	for index := half; index+half < len(values); index++ {
		smooth, slope := 0.0, 0.0
		for offset := 0; offset < window; offset++ {
			value := values[index-half+offset]
			smooth += smoothing[offset] * value
			slope += derivative[offset] * value
		}

		result.Values[index] = smooth
		result.Derivatives[index] = slope
	}

	return result, nil
}

// calculateWeights 求局部多项式最小二乘的平滑权重与一阶导数权重
// 等距窗口的设计矩阵与位置无关，故权重只需计算一次；增广矩阵为 [AᵀA | Aᵀ]。
func calculateWeights(window, order int) ([]float64, []float64, error) {
	size := order + 1
	half := window / 2

	nodes := make([]float64, window)
	powers := make([][]float64, size)
	for index := 0; index < window; index++ {
		nodes[index] = float64(index - half)
	}

	for degree := 0; degree < size; degree++ {
		powers[degree] = make([]float64, window)
		for index := 0; index < window; index++ {
			powers[degree][index] = math.Pow(nodes[index], float64(degree))
		}
	}

	rows := make([][]float64, size)
	for row := 0; row < size; row++ {
		augmented := make([]float64, size+window)

		for column := 0; column < size; column++ {
			sum := 0.0
			for index := 0; index < window; index++ {
				sum += powers[row][index] * powers[column][index]
			}
			augmented[column] = sum
		}

		copy(augmented[size:], powers[row])
		rows[row] = augmented
	}

	// 高斯-约当消元并部分选主元，把 [AᵀA | Aᵀ] 化为 [I | (AᵀA)⁻¹Aᵀ]
	for column := 0; column < size; column++ {
		pivot := column
		for row := column + 1; row < size; row++ {
			if math.Abs(rows[row][column]) > math.Abs(rows[pivot][column]) {
				pivot = row
			}
		}

		if math.Abs(rows[pivot][column]) < _SINGULAR_TOLERANCE {
			return nil, nil, ErrSingularSystem
		}

		rows[column], rows[pivot] = rows[pivot], rows[column]

		scale := rows[column][column]
		for index := column; index < size+window; index++ {
			rows[column][index] /= scale
		}

		for row := 0; row < size; row++ {
			if row == column {
				continue
			}

			factor := rows[row][column]
			if factor == 0 {
				continue
			}

			for index := column; index < size+window; index++ {
				rows[row][index] -= factor * rows[column][index]
			}
		}
	}

	smoothing := append([]float64(nil), rows[0][size:]...)
	derivative := make([]float64, window)
	if order >= 1 {
		copy(derivative, rows[1][size:])
	}

	return smoothing, derivative, nil
}
