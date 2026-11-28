package activator

// LeakyReLU 激活函数
//   - 表达式：leaky_relu(x) = x if x>0 else 0.01*x
type LeakyReLU struct {
	alpha float64 // 负区间的斜率（通常为0.01）
}

var _ Activator = (*LeakyReLU)(nil)

func NewLeakyReLU(alpha float64) *LeakyReLU {
	return &LeakyReLU{alpha: alpha}
}

// Activate 正向激活
func (own *LeakyReLU) Activate(x float64) float64 {
	if x > 0 {
		return x
	}

	return own.alpha * x
}

// Derive 反向求导：x > 0 时为 1，否则为 alpha
func (own *LeakyReLU) Derive(x float64) float64 {
	if x > 0 {
		return 1
	}

	return own.alpha
}

func (own *LeakyReLU) GetName() Name {
	return NAME_LEAKY_RELU
}
