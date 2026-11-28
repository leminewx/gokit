package quadrature

import "math"

// CalculateRomberg 用 Romberg 积分法计算 function 在 [a, b] 上的定积分
//   - 第 0 行是梯形公式，之后每次把步长减半，并对各行做 Richardson 外推；
//   - 收敛判据：相邻两行对角元之差（即 ErrorEstimate）<= tolerance；
//   - ErrorEstimate 始终填充，含义是最后两次外推结果之差的绝对值；
//   - 至少需要两行才能给出误差估计，因此 WithMaxIterations(1) 必然返回 ErrNotConverged。
func CalculateRomberg(function Function, a, b float64, options ...Option) (Result, error) {
	if function == nil {
		return Result{}, ErrNilFunction
	}

	own := newConfig(options)
	if err := own.validate(); err != nil {
		return Result{}, err
	}

	if err := validateInterval(a, b); err != nil {
		return Result{}, err
	}

	step := b - a
	value := (function(a) + function(b)) * step / 2
	if !isFinite(value) {
		return Result{}, wrapError(ErrNonFinite, "trapezoid on [%g, %g]", a, b)
	}

	rows := [][]float64{{value}}
	estimate := math.Inf(1)

	for iteration := 1; iteration < own.maxIterations; iteration++ {
		step /= 2
		if step == 0 {
			break
		}

		sum := 0.0
		// 步长减半后只需累加新增的奇数点，故增量为 2*step
		for x := a + step; x < b; x += 2 * step {
			sample := function(x)
			if !isFinite(sample) {
				return Result{}, wrapError(ErrNonFinite, "f(%g)", x)
			}

			sum += sample
		}

		previous := rows[iteration-1]
		row := make([]float64, iteration+1)
		row[0] = previous[0]/2 + step*sum
		for index := 1; index <= iteration; index++ {
			factor := math.Pow(4, float64(index)) - 1
			row[index] = row[index-1] + (row[index-1]-previous[index-1])/factor
		}

		rows = append(rows, row)
		value = row[iteration]
		estimate = math.Abs(row[iteration] - previous[iteration-1])
		if !isFinite(value) || !isFinite(estimate) {
			return Result{}, wrapError(ErrNonFinite, "romberg row %d", iteration)
		}

		if estimate <= own.tolerance {
			return Result{Value: value, Iterations: iteration + 1, ErrorEstimate: estimate}, nil
		}
	}

	return Result{Value: value, Iterations: len(rows), ErrorEstimate: estimate},
		wrapError(ErrNotConverged, "%d rows, estimate %g", len(rows), estimate)
}
