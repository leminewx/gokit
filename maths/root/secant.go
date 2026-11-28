package root

import "math"

// CalculateSecant 用两点弦截法求根
//   - 需要两个初值 a、b 且 f(a) 与 f(b) 不相等，否则返回 ErrNoSignChange；
//   - 收敛判据：|Δx| <= tolerance，或启用残差容差且 |f(x)| <= residual；
//   - 收敛阶约 1.618，无需导数，适合导数难以解析、差分又不稳定的函数。
func CalculateSecant(function Function, a, b float64, options ...Option) (Solution, error) {
	if function == nil {
		return Solution{}, ErrNilFunction
	}

	own := newConfig(options)
	if err := own.validate(); err != nil {
		return Solution{}, err
	}

	if !isFinite(a) || !isFinite(b) || a == b {
		return Solution{}, wrapError(ErrInvalidInterval, "[%g, %g]", a, b)
	}

	previous, current := a, b
	valuePrevious, valueCurrent := function(previous), function(current)
	if !isFinite(valuePrevious) || !isFinite(valueCurrent) {
		return Solution{}, wrapError(ErrNonFinite, "function value at the initial points")
	}

	if math.Abs(valueCurrent-valuePrevious) < _ZERO_THRESHOLD {
		return Solution{}, wrapError(ErrNoSignChange, "f(%g)=%g, f(%g)=%g", a, valuePrevious, b, valueCurrent)
	}

	for iteration := 1; iteration <= own.maxIterations; iteration++ {
		next := current - valueCurrent*(current-previous)/(valueCurrent-valuePrevious)
		if !isFinite(next) {
			return Solution{}, wrapError(ErrNonFinite, "secant step at iteration %d", iteration)
		}

		value := function(next)
		if !isFinite(value) {
			return Solution{}, wrapError(ErrNonFinite, "f(%g)", next)
		}

		if own.isConverged(next-current, value) {
			return Solution{Root: next, Value: value, Iterations: iteration}, nil
		}

		previous, current = current, next
		valuePrevious, valueCurrent = valueCurrent, value

		if math.Abs(valueCurrent-valuePrevious) < _ZERO_THRESHOLD {
			return Solution{Root: current, Value: valueCurrent, Iterations: iteration},
				wrapError(ErrZeroDerivative, "secant slope vanished at x=%g", current)
		}
	}

	return Solution{Root: current, Value: valueCurrent, Iterations: own.maxIterations},
		wrapError(ErrNotConverged, "%d iterations, last x=%g", own.maxIterations, current)
}
