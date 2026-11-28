package ode

import "math"

// CalculateRungeKutta4 以固定步长求解初值问题 y' = f(x, y)
//   - 步长固定，最后一步自动截断到 stop；
//   - 无误差控制，精度随步长按四阶下降；
//   - 返回的 Trajectory 同时包含起点与终点，Steps 为实际执行的步数。
func CalculateRungeKutta4(function Function, initial []float64, start, stop, step float64) (Trajectory, error) {
	if function == nil {
		return Trajectory{}, ErrNilFunction
	}

	if err := validateState(initial); err != nil {
		return Trajectory{}, err
	}

	if err := validateRange(start, stop); err != nil {
		return Trajectory{}, err
	}

	if err := validateStep(step); err != nil {
		return Trajectory{}, err
	}

	if (stop-start)/step > _MAX_STEP_COUNT {
		return Trajectory{}, wrapError(ErrTooManySteps, "step %g over [%g, %g] exceeds %d", step, start, stop, _MAX_STEP_COUNT)
	}

	// 跨度可能不是步长的整数倍，末步需截断；浮点除法可能多出末位误差，
	// 先按相对精度收紧再向上取整，避免产生一个量级为 1e-16 的伪末步
	ratio := (stop - start) / step
	total := int(math.Ceil(ratio - _ROUNDING_RATIO*math.Max(1, ratio)))
	if total <= 0 {
		total = 1
	}

	x := start
	state := cloneValues(initial)
	trajectory := Trajectory{
		Xs: []float64{x},
		Ys: [][]float64{cloneValues(state)},
	}

	for index := 1; index <= total; index++ {
		// 节点位置按起点加整数倍步长计算，避免逐步累加带来的漂移
		nextX := start + float64(index)*step
		last := index == total || nextX >= stop
		if last {
			nextX = stop
		}

		width := nextX - x
		if width <= 0 {
			break
		}

		next, err := CalculateRungeKutta4Step(function, x, state, width)
		if err != nil {
			return Trajectory{}, err
		}

		x = nextX
		state = next
		trajectory.Xs = append(trajectory.Xs, x)
		trajectory.Ys = append(trajectory.Ys, cloneValues(state))
		trajectory.Steps++

		if last {
			break
		}
	}

	return trajectory, nil
}

// CalculateRungeKutta4Step 以经典四阶 Runge-Kutta 公式推进单步
func CalculateRungeKutta4Step(function Function, x float64, state []float64, step float64) ([]float64, error) {
	if function == nil {
		return nil, ErrNilFunction
	}

	if err := validateState(state); err != nil {
		return nil, err
	}

	if !isFinite(x) {
		return nil, wrapError(ErrNonFinite, "independent variable %g", x)
	}

	if err := validateStep(step); err != nil {
		return nil, err
	}

	size := len(state)
	half := step / 2

	k1, err := evaluate(function, x, state, size)
	if err != nil {
		return nil, err
	}

	k2, err := evaluate(function, x+half,
		calculateCombination(state, half, stage{values: k1, coefficient: 1}), size)
	if err != nil {
		return nil, err
	}

	k3, err := evaluate(function, x+half,
		calculateCombination(state, half, stage{values: k2, coefficient: 1}), size)
	if err != nil {
		return nil, err
	}

	k4, err := evaluate(function, x+step,
		calculateCombination(state, step, stage{values: k3, coefficient: 1}), size)
	if err != nil {
		return nil, err
	}

	return calculateCombination(state, step,
		stage{values: k1, coefficient: 1.0 / 6},
		stage{values: k2, coefficient: 1.0 / 3},
		stage{values: k3, coefficient: 1.0 / 3},
		stage{values: k4, coefficient: 1.0 / 6},
	), nil
}
