package optimizer

import (
	"math"
	"sort"
)

// vertex 单纯形的一个顶点
type vertex struct {
	point []float64
	value float64
}

// MinimizeNelderMead 用 Nelder-Mead 单纯形法求多元函数的局部极小
//   - initial 为初始点，单纯形在此附近按 5% 的相对步长（零分量取 2.5e-4）构造；
//   - 收敛判据为顶点函数值落差与单纯形尺度同时小于 tolerance 对应的相对阈值；
//   - 目标函数出现 NaN/Inf 时返回 ErrNonFinite，达到迭代上限时返回 ErrNotConverged。
func MinimizeNelderMead(function MultiFunction, initial []float64, options ...Option) (MultiSolution, error) {
	if function == nil {
		return MultiSolution{}, ErrNilFunction
	}

	if len(initial) == 0 {
		return MultiSolution{}, ErrEmptyInitial
	}

	for index, value := range initial {
		if !isFinite(value) {
			return MultiSolution{}, wrapError(ErrNonFinite, "initial[%d]", index)
		}
	}

	own := newConfig(options)
	if err := own.validate(); err != nil {
		return MultiSolution{}, err
	}

	dimension := len(initial)
	vertices := make([]vertex, dimension+1)
	vertices[0] = vertex{point: append([]float64(nil), initial...)}

	for index := 1; index <= dimension; index++ {
		point := append([]float64(nil), initial...)
		if point[index-1] != 0 {
			point[index-1] *= 1 + _SIMPLEX_SCALE
		} else {
			point[index-1] = _SIMPLEX_OFFSET
		}

		vertices[index] = vertex{point: point}
	}

	evaluations := 0
	for index := range vertices {
		value, err := evaluateMulti(function, vertices[index].point)
		if err != nil {
			return MultiSolution{}, err
		}

		vertices[index].value = value
		evaluations++
	}

	for iteration := 1; iteration <= own.maxIterations; iteration++ {
		sort.Slice(vertices, func(first, second int) bool {
			return vertices[first].value < vertices[second].value
		})

		best := vertices[0]
		worst := vertices[dimension]

		magnitude := 1.0
		scale := 0.0
		for _, current := range vertices {
			for axis, coordinate := range current.point {
				magnitude = math.Max(magnitude, math.Abs(best.point[axis]))
				scale = math.Max(scale, math.Abs(coordinate-best.point[axis]))
			}
		}

		// 函数值落差与单纯形尺度同时收敛才停止，避免在狭长谷底提前退出
		if worst.value-best.value <= own.tolerance*math.Max(1, math.Abs(best.value)) &&
			scale <= own.tolerance*magnitude {
			return MultiSolution{
				X:           append([]float64(nil), best.point...),
				Value:       best.value,
				Iterations:  iteration,
				Evaluations: evaluations,
			}, nil
		}

		centroid := make([]float64, dimension)
		for index := 0; index < dimension; index++ {
			for axis := 0; axis < dimension; axis++ {
				centroid[axis] += vertices[index].point[axis]
			}
		}

		for axis := range centroid {
			centroid[axis] /= float64(dimension)
		}

		reflected := calculateReflection(centroid, worst.point, _REFLECTION)
		reflectedValue, err := evaluateMulti(function, reflected)
		if err != nil {
			return MultiSolution{}, err
		}

		evaluations++

		if reflectedValue < best.value {
			expanded := calculateReflection(centroid, worst.point, _EXPANSION)
			expandedValue, err := evaluateMulti(function, expanded)
			if err != nil {
				return MultiSolution{}, err
			}

			evaluations++

			if expandedValue < reflectedValue {
				vertices[dimension] = vertex{point: expanded, value: expandedValue}
			} else {
				vertices[dimension] = vertex{point: reflected, value: reflectedValue}
			}

			continue
		}

		if reflectedValue < vertices[dimension-1].value {
			vertices[dimension] = vertex{point: reflected, value: reflectedValue}
			continue
		}

		// 收缩：反射点优于最差点时沿反射方向外收缩，否则朝最优点内收缩
		inside := reflectedValue >= worst.value
		target := reflected
		if inside {
			target = worst.point
		}

		contracted := calculateInterpolation(centroid, target, _CONTRACTION)
		contractedValue, err := evaluateMulti(function, contracted)
		if err != nil {
			return MultiSolution{}, err
		}

		evaluations++

		accepted := contractedValue < worst.value
		if !inside {
			accepted = contractedValue <= reflectedValue
		}

		if accepted {
			vertices[dimension] = vertex{point: contracted, value: contractedValue}
			continue
		}

		// 收缩失败时向最优点整体收缩，并重新求值
		for index := 1; index <= dimension; index++ {
			for axis := 0; axis < dimension; axis++ {
				vertices[index].point[axis] = best.point[axis] +
					_SHRINK*(vertices[index].point[axis]-best.point[axis])
			}

			value, err := evaluateMulti(function, vertices[index].point)
			if err != nil {
				return MultiSolution{}, err
			}

			vertices[index].value = value
			evaluations++
		}
	}

	return MultiSolution{}, wrapError(ErrNotConverged, "%d iterations in %d dimensions", own.maxIterations, dimension)
}

// calculateInterpolation 计算 centroid + factor·(target - centroid)
func calculateInterpolation(centroid, target []float64, factor float64) []float64 {
	point := make([]float64, len(centroid))
	for axis := range centroid {
		point[axis] = centroid[axis] + factor*(target[axis]-centroid[axis])
	}

	return point
}

// calculateReflection 计算 centroid + factor·(centroid - origin)
func calculateReflection(centroid, origin []float64, factor float64) []float64 {
	point := make([]float64, len(centroid))
	for axis := range centroid {
		point[axis] = centroid[axis] + factor*(centroid[axis]-origin[axis])
	}

	return point
}
