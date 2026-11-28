package filter

// CalculateExponential 对序列做一阶指数平滑（因果低通）
//   - alpha 取值 (0, 1]，越大越贴近当前样本，越小越平滑；
//   - 首个样本作为初始状态直接输出；
//   - 输出存在相位滞后，需要零相位对齐时应在正反两个方向各做一次。
func CalculateExponential(values []float64, alpha float64) (Series, error) {
	if err := validateValues(values); err != nil {
		return Series{}, err
	}

	if !isFinite(alpha) || alpha <= 0 || alpha > 1 {
		return Series{}, wrapError(ErrInvalidAlpha, "%g", alpha)
	}

	result := make([]float64, len(values))
	result[0] = values[0]

	for index := 1; index < len(values); index++ {
		result[index] = alpha*values[index] + (1-alpha)*result[index-1]
	}

	return Series{Values: result}, nil
}
