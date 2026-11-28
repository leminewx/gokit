// Package fitter 提供带权重的多项式最小二乘拟合
//   - 以加权范德蒙德矩阵为设计矩阵，用 Householder QR 求解，避免法方程的平方条件数；
//   - 权重为 0 的样本会被忽略，正权重按 1/σ² 的语义理解；
//   - 同时给出加权残差平方和、加权总平方和与决定系数 R²；
//   - 系数按升幂排列，Coefficients[0] 为常数项。
//
// 设计目标：
//   - 零第三方依赖，纯标准库实现；
//   - 统一返回 Fit 与 error，不返回裸元组，不 panic；
//   - 无全局可变状态，所有函数与方法可并发调用。
//
// 已知限制：
//   - 只在原始坐标上做多项式拟合，自变量跨度很大时建议先归一化再拟合；
//   - 阶数上限为 12，更高阶的多项式在 float64 下已无明显数值意义；
//   - R² 不能判定模型是否过拟合，需要结合有效样本数与残差量级判断。
package fitter

import "math"

// Sample 带权重的二维样本
type Sample struct {
	X      float64 // 自变量
	Y      float64 // 因变量
	Weight float64 // 非负权重，0 表示忽略该样本
}

// Fit 多项式最小二乘拟合结果
type Fit struct {
	Coefficients     []float64 // 从常数项到 degree 次项的系数
	WeightedResidual float64   // 加权残差平方和 Σw(y-ŷ)²
	WeightedTotal    float64   // 加权总平方和 Σw(y-ȳ)²，ȳ 为加权均值
	RSquared         float64   // 决定系数 1 - 残差/总平方和，总平方和为 0 时取 1
	Samples          int       // 参与拟合的正权重样本数
}

const (
	// _MAX_DEGREE 为多项式阶数上限
	_MAX_DEGREE = 12
	// _SINGULAR_TOLERANCE 用于判定设计矩阵列是否退化（相对于列最大元素）
	_SINGULAR_TOLERANCE = 1e-12
)

// CalculateValue 用拟合系数在 x 处求值（Horner 形式）
func (own *Fit) CalculateValue(x float64) float64 {
	value := 0.0
	for index := len(own.Coefficients) - 1; index >= 0; index-- {
		value = value*x + own.Coefficients[index]
	}

	return value
}

// isFinite 判断浮点数是否为有限值
func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
