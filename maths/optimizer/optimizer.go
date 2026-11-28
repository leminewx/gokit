// Package optimizer 提供无约束极小化的数值方法
//   - Brent 法：一维区间搜索，抛物线插值与黄金分割混合，无需导数；
//   - Nelder-Mead：多维单纯形演化，只依赖函数值，适合不可导或噪声较大的目标。
//
// 设计目标：
//   - 零第三方依赖，纯标准库实现；
//   - 统一返回 Solution / MultiSolution 与 error，不返回裸元组，不 panic；
//   - 目标函数出现 NaN/Inf 立即以 ErrNonFinite 终止；
//   - 无全局可变状态，所有函数可并发调用。
//
// 典型用法：
//
//	solution, err := optimizer.MinimizeBrent(func(x float64) float64 { return (x - 2) * (x - 2) }, -5, 5)
//	if err == nil {
//	    fmt.Println(solution.X, solution.Value) // 2 0
//	}
//
// 已知限制：
//   - 只求局部极小，非凸问题的结果依赖搜索区间或初值；
//   - Nelder-Mead 在 10 维以上收敛明显变慢，高维问题应换用基于梯度的方法；
//   - 目标函数无下界或区间内没有极小值时以 ErrNotConverged 结束。
package optimizer

import "math"

// Function 一元目标函数
type Function func(x float64) float64

// MultiFunction 多元目标函数，入参切片在调用期间只读
type MultiFunction func(point []float64) float64

// Solution 一维极小化结果
type Solution struct {
	X           float64 // 极小点
	Value       float64 // 极小点处函数值
	Iterations  int     // 迭代次数
	Evaluations int     // 目标函数求值次数
}

// MultiSolution 多元极小化结果
type MultiSolution struct {
	X           []float64 // 极小点，与初值维度相同
	Value       float64   // 极小点处函数值
	Iterations  int       // 迭代次数
	Evaluations int       // 目标函数求值次数
}

// Option 配置极小化过程的选项
type Option func(*config)

type config struct {
	tolerance     float64 // 收敛容差
	maxIterations int     // 迭代上限
}

const (
	// _DEFAULT_TOLERANCE 为默认收敛容差
	_DEFAULT_TOLERANCE = 1e-8
	// _DEFAULT_MAX_ITERATION 为默认迭代上限
	_DEFAULT_MAX_ITERATION = 1000
	// _MACHINE_EPSILON 为 float64 的机器精度（2^-52）
	_MACHINE_EPSILON = 2.220446049250313e-16
	// _GOLDEN_RATIO 为黄金分割的比例 (3-√5)/2
	_GOLDEN_RATIO = 0.3819660112501051
	// _REFLECTION / _EXPANSION / _CONTRACTION / _SHRINK 为 Nelder-Mead 的单纯形系数
	_REFLECTION  = 1.0
	_EXPANSION   = 2.0
	_CONTRACTION = 0.5
	_SHRINK      = 0.5
	// _SIMPLEX_SCALE 与 _SIMPLEX_OFFSET 用于在初始点附近构造初始单纯形
	_SIMPLEX_SCALE  = 0.05
	_SIMPLEX_OFFSET = 0.00025
)

// WithTolerance 设置收敛容差，必须为正
func WithTolerance(tolerance float64) Option {
	return func(own *config) { own.tolerance = tolerance }
}

// WithMaxIterations 设置最大迭代次数，必须为正
func WithMaxIterations(iterations int) Option {
	return func(own *config) { own.maxIterations = iterations }
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

	if own.maxIterations <= 0 {
		return ErrInvalidIterations
	}

	return nil
}

// evaluate 求一元目标函数值并校验有限性
func evaluate(function Function, x float64) (float64, error) {
	value := function(x)
	if !isFinite(value) {
		return 0, wrapError(ErrNonFinite, "function at %g", x)
	}

	return value, nil
}

// evaluateMulti 求多元目标函数值并校验有限性，入参以副本传入
func evaluateMulti(function MultiFunction, point []float64) (float64, error) {
	value := function(append([]float64(nil), point...))
	if !isFinite(value) {
		return 0, wrapError(ErrNonFinite, "function at %v", point)
	}

	return value, nil
}

// isFinite 判断浮点数是否为有限值
func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
