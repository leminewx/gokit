// Package differentiator 提供采样序列与解析函数的一阶、二阶数值微分
//   - 采样序列：非等距三点公式，左右两侧模板同时可用时取平均并给出差异估计；
//   - 解析函数：五点模板 + Richardson 外推，一阶与二阶均达六阶精度；
//   - Savitzky-Golay：滑动窗口局部多项式拟合，一次给出平滑值与导数序列。
//
// 设计目标：
//   - 零第三方依赖，纯标准库实现；
//   - 统一返回 Derivative / Series 与 error，不返回裸元组，不 panic；
//   - ErrorEstimate 是两次估计之差的指示，不是严格误差上界，0 表示该路径不提供；
//   - 无全局可变状态，所有函数与方法可并发调用。
//
// 已知限制：
//   - 数据含噪声时差分会被噪声主导，应先滤波或改用 CalculateSavitzkyGolay；
//   - Savitzky-Golay 只处理等距样本，导数按单位间隔给出，实际间隔 dx 需自行除以 dx；
//   - 采样序列一律拒绝外推，求值点超出区间返回 ErrOutOfRange。
//
// 典型用法：
//
//	sampler, err := differentiator.NewSampler(xs, ys)
//	if err == nil {
//	    derivative, err := sampler.CalculateDerivative(0.5)
//	    fmt.Println(derivative.Value, derivative.ErrorEstimate)
//	}
package differentiator

import "math"

// Function 一元解析函数
type Function func(x float64) float64

// Derivative 微分结果
type Derivative struct {
	Value         float64 // 导数值
	ErrorEstimate float64 // 误差指示，>= 0，0 表示未提供
	Window        int     // 参与计算的样本点数或模板宽度
}

// Series Savitzky-Golay 处理结果
type Series struct {
	Values      []float64 // 平滑值，两端不足半窗处保持原值
	Derivatives []float64 // 一阶导数，两端不足半窗处为 0，间隔按 1 计算
	Window      int       // 窗口长度（奇数）
	Order       int       // 局部多项式阶数
}

const (
	// _MACHINE_EPSILON 为 float64 的机器精度（2^-52）
	_MACHINE_EPSILON = 2.220446049250313e-16
	// _MIN_SAMPLE 为采样序列的最少点数
	_MIN_SAMPLE = 3
	// _MIN_WINDOW 与 _MAX_WINDOW 界定 Savitzky-Golay 窗口范围
	_MIN_WINDOW = 3
	_MAX_WINDOW = 31
	// _MAX_ORDER 为局部多项式阶数上限
	_MAX_ORDER = 6
	// _SINGULAR_TOLERANCE 用于判定法方程主元是否退化
	_SINGULAR_TOLERANCE = 1e-12
	// _FIVE_POINT_WINDOW 为解析函数五点模板的宽度
	_FIVE_POINT_WINDOW = 5
)

// isFinite 判断浮点数是否为有限值
func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
