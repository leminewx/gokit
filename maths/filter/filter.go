// Package filter 提供一维序列的平滑与去野值滤波
//   - 滑动平均：局部窗口取算术平均，平滑高频噪声；
//   - 中值滤波：局部窗口取中位数，抑制脉冲噪声并保留阶跃边缘；
//   - Hampel 滤波：以中位数与绝对偏差判定野值，用局部中位数替换；
//   - 指数平滑：一阶低通，只依赖历史样本，可在线更新。
//
// 设计目标：
//   - 零第三方依赖，纯标准库实现；
//   - 输出长度与输入一致，不丢样本；
//   - 边界策略由各函数显式说明，不做隐式外推；
//   - 无全局可变状态，所有函数可并发调用。
//
// 已知限制：
//   - 滑动平均在两端收缩窗口，边缘平滑强度弱于中段；
//   - 中值滤波与 Hampel 滤波在两端不足半窗处保持原值；
//   - 指数平滑是因果滤波，存在相位滞后，需要零相位对齐时应在正反两个方向各做一次。
package filter

import "math"

// Series 滤波结果
type Series struct {
	Values   []float64 // 滤波后的序列，长度与输入一致
	Replaced int       // 被修正的样本数，仅 Hampel 统计，其余函数恒为 0
}

const (
	// _MIN_WINDOW 为中值滤波与 Hampel 滤波的最小窗口
	_MIN_WINDOW = 3
	// _MAD_SCALE 把中位绝对偏差换算为正态分布标准差的一致估计
	_MAD_SCALE = 1.4826
)

// validateValues 校验序列非空且全部有限
func validateValues(values []float64) error {
	if len(values) == 0 {
		return ErrEmptyInput
	}

	for index, value := range values {
		if !isFinite(value) {
			return wrapError(ErrNonFinite, "values[%d]", index)
		}
	}

	return nil
}

// validateOddWindow 校验窗口为不小于 3 的奇数且不超过序列长度
func validateOddWindow(values []float64, window int) error {
	if window < _MIN_WINDOW || window > len(values) || window%2 == 0 {
		return wrapError(ErrInvalidWindow, "%d over %d", window, len(values))
	}

	return nil
}

// isFinite 判断浮点数是否为有限值
func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
