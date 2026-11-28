package interpolator

import "math"

// validateBoundary 校验边界条件与数据的相容性
func validateBoundary(condition BoundaryCondition, xs, ys []float64, own *config) error {
	if own.policy > EXTRAPOLATION_WRAP {
		return wrapError(ErrInvalidBoundary, "unknown extrapolation policy %d", own.policy)
	}

	switch condition {
	case BOUNDARY_CLAMPED:
		if !own.hasClampedBoundary {
			return ErrMissingDerivative
		}

		if !isFinite(own.leftDerivative) || !isFinite(own.rightDerivative) {
			return wrapError(ErrNonFinite, "clamped derivatives %g, %g", own.leftDerivative, own.rightDerivative)
		}
	case BOUNDARY_PERIODIC:
		if len(xs) < _PERIODIC_MIN_SAMPLE {
			return wrapError(ErrInvalidBoundary, "periodic boundary needs >= %d samples, got %d", _PERIODIC_MIN_SAMPLE, len(xs))
		}

		last := len(ys) - 1
		scale := math.Max(1, math.Abs(ys[0]))
		if math.Abs(ys[last]-ys[0]) > _PERIODIC_TOLERANCE*scale {
			return wrapError(ErrInvalidBoundary, "periodic boundary needs equal end values, got %g and %g", ys[0], ys[last])
		}
	case BOUNDARY_NOT_A_KNOT:
		if len(xs) < _NOT_A_KNOT_MIN_SAMPLE {
			return wrapError(ErrInvalidBoundary, "not-a-knot boundary needs >= %d samples, got %d", _NOT_A_KNOT_MIN_SAMPLE, len(xs))
		}
	case BOUNDARY_NATURAL:
	default:
		return wrapError(ErrInvalidBoundary, "unknown boundary condition %d", condition)
	}

	if own.policy == EXTRAPOLATION_WRAP && condition != BOUNDARY_PERIODIC {
		return wrapError(ErrInvalidBoundary, "wrap extrapolation requires the periodic boundary")
	}

	return nil
}

// calculateMoments 求解各节点的二阶导数
func calculateMoments(xs, ys []float64, condition BoundaryCondition, own *config) ([]float64, error) {
	size := len(xs)
	steps := make([]float64, size-1)
	for index := range steps {
		steps[index] = xs[index+1] - xs[index]
	}

	switch condition {
	case BOUNDARY_PERIODIC:
		return calculatePeriodicMoments(ys, steps)
	case BOUNDARY_NOT_A_KNOT:
		return calculateNotAKnotMoments(ys, steps)
	}

	lower := make([]float64, size)
	diagonal := make([]float64, size)
	upper := make([]float64, size)
	right := make([]float64, size)

	switch condition {
	case BOUNDARY_NATURAL:
		diagonal[0], diagonal[size-1] = 1, 1
	case BOUNDARY_CLAMPED:
		diagonal[0], upper[0] = 2, 1
		right[0] = 6 / steps[0] * ((ys[1]-ys[0])/steps[0] - own.leftDerivative)

		lower[size-1], diagonal[size-1] = 1, 2
		right[size-1] = 6 / steps[size-2] * (own.rightDerivative - (ys[size-1]-ys[size-2])/steps[size-2])
	}

	// 内部节点使用统一的三对角方程
	for index := 1; index < size-1; index++ {
		span := steps[index-1] + steps[index]
		lower[index] = steps[index-1] / span
		diagonal[index] = 2
		upper[index] = steps[index] / span
		right[index] = 6 / span * ((ys[index+1]-ys[index])/steps[index] - (ys[index]-ys[index-1])/steps[index-1])
	}

	return solveTridiagonal(lower, diagonal, upper, right)
}

// calculateNotAKnotMoments 用首末段三阶导数连续的条件求解非扭结样条
//   - 由三阶导数连续得到 M_0 与 M_{n-1} 关于内点的表达式；
//   - 代入首末内部节点的插值方程后，剩余未知量仍是标准三对角系统。
func calculateNotAKnotMoments(ys []float64, steps []float64) ([]float64, error) {
	size := len(ys)
	count := size - 2

	lower := make([]float64, count)
	diagonal := make([]float64, count)
	upper := make([]float64, count)
	right := make([]float64, count)

	first, second := steps[0], steps[1]
	diagonal[0] = 2 + first/second
	upper[0] = (second - first) / second
	right[0] = 6 / (first + second) * ((ys[2]-ys[1])/second - (ys[1]-ys[0])/first)

	for offset := 1; offset < count-1; offset++ {
		index := offset + 1
		span := steps[index-1] + steps[index]
		lower[offset] = steps[index-1] / span
		diagonal[offset] = 2
		upper[offset] = steps[index] / span
		right[offset] = 6 / span * ((ys[index+1]-ys[index])/steps[index] - (ys[index]-ys[index-1])/steps[index-1])
	}

	previous, last := steps[size-3], steps[size-2]
	lower[count-1] = (previous - last) / previous
	diagonal[count-1] = 2 + last/previous
	right[count-1] = 6 / (previous + last) * ((ys[size-1]-ys[size-2])/last - (ys[size-2]-ys[size-3])/previous)

	inner, err := solveTridiagonal(lower, diagonal, upper, right)
	if err != nil {
		return nil, err
	}

	moments := make([]float64, size)
	copy(moments[1:size-1], inner)
	moments[0] = ((first+second)*moments[1] - first*moments[2]) / second
	moments[size-1] = moments[size-2] + (last/previous)*(moments[size-2]-moments[size-3])

	return moments, nil
}

// calculatePeriodicMoments 求解周期样条的二阶导数
//   - 末节点与首节点重合，未知量为 M_0..M_{n-2}；
//   - 每个节点都按周期邻居列方程，形成循环三对角系统。
func calculatePeriodicMoments(ys []float64, steps []float64) ([]float64, error) {
	size := len(ys)
	count := size - 1
	wrap := steps[size-2]

	lower := make([]float64, count)
	diagonal := make([]float64, count)
	upper := make([]float64, count)
	right := make([]float64, count)

	for index := 0; index < count; index++ {
		previous := wrap
		if index > 0 {
			previous = steps[index-1]
		}

		next := wrap
		if index < count-1 {
			next = steps[index]
		}

		previousValue := ys[size-2]
		if index > 0 {
			previousValue = ys[index-1]
		}

		nextValue := ys[0]
		if index < count-1 {
			nextValue = ys[index+1]
		}

		span := previous + next
		lower[index] = previous / span
		diagonal[index] = 2
		upper[index] = next / span
		right[index] = 6 / span * ((nextValue-ys[index])/next - (ys[index]-previousValue)/previous)
	}

	// 左下角与右上角元素即首末节点的周期耦合项
	lowerLeft, upperRight := upper[count-1], lower[0]
	lower[0], upper[count-1] = 0, 0

	inner, err := solveCyclicTridiagonal(lower, diagonal, upper, right, lowerLeft, upperRight)
	if err != nil {
		return nil, err
	}

	moments := make([]float64, size)
	copy(moments, inner)
	moments[size-1] = moments[0]

	return moments, nil
}

// solveTridiagonal 用追赶法求解三对角线性方程组
//   - lower[0] 与 upper[n-1] 不参与计算，可为任意值；
//   - 主元绝对值小于容差时返回 ErrSingularSystem。
func solveTridiagonal(lower, diagonal, upper, right []float64) ([]float64, error) {
	size := len(diagonal)
	if size == 0 {
		return nil, wrapError(ErrSingularSystem, "empty system")
	}

	solution := make([]float64, size)
	modified := make([]float64, size)

	if math.Abs(diagonal[0]) < _PIVOT_TOLERANCE {
		return nil, wrapError(ErrSingularSystem, "zero pivot at row 0")
	}

	modified[0] = upper[0] / diagonal[0]
	solution[0] = right[0] / diagonal[0]

	for index := 1; index < size; index++ {
		pivot := diagonal[index] - lower[index]*modified[index-1]
		if math.Abs(pivot) < _PIVOT_TOLERANCE {
			return nil, wrapError(ErrSingularSystem, "zero pivot at row %d", index)
		}

		if index < size-1 {
			modified[index] = upper[index] / pivot
		}

		solution[index] = (right[index] - lower[index]*solution[index-1]) / pivot
	}

	for index := size - 2; index >= 0; index-- {
		solution[index] -= modified[index] * solution[index+1]
	}

	return solution, nil
}

// solveCyclicTridiagonal 用 Sherman-Morrison 公式求解循环三对角方程组
//   - lowerLeft 为左下角元素 A[n-1][0]，upperRight 为右上角元素 A[0][n-1]；
//   - 通过低秩修正把循环系统拆成两次标准三对角求解。
func solveCyclicTridiagonal(lower, diagonal, upper, right []float64, lowerLeft, upperRight float64) ([]float64, error) {
	size := len(diagonal)
	if size < 3 {
		return nil, wrapError(ErrSingularSystem, "cyclic system needs >= 3 unknowns, got %d", size)
	}

	gamma := -diagonal[0]
	if math.Abs(gamma) < _PIVOT_TOLERANCE {
		return nil, wrapError(ErrSingularSystem, "cyclic pivot is zero")
	}

	modifiedDiagonal := append([]float64(nil), diagonal...)
	modifiedDiagonal[0] -= gamma
	modifiedDiagonal[size-1] -= lowerLeft * upperRight / gamma

	first, err := solveTridiagonal(lower, modifiedDiagonal, upper, right)
	if err != nil {
		return nil, err
	}

	correction := make([]float64, size)
	correction[0], correction[size-1] = gamma, lowerLeft
	second, err := solveTridiagonal(lower, modifiedDiagonal, upper, correction)
	if err != nil {
		return nil, err
	}

	ratio := upperRight / gamma
	denominator := 1 + second[0] + ratio*second[size-1]
	if math.Abs(denominator) < _PIVOT_TOLERANCE {
		return nil, wrapError(ErrSingularSystem, "cyclic correction denominator is zero")
	}

	factor := (first[0] + ratio*first[size-1]) / denominator
	if !isFinite(factor) {
		return nil, wrapError(ErrNonFinite, "cyclic correction factor")
	}

	solution := make([]float64, size)
	for index := range solution {
		solution[index] = first[index] - factor*second[index]
	}

	return solution, nil
}
