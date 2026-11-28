package differentiator

import "math"

// CalculateFunctionDerivative 用五点模板 + 一次 Richardson 外推计算一阶导数
// 精度约六阶，误差指示取两次外推估计之差。
func CalculateFunctionDerivative(function Function, x float64) (Derivative, error) {
	return calculateFunctionDerivative(function, x, 1)
}

// CalculateFunctionSecondDerivative 用五点模板 + 一次 Richardson 外推计算二阶导数
// 精度约六阶，误差指示取两次外推估计之差。
func CalculateFunctionSecondDerivative(function Function, x float64) (Derivative, error) {
	return calculateFunctionDerivative(function, x, 2)
}

// calculateFunctionDerivative 按阶数选择五点模板并做 Richardson 外推
func calculateFunctionDerivative(function Function, x float64, order int) (Derivative, error) {
	if function == nil {
		return Derivative{}, ErrNilFunction
	}

	if !isFinite(x) {
		return Derivative{}, wrapError(ErrNonFinite, "evaluation point %g", x)
	}

	// 五点模板的舍入与截断误差在步长量级 eps^(1/5) 处平衡，并随自变量量级放大
	step := math.Pow(_MACHINE_EPSILON, 0.2) * math.Max(1, math.Abs(x))

	coarse, err := calculateFivePoint(function, x, step, order)
	if err != nil {
		return Derivative{}, err
	}

	fine, err := calculateFivePoint(function, x, step/2, order)
	if err != nil {
		return Derivative{}, err
	}

	// 两次半步长结果做 Richardson 外推，主导误差项相互抵消
	return Derivative{
		Value:         (16*fine - coarse) / 15,
		ErrorEstimate: math.Abs(fine-coarse) / 15,
		Window:        _FIVE_POINT_WINDOW,
	}, nil
}

// calculateFivePoint 在等距五点模板上按阶数求导
func calculateFivePoint(function Function, x, step float64, order int) (float64, error) {
	values := make([]float64, _FIVE_POINT_WINDOW)
	for index, offset := range []float64{-2, -1, 0, 1, 2} {
		position := x + offset*step
		value := function(position)

		if !isFinite(value) {
			return 0, wrapError(ErrNonFinite, "function at %g", position)
		}

		values[index] = value
	}

	value := (values[0] - 8*values[1] + 8*values[3] - values[4]) / (12 * step)
	if order == 2 {
		value = (-values[4] + 16*values[3] - 30*values[2] + 16*values[1] - values[0]) / (12 * step * step)
	}

	if !isFinite(value) {
		return 0, wrapError(ErrNonFinite, "derivative at %g", x)
	}

	return value, nil
}
