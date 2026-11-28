package activator

import (
	"math"
)

// Tanh 激活函数
//   - 表达式：tanh(x) = (exp(x) - exp(-x)) / (exp(x) + exp(-x))
type Tanh struct{}

var _ Activator = (*Tanh)(nil)

func NewTanh() *Tanh { return &Tanh{} }

// Activate 正向激活
func (own *Tanh) Activate(x float64) float64 {
	return math.Tanh(x)
}

// Derive 反向求导：1 - tanh(x)^2
func (own *Tanh) Derive(x float64) float64 {
	y := own.Activate(x)
	return 1 - y*y
}

func (own *Tanh) GetName() Name {
	return NAME_TANH
}
