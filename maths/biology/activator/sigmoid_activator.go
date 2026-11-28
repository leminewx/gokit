package activator

import (
	"math"
)

// Sigmoid 激活函数
//   - 表达式：sigmoid(x) = 1 / (1 + exp(-x))
type Sigmoid struct{}

var _ Activator = (*Sigmoid)(nil)

func NewSigmoid() *Sigmoid { return &Sigmoid{} }

// Activate 正向激活
func (own *Sigmoid) Activate(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// Derive 反向求导：sigmoid(x) * (1 - sigmoid(x))
func (own *Sigmoid) Derive(x float64) float64 {
	y := own.Activate(x)
	return y * (1 - y)
}

func (own *Sigmoid) GetName() Name {
	return NAME_SIGMOID
}
