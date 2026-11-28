package activator

// Activator 激活函数接口，定义正向计算和反向求导方法
type Activator interface {
	// Activate 正向传播计算激活值
	Activate(x float64) float64
	// Derive 反向传播计算梯度
	Derive(x float64) float64
	// GetName 获取激活函数名称
	GetName() Name
}

// Name 定义激活函数名称的数据类型
type Name string

const (
	// NAME_RELU ReLU 激活函数
	NAME_RELU Name = "ReLU"
	// NAME_TANH Tanh 激活函数
	NAME_TANH Name = "Tanh"
	// NAME_SIGMOID Sigmoid 激活函数
	NAME_SIGMOID Name = "Sigmoid"
	// NAME_LEAKY_RELU LeakyReLU 激活函数
	NAME_LEAKY_RELU Name = "LeakyReLU"
)

// ActivateBatch 对切片批量应用激活函数
func ActivateBatch(activator Activator, inputs []float64) []float64 {
	outputs := make([]float64, len(inputs))
	for i, x := range inputs {
		outputs[i] = activator.Activate(x)
	}

	return outputs
}
