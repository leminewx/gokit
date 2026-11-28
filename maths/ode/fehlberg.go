package ode

import "math"

// CalculateFehlberg 以 Runge-Kutta-Fehlberg 4(5) 方法求解初值问题 y' = f(x, y)
//   - 每步用同一组六级求值同时得到四阶与五阶解，两者之差作为局部误差估计；
//   - 误差超限则缩小步长重试（计入 Rejected），达标则接受并尝试放大步长；
//   - 步长因子限制在 [_MIN_STEP_SHRINK, _MAX_STEP_GROWTH] 内，避免剧烈震荡；
//   - 步长触底时强制接受，防止在奇点附近陷入死循环。
func CalculateFehlberg(function Function, initial []float64, start, stop float64, options ...Option) (Trajectory, error) {
	if function == nil {
		return Trajectory{}, ErrNilFunction
	}

	if err := validateState(initial); err != nil {
		return Trajectory{}, err
	}

	if err := validateRange(start, stop); err != nil {
		return Trajectory{}, err
	}

	own := newConfig(options)
	if err := own.validate(); err != nil {
		return Trajectory{}, err
	}

	own.resolve(stop - start)

	x := start
	state := cloneValues(initial)
	width := own.initialStep
	attempts := 0
	trajectory := Trajectory{
		Xs: []float64{x},
		Ys: [][]float64{cloneValues(state)},
	}

	for x < stop {
		if attempts >= own.maxIterations {
			return Trajectory{}, wrapError(ErrNotConverged, "%d attempts ending at x = %g", attempts, x)
		}

		step := math.Min(width, stop-x)
		covered := step >= stop-x
		next, estimate, err := calculateFehlbergStep(function, x, state, step)
		if err != nil {
			return Trajectory{}, err
		}

		attempts++

		if estimate <= own.tolerance || step <= own.minStep {
			// step 取自 min(width, stop-x)，覆盖末段时直接对齐到 stop
			if covered {
				x = stop
			} else {
				x += step
			}

			state = next
			trajectory.Xs = append(trajectory.Xs, x)
			trajectory.Ys = append(trajectory.Ys, cloneValues(state))
			trajectory.Steps++
		} else {
			trajectory.Rejected++
		}

		width = own.calculateStep(step, estimate)
	}

	return trajectory, nil
}

// calculateFehlbergStep 推进单步，返回五阶解与局部误差估计
// Butcher 系数取经典 Fehlberg 4(5) 表：
//
//	c2=1/4      a21=1/4
//	c3=3/8      a31=3/32          a32=9/32
//	c4=12/13    a41=1932/2197     a42=-7200/2197    a43=7296/2197
//	c5=1        a51=439/216       a52=-8            a53=3680/513   a54=-845/4104
//	c6=1/2      a61=-8/27         a62=2             a63=-3544/2565 a64=1859/4104 a65=-11/40
//	b(4)=25/216, 1408/2565, 2197/4104, -1/5
//	b(5)=16/135, 6656/12825, 28561/56430, -9/50, 2/55
func calculateFehlbergStep(function Function, x float64, state []float64, step float64) ([]float64, float64, error) {
	size := len(state)
	half := step / 2

	k1, err := evaluate(function, x, state, size)
	if err != nil {
		return nil, 0, err
	}

	k2, err := evaluate(function, x+step/4,
		calculateCombination(state, step, stage{values: k1, coefficient: 1.0 / 4}), size)
	if err != nil {
		return nil, 0, err
	}

	k3, err := evaluate(function, x+3*step/8,
		calculateCombination(state, step,
			stage{values: k1, coefficient: 3.0 / 32},
			stage{values: k2, coefficient: 9.0 / 32}), size)
	if err != nil {
		return nil, 0, err
	}

	k4, err := evaluate(function, x+12*step/13,
		calculateCombination(state, step,
			stage{values: k1, coefficient: 1932.0 / 2197},
			stage{values: k2, coefficient: -7200.0 / 2197},
			stage{values: k3, coefficient: 7296.0 / 2197}), size)
	if err != nil {
		return nil, 0, err
	}

	k5, err := evaluate(function, x+step,
		calculateCombination(state, step,
			stage{values: k1, coefficient: 439.0 / 216},
			stage{values: k2, coefficient: -8},
			stage{values: k3, coefficient: 3680.0 / 513},
			stage{values: k4, coefficient: -845.0 / 4104}), size)
	if err != nil {
		return nil, 0, err
	}

	k6, err := evaluate(function, x+half,
		calculateCombination(state, step,
			stage{values: k1, coefficient: -8.0 / 27},
			stage{values: k2, coefficient: 2},
			stage{values: k3, coefficient: -3544.0 / 2565},
			stage{values: k4, coefficient: 1859.0 / 4104},
			stage{values: k5, coefficient: -11.0 / 40}), size)
	if err != nil {
		return nil, 0, err
	}

	fourth := calculateCombination(state, step,
		stage{values: k1, coefficient: 25.0 / 216},
		stage{values: k3, coefficient: 1408.0 / 2565},
		stage{values: k4, coefficient: 2197.0 / 4104},
		stage{values: k5, coefficient: -1.0 / 5})

	fifth := calculateCombination(state, step,
		stage{values: k1, coefficient: 16.0 / 135},
		stage{values: k3, coefficient: 6656.0 / 12825},
		stage{values: k4, coefficient: 28561.0 / 56430},
		stage{values: k5, coefficient: -9.0 / 50},
		stage{values: k6, coefficient: 2.0 / 55})

	estimate := 0.0
	for index := range fifth {
		if !isFinite(fifth[index]) {
			return nil, 0, wrapError(ErrNonFinite, "solution[%d] at x = %g", index, x)
		}

		if difference := math.Abs(fifth[index] - fourth[index]); difference > estimate {
			estimate = difference
		}
	}

	return fifth, estimate, nil
}
