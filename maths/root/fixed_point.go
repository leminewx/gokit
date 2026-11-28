package root

import "math"

// CalculateFixedPoint 用不动点迭代求 x = function(x) 的解
//   - function 即迭代映射 g(x)，收敛要求其在根邻域内为压缩映射（|g'| < 1），本函数不做预判；
//   - 启用 WithAitken() 时使用 Aitken Δ² 加速，可把线性收敛改善为近似二次收敛；
//   - 收敛判据：|Δx| <= tolerance，或启用残差容差且 |g(x)-x| <= residual；
//   - Solution.Value 返回不动点残差 g(Root)-Root；
//   - 发散（超过迭代上限或出现非有限值）分别返回 ErrNotConverged / ErrNonFinite。
func CalculateFixedPoint(function Function, initial float64, options ...Option) (Solution, error) {
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
		next := function(current)
		if !isFinite(next) {
			return Solution{}, wrapError(ErrNonFinite, "g(%g)", current)
		}

		residual := next - current
		if own.isConverged(residual, residual) {
			return Solution{Root: next, Value: residual, Iterations: iteration}, nil
		}

		if own.atiken {
			if accelerated, ok, err := calculateAitken(function, current, next); err != nil {
				return Solution{}, err
			} else if ok {
				current = accelerated
				continue
			}
		}

		current = next
	}

	return Solution{Root: current, Value: function(current) - current, Iterations: own.maxIterations},
		wrapError(ErrNotConverged, "%d iterations, last x=%g", own.maxIterations, current)
}

// calculateAitken 用 Aitken Δ² 公式由连续两次迭代值外推加速值
//   - 返回 ok=false 表示分母退化（接近等差收敛），调用方应回退到普通迭代；
//   - 需要额外一次 g 的求值，因此每次加速消耗两次函数调用。
func calculateAitken(function Function, current, next float64) (float64, bool, error) {
	following := function(next)
	if !isFinite(following) {
		return 0, false, wrapError(ErrNonFinite, "g(%g)", next)
	}

	denominator := following - 2*next + current
	if math.Abs(denominator) < _ZERO_THRESHOLD {
		return 0, false, nil
	}

	accelerated := current - (next-current)*(next-current)/denominator
	if !isFinite(accelerated) {
		return 0, false, wrapError(ErrNonFinite, "aitken extrapolation at x=%g", next)
	}

	return accelerated, true, nil
}
