package quadrature

import "math"

// gaussRule 为 n 点 Gauss-Legendre 公式在标准区间 [-1, 1] 上的节点与权重
type gaussRule struct {
	nodes   []float64
	weights []float64
}

// gaussRules 为 2~5 点 Gauss-Legendre 节点与权重（公开标准值），调用方不得修改
var gaussRules = map[int]gaussRule{
	2: {
		nodes:   []float64{-0.5773502691896257, 0.5773502691896257},
		weights: []float64{1, 1},
	},
	3: {
		nodes:   []float64{-0.7745966692414834, 0, 0.7745966692414834},
		weights: []float64{5.0 / 9, 8.0 / 9, 5.0 / 9},
	},
	4: {
		nodes:   []float64{-0.8611363115940526, -0.3399810435848563, 0.3399810435848563, 0.8611363115940526},
		weights: []float64{0.3478548451374538, 0.6521451548625461, 0.6521451548625461, 0.3478548451374538},
	},
	5: {
		nodes:   []float64{-0.906179845938664, -0.5384693101056831, 0, 0.5384693101056831, 0.906179845938664},
		weights: []float64{0.2369268850561891, 0.4786286704993665, 0.5688888888888889, 0.4786286704993665, 0.2369268850561891},
	},
}

// CalculateGaussLegendre 用 Gauss-Legendre 求积公式计算定积分
//   - order 为节点数（2~5）：n 点公式对 2n-1 次多项式精确；
//   - panels 为分区数，每个分区独立套用 n 点公式，总求值次数为 order*panels；
//   - 默认不提供误差估计，传入 WithErrorEstimation() 后以面板数翻倍计算差值，
//     并按 2^(2n)-1 做 Richardson 折算。
func CalculateGaussLegendre(function Function, a, b float64, options ...Option) (Result, error) {
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

	value, err := calculateGaussLegendre(function, a, b, own.order, own.panels)
	if err != nil {
		return Result{}, err
	}

	result := Result{Value: value, Iterations: 1}
	if own.errorEstimation {
		refined, err := calculateGaussLegendre(function, a, b, own.order, own.panels*2)
		if err != nil {
			return Result{}, err
		}

		result.ErrorEstimate = math.Abs(refined-value) / (math.Pow(2, float64(2*own.order)) - 1)
	}

	return result, nil
}

// calculateGaussLegendre 按给定节点数与面板数计算 Gauss-Legendre 积分
func calculateGaussLegendre(function Function, a, b float64, order, panels int) (float64, error) {
	rule, ok := gaussRules[order]
	if !ok {
		return 0, wrapError(ErrInvalidOrder, "%d", order)
	}

	width := (b - a) / float64(panels)
	half := width / 2
	total := 0.0

	for panel := 0; panel < panels; panel++ {
		center := a + float64(panel)*width + half
		subtotal := 0.0
		for index, node := range rule.nodes {
			point := center + half*node
			sample := function(point)
			if !isFinite(sample) {
				return 0, wrapError(ErrNonFinite, "f(%g)", point)
			}

			subtotal += rule.weights[index] * sample
		}

		total += subtotal * half
	}

	return total, nil
}
