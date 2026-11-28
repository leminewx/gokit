// Package quadrature 提供一元函数在有限区间上的数值积分方法
//   - Romberg：梯形序列 + Richardson 外推，收敛快，自带误差估计；
//   - 复化 Newton-Cotes：梯形 / Simpson / Simpson 3/8 / Boole，固定分区数，实现简单；
//   - 自适应 Newton-Cotes：按局部误差估计递归细分，适合局部变化剧烈的被积函数；
//   - Gauss-Legendre：2~5 点高斯节点，n 点对 2n-1 次多项式精确，等量求值下精度最高。
//
// 设计目标：
//   - 零第三方依赖，节点与权重全部按公开标准值内置；
//   - 统一返回 Result 与 error，不返回裸元组，不 panic；
//   - 被积函数返回 NaN/Inf 时立即以 ErrNonFinite 终止；
//   - 误差估计始终满足 ErrorEstimate >= 0，0 表示该算法默认不提供估计。
//
// 误差估计约定：
//   - Romberg 与自适应 Newton-Cotes 天然产生估计值，始终填充；
//   - 复化 Newton-Cotes 与 Gauss-Legendre 默认不估计（填 0），
//     传入 WithErrorEstimation() 后以约两倍函数求值换取估计值。
//   - ErrorEstimate 是算法内部的收敛指示，不是严格误差上界。
//
// 典型用法：
//
//	result, err := quadrature.CalculateRomberg(func(x float64) float64 { return math.Sin(x) }, 0, math.Pi)
//	if err == nil {
//	    fmt.Println(result.Value) // 2
//	}
//
// 已知限制：
//   - 只支持有限区间与端点有限的被积函数，奇点积分需先做变量替换；
//   - 不处理振荡积分的特殊加速；
//   - 自适应方法的最坏求值次数随深度指数增长，默认深度已限制在 16 层。
package quadrature

import "math"

// Function 一元被积函数
type Function func(x float64) float64

// Result 积分结果
type Result struct {
	Value         float64 // 积分近似值
	Iterations    int     // Romberg 的序列行数；自适应方法的细分叶子数；其余方法固定为 1
	ErrorEstimate float64 // 误差指示，>= 0，0 表示未提供
}

// Option 配置积分过程的选项
type Option func(*config)

type config struct {
	tolerance       float64 // 收敛容差
	maxIterations   int     // Romberg 序列上限或自适应深度上限
	panels          int     // 复化与 Gauss 的分区数
	order           int     // Newton-Cotes 节点数或 Gauss 节点数
	errorEstimation bool    // 是否对无天然估计的方法额外计算估计值
}

const (
	// _DEFAULT_TOLERANCE 为默认收敛容差
	_DEFAULT_TOLERANCE = 1e-10
	// _DEFAULT_MAX_ITERATION 为默认 Romberg 行数上限与自适应深度上限
	_DEFAULT_MAX_ITERATION = 16
	// _DEFAULT_PANEL 为默认分区数
	_DEFAULT_PANEL = 16
	// _DEFAULT_ORDER 为默认节点数
	_DEFAULT_ORDER = 3
	// _MIN_ORDER 与 _MAX_ORDER 界定支持的节点数范围
	_MIN_ORDER = 2
	_MAX_ORDER = 5
)

// WithTolerance 设置收敛容差，必须为正
func WithTolerance(tolerance float64) Option {
	return func(own *config) { own.tolerance = tolerance }
}

// WithMaxIterations 设置 Romberg 序列行数上限或自适应深度上限，必须为正
func WithMaxIterations(iterations int) Option {
	return func(own *config) { own.maxIterations = iterations }
}

// WithPanels 设置复化 Newton-Cotes 与 Gauss-Legendre 的分区数，必须为正
func WithPanels(panels int) Option {
	return func(own *config) { own.panels = panels }
}

// WithOrder 设置 Newton-Cotes 的节点数（2~5）或 Gauss-Legendre 的节点数（2~5）
func WithOrder(order int) Option {
	return func(own *config) { own.order = order }
}

// WithErrorEstimation 为无天然误差估计的方法启用额外估计（约两倍函数求值）
func WithErrorEstimation() Option {
	return func(own *config) { own.errorEstimation = true }
}

// newConfig 应用选项并填充默认值
func newConfig(options []Option) *config {
	own := &config{
		tolerance:     _DEFAULT_TOLERANCE,
		maxIterations: _DEFAULT_MAX_ITERATION,
		panels:        _DEFAULT_PANEL,
		order:         _DEFAULT_ORDER,
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

	if own.panels <= 0 {
		return ErrInvalidPanels
	}

	if own.order < _MIN_ORDER || own.order > _MAX_ORDER {
		return wrapError(ErrInvalidOrder, "%d, want %d..%d", own.order, _MIN_ORDER, _MAX_ORDER)
	}

	return nil
}

// validateInterval 校验积分区间，要求 a < b 且两端有限
func validateInterval(a, b float64) error {
	if !isFinite(a) || !isFinite(b) || a >= b {
		return wrapError(ErrInvalidInterval, "[%g, %g]", a, b)
	}

	return nil
}

// isFinite 判断浮点数是否为有限值
func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
