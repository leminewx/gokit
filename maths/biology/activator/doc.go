// Package activator 提供神经网络常用的逐元素激活函数（ReLU、LeakyReLU、Sigmoid、Tanh）
// 与多分类归一化 Softmax，以及 Softmax 的反向梯度。
//
// 激活函数均为零值可用、无状态的纯函数，可并发调用；CalculateSoftmax 对空输入返回 nil。
// 输出不做数值截断，调用方应保证输入有限，否则结果可能为 NaN。
package activator
