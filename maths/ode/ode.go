// Package ode 提供常微分方程初值问题的数值解法
//   - 四阶 Runge-Kutta：固定步长，四阶精度，无内置误差控制；
//   - Runge-Kutta-Fehlberg 4(5)：嵌入四/五阶对，按局部误差自动调整步长。
//
// 设计目标：
//   - 零第三方依赖，纯标准库实现；
//   - 统一返回 Trajectory 与 error，不返回裸元组，不 panic；
//   - 向量状态以 []float64 表示，右端函数必须返回与输入等长的切片；
//   - 求值结果出现 NaN/Inf 或长度变化时立即终止；
//   - 入参切片不被改写，Trajectory 中每一行数据互不共享底层数组。
//
// 典型用法：
//
//	trajectory, err := ode.CalculateRungeKutta4(
//	    func(x float64, y []float64) []float64 { return []float64{y[0]} },
//	    []float64{1}, 0, 1, 0.01)
//	if err == nil {
//	    fmt.Println(trajectory.Ys[len(trajectory.Ys)-1][0]) // ≈ e
//	}
//
// 已知限制：
//   - 只处理一阶显式方程组的初值问题，高阶方程需先降阶为一阶方程组；
//   - 固定步长不感知局部误差，刚性方程应改用隐式方法；
//   - Fehlberg 的误差估计是局部截断误差的指示，不构成严格误差上界；
//   - 不提供事件检测、稠密输出与反向积分。
package ode

import "math"

// Function 右端函数，返回 dy/dx
type Function func(x float64, y []float64) []float64

// Trajectory 积分轨迹
type Trajectory struct {
	Xs       []float64   // 自变量序列，首元素为起点，末元素为终点
	Ys       [][]float64 // 状态序列，Ys[i] 对应 Xs[i]，各行互不共享底层数组
	Steps    int         // 接受的步数
	Rejected int         // 被拒绝并重试的步数
}

// Option 配置积分过程的选项
type Option func(*config)

type config struct {
	tolerance     float64 // 局部误差容差
	initialStep   float64 // 初始步长，0 表示按区间跨度自动选取
	minStep       float64 // 步长下界，0 表示按区间跨度自动选取
	maxStep       float64 // 步长上界，0 表示按区间跨度自动选取
	maxIterations int     // 步数上限，含被拒绝的尝试
}

const (
	// _DEFAULT_TOLERANCE 为默认局部误差容差
	_DEFAULT_TOLERANCE = 1e-8
	// _DEFAULT_MAX_ITERATION 为默认步数上限
	_DEFAULT_MAX_ITERATION = 100000
	// _DEFAULT_INITIAL_DIVISION 决定自动初始步长，即区间跨度的等分数
	_DEFAULT_INITIAL_DIVISION = 100
	// _MIN_STEP_RATIO 决定自动步长下界，即区间跨度的比例
	_MIN_STEP_RATIO = 1e-12
	// _MAX_STEP_COUNT 为固定步长允许的最大步数
	_MAX_STEP_COUNT = 10000000
	// _SAFETY_FACTOR 为自适应步长因子的保守系数
	_SAFETY_FACTOR = 0.9
	// _MIN_STEP_SHRINK 与 _MAX_STEP_GROWTH 界定单次步长调整幅度
	_MIN_STEP_SHRINK = 0.2
	_MAX_STEP_GROWTH = 5
	// _ERROR_EXPONENT 为步长因子指数，等于 1/(局部误差阶+1)
	_ERROR_EXPONENT = 0.2
	// _ROUNDING_RATIO 用于吸收区间跨度与步长比值中的浮点末位误差
	_ROUNDING_RATIO = 1e-12
)

// WithTolerance 设置局部误差容差，必须为正
func WithTolerance(tolerance float64) Option {
	return func(own *config) { own.tolerance = tolerance }
}

// WithInitialStep 设置初始步长，必须为正，0 表示按区间跨度自动选取
func WithInitialStep(step float64) Option {
	return func(own *config) { own.initialStep = step }
}

// WithStepRange 设置自适应步长的下界与上界，必须满足 0 <= minimum <= maximum
func WithStepRange(minimum, maximum float64) Option {
	return func(own *config) {
		own.minStep = minimum
		own.maxStep = maximum
	}
}

// WithMaxIterations 设置步数上限（含被拒绝的尝试），必须为正
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

// validate 校验自适应选项
func (own *config) validate() error {
	if !isFinite(own.tolerance) || own.tolerance <= 0 {
		return ErrInvalidTolerance
	}

	if own.maxIterations <= 0 {
		return ErrInvalidIterations
	}

	if !isFinite(own.initialStep) || own.initialStep < 0 {
		return wrapError(ErrInvalidStep, "%g", own.initialStep)
	}

	if !isFinite(own.minStep) || own.minStep < 0 || !isFinite(own.maxStep) || own.maxStep < 0 {
		return wrapError(ErrInvalidStepRange, "[%g, %g]", own.minStep, own.maxStep)
	}

	if own.minStep > 0 && own.maxStep > 0 && own.minStep > own.maxStep {
		return wrapError(ErrInvalidStepRange, "[%g, %g]", own.minStep, own.maxStep)
	}

	return nil
}

// resolve 依据区间跨度补齐步长缺省值，并把初始步长限制在合法范围内
func (own *config) resolve(span float64) {
	if own.minStep <= 0 {
		own.minStep = span * _MIN_STEP_RATIO
	}

	if own.minStep <= 0 {
		own.minStep = math.SmallestNonzeroFloat64
	}

	if own.maxStep <= 0 {
		own.maxStep = span
	}

	if own.minStep > own.maxStep {
		own.minStep = own.maxStep
	}

	if own.initialStep <= 0 {
		own.initialStep = span / _DEFAULT_INITIAL_DIVISION
	}

	own.initialStep = math.Max(own.minStep, math.Min(own.maxStep, own.initialStep))
}

// calculateStep 依据局部误差估计给出下一步步长
func (own *config) calculateStep(width, estimate float64) float64 {
	factor := float64(_MAX_STEP_GROWTH)
	if estimate > 0 {
		factor = _SAFETY_FACTOR * math.Pow(own.tolerance/estimate, _ERROR_EXPONENT)
		factor = math.Max(_MIN_STEP_SHRINK, math.Min(float64(_MAX_STEP_GROWTH), factor))
	}

	return math.Max(own.minStep, math.Min(own.maxStep, width*factor))
}

// validateRange 校验积分区间，要求 start < stop 且两端有限
func validateRange(start, stop float64) error {
	if !isFinite(start) || !isFinite(stop) || start >= stop {
		return wrapError(ErrInvalidRange, "[%g, %g]", start, stop)
	}

	return nil
}

// validateState 校验初始状态非空且全部有限
func validateState(initial []float64) error {
	if len(initial) == 0 {
		return ErrEmptyState
	}

	for index, value := range initial {
		if !isFinite(value) {
			return wrapError(ErrNonFinite, "initial[%d]", index)
		}
	}

	return nil
}

// validateStep 校验单步步长
func validateStep(step float64) error {
	if !isFinite(step) || step <= 0 {
		return wrapError(ErrInvalidStep, "%g", step)
	}

	return nil
}

// stage 表示右端函数的一次求值及其在线性组合中的系数
type stage struct {
	values      []float64
	coefficient float64
}

// calculateCombination 计算 state + step * Σ coefficient*values
func calculateCombination(state []float64, step float64, stages ...stage) []float64 {
	result := make([]float64, len(state))
	copy(result, state)

	for _, item := range stages {
		for index := range result {
			result[index] += step * item.coefficient * item.values[index]
		}
	}

	return result
}

// evaluate 求值右端函数，并校验返回长度与有限性
// 传入副本以避免调用方改写内部状态
func evaluate(function Function, x float64, state []float64, size int) ([]float64, error) {
	values := function(x, cloneValues(state))

	if len(values) != size {
		return nil, wrapError(ErrStateLengthChanged, "got %d, want %d at x = %g", len(values), size, x)
	}

	for index, value := range values {
		if !isFinite(value) {
			return nil, wrapError(ErrNonFinite, "derivative[%d] at x = %g", index, x)
		}
	}

	return values, nil
}

// cloneValues 复制一维向量
func cloneValues(values []float64) []float64 {
	cloned := make([]float64, len(values))
	copy(cloned, values)

	return cloned
}

// isFinite 判断浮点数是否为有限值
func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
