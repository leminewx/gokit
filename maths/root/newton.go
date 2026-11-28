package root

import "math"

// CalculateBisection 用二分法求 function 在 [a, b] 上的根
//   - 要求 function(a) 与 function(b) 异号（端点恰为根时直接返回），否则返回 ErrNoSignChange；
//   - 收敛判据：区间半宽 (b-a)/2 <= tolerance，或启用残差容差且 |f(x)| <= residual；
//   - 收敛速度线性，每步误差减半，适合作为其他方法的兜底与区间确认。
func CalculateBisection(function Function, a, b float64, options ...Option) (Solution, error) {
	if function == nil {
		return Solution{}, ErrNilFunction
	}

	own := newConfig(options)
	if err := own.validate(); err != nil {
		return Solution{}, err
	}

	if !isFinite(a) || !isFinite(b) || a >= b {
		return Solution{}, wrapError(ErrInvalidInterval, "[%g, %g]", a, b)
	}

	valueA, valueB := function(a), function(b)
	if !isFinite(valueA) || !isFinite(valueB) {
		return Solution{}, wrapError(ErrNonFinite, "function value at the interval endpoints")
	}

	if valueA == 0 {
		return Solution{Root: a, Value: valueA}, nil
	}

	if valueB == 0 {
		return Solution{Root: b, Value: valueB}, nil
	}

	if (valueA > 0) == (valueB > 0) {
		return Solution{}, wrapError(ErrNoSignChange, "f(%g)=%g, f(%g)=%g", a, valueA, b, valueB)
	}

	for iteration := 1; iteration <= own.maxIterations; iteration++ {
		middle := a + (b-a)/2
		value := function(middle)
		if !isFinite(value) {
			return Solution{}, wrapError(ErrNonFinite, "f(%g)", middle)
		}

		if value == 0 || own.isConverged((b-a)/2, value) {
			return Solution{Root: middle, Value: value, Iterations: iteration}, nil
		}

		// 始终保留变号的一侧，保证区间内必定存在根
		if (valueA > 0) != (value > 0) {
			b, valueB = middle, value
		} else {
			a, valueA = middle, value
		}
	}

	return Solution{Root: (a + b) / 2, Value: function((a + b) / 2), Iterations: own.maxIterations},
		wrapError(ErrNotConverged, "%d iterations, interval [%g, %g] width %g", own.maxIterations, a, b, b-a)
}

// CalculateNewton 用牛顿法求根
//   - 未通过 WithDerivative 提供导数时，使用中心差分离散近似（精度略低于解析导数）；
//   - 收敛判据：|Δx| <= tolerance，或启用残差容差且 |f(x)| <= residual；
//   - 导数绝对值小于 1e-14 时返回 ErrZeroDerivative；
//   - 收敛速度二次，适合光滑函数与较好的初值。
func CalculateNewton(function Function, initial float64, options ...Option) (Solution, error) {
	if function == nil {
		return Solution{}, ErrNilFunction
	}

	own := newConfig(options)
	if err := own.validate(); err != nil {
		return Solution{}, err
	}

	if !isFinite(initial) {
		return Solution{}, wrapError(ErrInvalidInitial, "%g", initial)
	}

	current := initial
	for iteration := 1; iteration <= own.maxIterations; iteration++ {
		value := function(current)
		if !isFinite(value) {
			return Solution{}, wrapError(ErrNonFinite, "f(%g)", current)
		}

		derivative, err := own.calculateDerivative(function, current)
		if err != nil {
			return Solution{}, err
		}

		if math.Abs(derivative) < _ZERO_THRESHOLD {
			return Solution{}, wrapError(ErrZeroDerivative, "x=%g, f(x)=%g", current, value)
		}

		step := value / derivative
		next := current - step
		if !isFinite(next) {
			return Solution{}, wrapError(ErrNonFinite, "newton step at x=%g", current)
		}

		if own.isConverged(step, value) {
			return Solution{Root: next, Value: function(next), Iterations: iteration}, nil
		}

		current = next
	}

	return Solution{Root: current, Value: function(current), Iterations: own.maxIterations},
		wrapError(ErrNotConverged, "%d iterations, last x=%g", own.maxIterations, current)
}
