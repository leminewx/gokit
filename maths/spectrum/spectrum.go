// Package spectrum 提供离散傅里叶变换、窗函数与单边幅度谱
//   - FFT / IFFT：迭代基 2 Cooley-Tukey 算法，序列长度必须为 2 的幂；
//   - 窗函数：矩形、Hann、Hamming、Blackman，采用适合 DFT 的周期形式；
//   - 幅度谱：加窗后做 FFT，给出单边幅度并按窗函数相干增益归一化。
//
// 设计目标：
//   - 零第三方依赖，纯标准库实现；
//   - 统一返回切片或 Spectrum 与 error，不返回裸元组，不 panic；
//   - 输入出现 NaN/Inf 时立即返回 ErrNonFinite；
//   - 无全局可变状态，所有函数可并发调用。
//
// 已知限制：
//   - 只支持 2 的幂长度，任意长度需调用方先补零或截断；
//   - IFFT 的归一化因子为 1/N，与 FFT 互为逆变换；
//   - 幅度谱只适用于实数信号，负数频率分量不返回。
package spectrum

import "math"

// Window 窗函数类型
type Window int

const (
	// WINDOW_RECTANGULAR 为矩形窗，等价于不加窗
	WINDOW_RECTANGULAR Window = iota
	// WINDOW_HANN 为 Hann 窗
	WINDOW_HANN
	// WINDOW_HAMMING 为 Hamming 窗
	WINDOW_HAMMING
	// WINDOW_BLACKMAN 为 Blackman 窗
	WINDOW_BLACKMAN
)

// Spectrum 单边幅度谱
type Spectrum struct {
	Frequencies []float64 // 频率序列，长度 N/2+1，单位与 sampleRate 一致
	Amplitudes  []float64 // 幅度序列，已按窗函数相干增益归一化
	Resolution  float64   // 频率分辨率 sampleRate / N
}

// isFinite 判断浮点数是否为有限值
func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// isPowerOfTwo 判断长度是否为 2 的幂
func isPowerOfTwo(length int) bool {
	return length > 0 && length&(length-1) == 0
}
