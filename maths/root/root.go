// Package root 提供一元实函数与复函数的数值求根方法
//   - 二分法：只需区间端点变号，收敛有保证，速度为线性；
//   - 牛顿法：需导数（可自动差分），二次收敛，依赖初值质量；
//   - 弦截法：无需导数，收敛阶约 1.618；
//   - 不动点迭代：求解 x = g(x)，收敛依赖压缩性，可选 Aitken 加速；
//   - Muller 法：抛物线插值，可求复根。
//
// 设计目标：
//   - 零第三方依赖，纯标准库实现；
//   - 统一返回 Solution / ComplexSolution 与 error，不返回裸元组，不 panic；
//   - 迭代中出现 NaN/Inf 立即以 ErrNonFinite 终止，避免污染值继续传播；
//   - 无全局可变状态，所有函数可并发调用。
//
// 典型用法：
//
//	solution, err := root.CalculateNewton(func(x float64) float64 { return x*x - 2 }, 1.5)
//	if err == nil {
//	    fmt.Println(solution.Root) // 1.4142135623730951
//	}
//
// 已知限制：
//   - 每个方法只保证找到一个根，多重根附近牛顿法退化为线性收敛；
//   - 不动点迭代不预判 g 的压缩性，发散时以 ErrNotConverged 结束；
//   - Muller 法对初值敏感，三个初值必须互不相同；
//   - 不识别区间内的不连续点，奇点附近的求根结果不可信。
package root

import "math"

// Function 一元实函数
type Function func(x float64) float64

// Derivative 一元实函数的导数
type Derivative func(x float64) float64

// ComplexFunction 一元复函数
type ComplexFunction func(z complex128) complex128

// Solution 实根求解结果
type Solution struct {
	Root       float64 // 收敛后的近似根
	Value      float64 // 根处的残差 f(Root)
	Iterations int     // 实际迭代次数
}

// ComplexSolution 复根求解结果
type ComplexSolution struct {
	Root       complex128 // 收敛后的近似根
	Value      complex128 // 根处的残差 f(Root)
	Iterations int        // 实际迭代次数
}

// Option 配置求根过程的选项
type Option func(*config)

type config struct {
	tolerance     float64    // 自变量容差
	residual      float64    // 残差容差，0 表示不启用
	maxIterations int        // 最大迭代次数
	derivative    Derivative // 显式导数，nil 时使用中心差分
	atiken        bool       // 不动点迭代是否启用 Aitken 加速
}

const (
	// _DEFAULT_TOLERANCE 为默认自变量容差
	_DEFAULT_TOLERANCE = 1e-9
	// _DEFAULT_MAX_ITERATION 为默认迭代上限
	_DEFAULT_MAX_ITERATION = 100
	// _MACHINE_EPSILON 为 float64 的机器精度（2^-52）
	_MACHINE_EPSILON = 2.220446049250313e-16
	// _ZERO_THRESHOLD 用于判定导数或分母是否已退化
	_ZERO_THRESHOLD = 1e-14
)

// WithTolerance 设置自变量收敛容差，必须为正
func WithTolerance(tolerance float64) Option {
	return func(own *config) { own.tolerance = tolerance }
}

// WithResidual 设置残差收敛容差 |f(x)| <= residual，必须为正
func WithResidual(residual float64) Option {
	return func(own *config) { own.residual = residual }
}

// WithMaxIterations 设置最大迭代次数，必须为正
func WithMaxIterations(iterations int) Option {
	return func(own *config) { own.maxIterations = iterations }
}

// WithDerivative 为牛顿法显式提供导数，省略时使用中心差分近似
func WithDerivative(derivative Derivative) Option {
	return func(own *config) { own.derivative = derivative }
}

// WithAitken 为不动点迭代启用 Aitken Δ² 加速
func WithAitken() Option {
	return func(own *config) { own.atiken = true }
}

// newConfig 应用选项并填充默认值
func newConfig(options []Option) *config {
	own := &config{
		tolerance:     _DEFAULT_TOLERANCE,
		maxIterations: _DEFAULT_MAX_ITERATION,
	}

	for _, option := range options {
		option(own)
	}

	return own
}

// validate 校验通用选项
func (own *config) validate() error {
	if !isFinite(own.tolerance) || own.tolerance <= 0 {
		return ErrInvalidTolerance
	}

	if !isFinite(own.residual) || own.residual < 0 {
		return ErrInvalidResidual
	}

	if own.maxIterations <= 0 {
		return ErrInvalidIterations
	}

	return nil
}

// isConverged 判断自变量增量是否已满足收敛条件
func (own *config) isConverged(step, value float64) bool {
	if math.Abs(step) <= own.tolerance {
		return true
	}

	return own.residual > 0 && math.Abs(value) <= own.residual
}

// calculateDerivative 返回显式导数，缺省时以中心差分离散近似
func (own *config) calculateDerivative(function Function, x float64) (float64, error) {
	if own.derivative != nil {
		value := own.derivative(x)
		if !isFinite(value) {
			return 0, wrapError(ErrNonFinite, "derivative at %g", x)
		}

		return value, nil
	}

	// 中心差分的最优步长量级为 eps^(1/3)，并随量级放大以保持相对精度
	step := math.Cbrt(_MACHINE_EPSILON) * math.Max(1, math.Abs(x))
	forward, backward := function(x+step), function(x-step)
	if !isFinite(forward) || !isFinite(backward) {
		return 0, wrapError(ErrNonFinite, "difference quotient at %g", x)
	}

	return (forward - backward) / (2 * step), nil
}

// isFinite 判断浮点数是否为有限值
func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// isFiniteComplex 判断复数是否为有限值
func isFiniteComplex(value complex128) bool {
	return isFinite(real(value)) && isFinite(imag(value))
}
