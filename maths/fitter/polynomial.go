package fitter

import "math"

// CalculatePolynomial 对样本做 degree 次多项式加权最小二乘拟合
//   - degree 取值 0..12，正权重样本数必须大于 degree 才能确定全部系数；
//   - 权重为 0 的样本被忽略，负权重或非有限值返回错误；
//   - 返回的系数按升幂排列，可用 Fit.CalculateValue 求值。
func CalculatePolynomial(samples []Sample, degree int) (Fit, error) {
	if len(samples) == 0 {
		return Fit{}, ErrEmptyInput
	}

	if degree < 0 || degree > _MAX_DEGREE {
		return Fit{}, wrapError(ErrInvalidDegree, "%d, want 0..%d", degree, _MAX_DEGREE)
	}

	weighted := 0
	for index, sample := range samples {
		if !isFinite(sample.X) || !isFinite(sample.Y) || !isFinite(sample.Weight) {
			return Fit{}, wrapError(ErrNonFinite, "sample[%d]", index)
		}

		if sample.Weight < 0 {
			return Fit{}, wrapError(ErrNegativeWeight, "sample[%d] = %g", index, sample.Weight)
		}

		if sample.Weight > 0 {
			weighted++
		}
	}

	if weighted <= degree {
		return Fit{}, wrapError(ErrTooFewSamples, "%d <= %d", weighted, degree)
	}

	size := degree + 1
	matrix := make([][]float64, 0, weighted)
	rhs := make([]float64, 0, weighted)

	for _, sample := range samples {
		if sample.Weight == 0 {
			continue
		}

		scale := math.Sqrt(sample.Weight)
		row := make([]float64, size)
		power := scale
		for column := 0; column < size; column++ {
			row[column] = power
			power *= sample.X
		}

		matrix = append(matrix, row)
		rhs = append(rhs, scale*sample.Y)
	}

	coefficients, err := calculateLeastSquares(matrix, rhs)
	if err != nil {
		return Fit{}, err
	}

	residual, total := calculateStatistics(samples, coefficients)
	return Fit{
		Coefficients:     coefficients,
		WeightedResidual: residual,
		WeightedTotal:    total,
		RSquared:         calculateRSquared(residual, total),
		Samples:          weighted,
	}, nil
}

// calculateLeastSquares 用 Householder QR 求解超定方程 A c = b
// A 的前 columns 行在返回时已被就地改写，调用方不应复用。
func calculateLeastSquares(matrix [][]float64, rhs []float64) ([]float64, error) {
	rows, columns := len(matrix), len(matrix[0])

	for column := 0; column < columns; column++ {
		scale, norm := 0.0, 0.0
		for row := column; row < rows; row++ {
			value := matrix[row][column]
			scale = math.Max(scale, math.Abs(value))
			norm += value * value
		}

		if scale == 0 || math.Sqrt(norm) <= _SINGULAR_TOLERANCE*scale {
			return nil, wrapError(ErrSingularSystem, "column %d", column)
		}

		norm = math.Sqrt(norm)
		pivot := matrix[column][column]
		if pivot > 0 {
			norm = -norm
		}

		// Householder 向量 v = x - norm·e1，其中 x 为当前列的对角以下部分
		vector := make([]float64, rows-column)
		vector[0] = pivot - norm
		for row := column + 1; row < rows; row++ {
			vector[row-column] = matrix[row][column]
		}

		square := 0.0
		for _, value := range vector {
			square += value * value
		}

		if square > 0 {
			factor := 2 / square
			applyHouseholder(matrix, vector, factor, column, columns)
			applyHouseholderVector(rhs, vector, factor, column)
		}

		matrix[column][column] = norm
		for row := column + 1; row < rows; row++ {
			matrix[row][column] = 0
		}
	}

	coefficients := make([]float64, columns)
	for row := columns - 1; row >= 0; row-- {
		sum := rhs[row]
		for column := row + 1; column < columns; column++ {
			sum -= matrix[row][column] * coefficients[column]
		}

		if matrix[row][row] == 0 {
			return nil, wrapError(ErrSingularSystem, "diagonal %d", row)
		}

		coefficients[row] = sum / matrix[row][row]
	}

	return coefficients, nil
}

// applyHouseholder 把 H = I - factor·v·vᵀ 左乘到矩阵的 [column:, column:] 子块
func applyHouseholder(matrix [][]float64, vector []float64, factor float64, column, columns int) {
	for sub := column; sub < columns; sub++ {
		dot := 0.0
		for row := column; row < len(matrix); row++ {
			dot += vector[row-column] * matrix[row][sub]
		}

		dot *= factor
		for row := column; row < len(matrix); row++ {
			matrix[row][sub] -= dot * vector[row-column]
		}
	}
}

// applyHouseholderVector 对右端向量施加同一个 Householder 变换
func applyHouseholderVector(values, vector []float64, factor float64, column int) {
	dot := 0.0
	for row := column; row < len(values); row++ {
		dot += vector[row-column] * values[row]
	}

	dot *= factor
	for row := column; row < len(values); row++ {
		values[row] -= dot * vector[row-column]
	}
}

// calculateStatistics 返回加权残差平方和与加权总平方和
func calculateStatistics(samples []Sample, coefficients []float64) (float64, float64) {
	weightSum, valueSum := 0.0, 0.0
	for _, sample := range samples {
		weightSum += sample.Weight
		valueSum += sample.Weight * sample.Y
	}

	mean := valueSum / weightSum
	residual, total := 0.0, 0.0

	for _, sample := range samples {
		if sample.Weight == 0 {
			continue
		}

		predicted := 0.0
		for index := len(coefficients) - 1; index >= 0; index-- {
			predicted = predicted*sample.X + coefficients[index]
		}

		difference := sample.Y - predicted
		residual += sample.Weight * difference * difference
		total += sample.Weight * (sample.Y - mean) * (sample.Y - mean)
	}

	return residual, total
}

// calculateRSquared 由加权残差与总平方和给出决定系数
func calculateRSquared(residual, total float64) float64 {
	if total == 0 {
		return 1
	}

	return 1 - residual/total
}
