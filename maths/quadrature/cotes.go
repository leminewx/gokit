package quadrature

import "math"

// cotesWeights 为闭区间 Newton-Cotes 公式的归一化权重，键为节点数
// 权重和为 1，乘子区间宽度即为该区间的积分近似，调用方不得修改
var cotesWeights = map[int][]float64{
	2: {0.5, 0.5},
	3: {1.0 / 6, 4.0 / 6, 1.0 / 6},
	4: {1.0 / 8, 3.0 / 8, 3.0 / 8, 1.0 / 8},
	5: {7.0 / 90, 32.0 / 90, 12.0 / 90, 32.0 / 90, 7.0 / 90},
}

// CalculateCompositeCotes 用复化 Newton-Cotes 公式计算定积分
//   - order 为每个子区间的节点数：2 梯形、3 Simpson、4 Simpson 3/8、5 Boole；
//   - panels 为子区间个数，总求值次数为 order*panels；
//   - n 点公式对 n-1 次多项式精确（n 为奇数时可达 n 次）；
//   - 默认不提供误差估计，传入 WithErrorEstimation() 后以面板数翻倍计算差值，
//     并按 2^p-1 做 Richardson 折算（p 为该公式的误差阶）。
func CalculateCompositeCotes(function Function, a, b float64, options ...Option) (Result, error) {
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

	value, err := calculateCompositeCotes(function, a, b, own.order, own.panels)
	if err != nil {
		return Result{}, err
	}

	result := Result{Value: value, Iterations: 1}
	if own.errorEstimation {
		refined, err := calculateCompositeCotes(function, a, b, own.order, own.panels*2)
		if err != nil {
			return Result{}, err
		}

		result.ErrorEstimate = math.Abs(refined-value) / (math.Pow(2, float64(cotesAccuracy(own.order))) - 1)
	}

	return result, nil
}

// calculateCompositeCotes 按给定节点数与面板数计算复化 Newton-Cotes 积分
func calculateCompositeCotes(function Function, a, b float64, order, panels int) (float64, error) {
	weights := cotesWeights[order]
	if weights == nil {
		return 0, wrapError(ErrInvalidOrder, "%d", order)
	}

	width := (b - a) / float64(panels)
	step := width / float64(order-1)
	total := 0.0

	for panel := 0; panel < panels; panel++ {
		left := a + float64(panel)*width
		for index, weight := range weights {
			point := left + float64(index)*step
			sample := function(point)
			if !isFinite(sample) {
				return 0, wrapError(ErrNonFinite, "f(%g)", point)
			}

			total += weight * sample
		}
	}

	return total * width, nil
}

// cotesAccuracy 返回 n 点 Newton-Cotes 公式的误差阶
//   - 奇数节点（Simpson、Boole）误差为 O(h^(n+1))；
//   - 偶数节点（梯形、Simpson 3/8）误差为 O(h^n)。
func cotesAccuracy(order int) int {
	return 2 * ((order + 1) / 2)
}
