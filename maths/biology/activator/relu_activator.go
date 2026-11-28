package activator

// ReLU 激活函数
//   - 表达式：relu(x) = max(0, x)
type ReLU struct{}

var _ Activator = (*ReLU)(nil)

// NewReLU 创建 ReLU 激活函数
func NewReLU() *ReLU { return &ReLU{} }

// Activate 正向激活
func (own *ReLU) Activate(x float64) float64 {
	if x > 0 {
		return x
	}

	return 0
}

// Derive 反向求导：x > 0 时为 1，否则为 0
func (own *ReLU) Derive(x float64) float64 {
	if x > 0 {
		return 1
	}

	return 0
}

func (own *ReLU) GetName() Name {
	return NAME_RELU
}
