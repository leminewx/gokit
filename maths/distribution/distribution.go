// Package distribution 提供常用概率分布的密度、分布函数与分位数
//   - Normal：正态分布，分位数基于 math.Erfinv；
//   - Uniform：连续均匀分布；
//   - Exponential：指数分布，以速率 λ 参数化；
//   - Poisson：泊松分布，以均值 λ 参数化。
//
// 设计目标：
//   - 零第三方依赖，纯标准库实现；
//   - 构造函数统一校验参数，非法参数立即返回 error；
//   - 密度与分布函数返回 (值, error)，求值点非有限时返回 ErrNonFinite；
//   - 无全局可变状态，所有类型与方法可并发调用。
//
// 已知限制：
//   - 泊松分布的密度与分布函数逐项求和，均值极大时计算量随 k 线性增长；
//   - 分位数只接受 (0, 1) 内的概率，端点对应的极限值不返回；
//   - 不提供随机抽样，抽样能力见 maths/sampler。
package distribution

import "math"

const (
	// _SQRT_TWO_PI 为 √(2π)
	_SQRT_TWO_PI = 2.5066282746310002
	// _POISSON_TAIL 控制泊松分布 CDF 的近 1 尾部截断，取 40 倍标准差
	_POISSON_TAIL = 40.0
	// _MAX_POISSON_ITERATION 为泊松分位数搜索的项数上限
	_MAX_POISSON_ITERATION = 1000000
)

// validateProbability 校验概率位于 (0, 1)
func validateProbability(probability float64) error {
	if !isFinite(probability) || probability <= 0 || probability >= 1 {
		return wrapError(ErrInvalidProbability, "%g", probability)
	}

	return nil
}

// isFinite 判断浮点数是否为有限值
func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
