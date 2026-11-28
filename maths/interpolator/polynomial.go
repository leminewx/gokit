package interpolator

// Polynomial 为整体插值多项式的升幂系数表示，构造后不可变
type Polynomial struct {
	coefficients []float64
}

// NewLagrange 由采样点构造 Lagrange 插值多项式
//   - 逐点展开基函数后线性组合，时间复杂度 O(n³)，适合采样点较少的场景；
//   - 与 NewNewton 得到的是同一条多项式，Newton 形式在数值上通常更稳定。
func NewLagrange(xs, ys []float64) (*Polynomial, error) {
	if err := validatePolynomialSamples(xs, ys); err != nil {
		return nil, err
	}

	coefficients := make([]float64, len(xs))
	for index := range xs {
		basis := calculateLagrangeBasis(xs, index)
		for degree, value := range basis {
			coefficients[degree] += ys[index] * value
		}
	}

	return &Polynomial{coefficients: coefficients}, nil
}

// NewNewton 由采样点用差商法构造插值多项式
//   - 先求各阶差商，再逐步展开成升幂系数，时间复杂度 O(n²)；
//   - 差商表本身支持增量添加节点，本实现仅返回展开后的多项式。
func NewNewton(xs, ys []float64) (*Polynomial, error) {
	if err := validatePolynomialSamples(xs, ys); err != nil {
		return nil, err
	}

	size := len(xs)
	differences := append([]float64(nil), ys...)
	for order := 1; order < size; order++ {
		for index := size - 1; index >= order; index-- {
			differences[index] = (differences[index] - differences[index-1]) / (xs[index] - xs[index-order])
		}
	}

	// Newton 形式：P = c0 + Σ c_k·∏(x - x_j)，乘积项需独立累加
	coefficients := []float64{differences[0]}
	product := []float64{1}
	for index := 1; index < size; index++ {
		next := make([]float64, len(product)+1)
		for degree, coefficient := range product {
			next[degree] -= xs[index-1] * coefficient
			next[degree+1] += coefficient
		}

		product = next
		if len(coefficients) < len(product) {
			coefficients = append(coefficients, make([]float64, len(product)-len(coefficients))...)
		}

		for degree, coefficient := range product {
			coefficients[degree] += differences[index] * coefficient
		}
	}

	return &Polynomial{coefficients: coefficients}, nil
}

// Calculate 用 Horner 方案求多项式在 x 处的值
func (own *Polynomial) Calculate(x float64) float64 {
	value := 0.0
	for index := len(own.coefficients) - 1; index >= 0; index-- {
		value = value*x + own.coefficients[index]
	}

	return value
}

// CalculateDerivative 用 Horner 方案求多项式在 x 处的导数
func (own *Polynomial) CalculateDerivative(x float64) float64 {
	value := 0.0
	for index := len(own.coefficients) - 1; index >= 1; index-- {
		value = value*x + float64(index)*own.coefficients[index]
	}

	return value
}

// GetDegree 返回多项式次数，等于采样点数减一
func (own *Polynomial) GetDegree() int {
	return len(own.coefficients) - 1
}

// GetCoefficients 返回升幂系数的副本，修改返回值不影响多项式本身
func (own *Polynomial) GetCoefficients() []float64 {
	return append([]float64(nil), own.coefficients...)
}

// calculateLagrangeBasis 展开第 index 个 Lagrange 基函数为升幂系数
func calculateLagrangeBasis(xs []float64, index int) []float64 {
	basis := []float64{1}
	denominator := 1.0

	for other := range xs {
		if other == index {
			continue
		}

		next := make([]float64, len(basis)+1)
		for degree, coefficient := range basis {
			next[degree] -= xs[other] * coefficient
			next[degree+1] += coefficient
		}

		basis = next
		denominator *= xs[index] - xs[other]
	}

	for degree := range basis {
		basis[degree] /= denominator
	}

	return basis
}

// validatePolynomialSamples 校验采样点并限制整体多项式的规模
func validatePolynomialSamples(xs, ys []float64) error {
	if err := validateSamples(xs, ys); err != nil {
		return err
	}

	if len(xs) > _MAX_POLYNOMIAL_SAMPLE {
		return wrapError(ErrTooManySamples, "%d, want <= %d", len(xs), _MAX_POLYNOMIAL_SAMPLE)
	}

	return nil
}
