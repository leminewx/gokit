package optimizer

import "math"

// MinimizeBrent 在闭区间 [lower, upper] 内用 Brent 法求一元函数的局部极小
//   - 抛物线插值在函数接近二次时快速收敛，退化时自动切换为黄金分割，始终维持括住极小的区间；
//   - 收敛判据为极小点的不确定区间宽度相对 tolerance；
//   - 目标函数在搜索过程中出现 NaN/Inf 时返回 ErrNonFinite。
func MinimizeBrent(function Function, lower, upper float64, options ...Option) (Solution, error) {
	if function == nil {
		return Solution{}, ErrNilFunction
	}

	if !isFinite(lower) || !isFinite(upper) || lower >= upper {
		return Solution{}, wrapError(ErrInvalidRange, "[%g, %g]", lower, upper)
	}

	own := newConfig(options)
	if err := own.validate(); err != nil {
		return Solution{}, err
	}

	a, b := lower, upper
	x := a + _GOLDEN_RATIO*(b-a)
	w, v := x, x

	fx, err := evaluate(function, x)
	if err != nil {
		return Solution{}, err
	}

	fw, fv := fx, fx
	evaluations := 1
	d, e := 0.0, 0.0

	for iteration := 1; iteration <= own.maxIterations; iteration++ {
		middle := 0.5 * (a + b)
		tolerance := own.tolerance*math.Abs(x) + _MACHINE_EPSILON
		limit := 2 * tolerance

		if math.Abs(x-middle) <= limit-0.5*(b-a) {
			return Solution{X: x, Value: fx, Iterations: iteration, Evaluations: evaluations}, nil
		}

		// 抛物线插值：步长必须落在区间内且小于上一步的一半，否则退回黄金分割
		parabolic := false
		if math.Abs(e) > tolerance {
			r := (x - w) * (fx - fv)
			q := (x - v) * (fx - fw)
			p := (x-v)*q - (x-w)*r
			q = 2 * (q - r)

			if q > 0 {
				p = -p
			}

			q = math.Abs(q)
			previous := e
			e = d

			if math.Abs(p) < math.Abs(0.5*q*previous) && p > q*(a-x) && p < q*(b-x) {
				d = p / q
				step := x + d

				if step-a < limit || b-step < limit {
					d = math.Copysign(tolerance, middle-x)
				}

				parabolic = true
			}
		}

		if !parabolic {
			if x >= middle {
				e = a - x
			} else {
				e = b - x
			}

			d = _GOLDEN_RATIO * e
		}

		next := x + d
		if math.Abs(d) < tolerance {
			next = x + math.Copysign(tolerance, d)
		}

		fnext, err := evaluate(function, next)
		if err != nil {
			return Solution{}, err
		}

		evaluations++

		if fnext <= fx {
			if next >= x {
				a = x
			} else {
				b = x
			}

			v, w, x = w, x, next
			fv, fw, fx = fw, fx, fnext
			continue
		}

		if next < x {
			a = next
		} else {
			b = next
		}

		if fnext <= fw || w == x {
			v, w = w, next
			fv, fw = fw, fnext
		} else if fnext <= fv || v == x || v == w {
			v, fv = next, fnext
		}
	}

	return Solution{}, wrapError(ErrNotConverged, "%d iterations over [%g, %g]", own.maxIterations, lower, upper)
}
