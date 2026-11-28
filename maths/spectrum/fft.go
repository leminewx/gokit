package spectrum

import "math"

// CalculateFFT 计算实数序列的离散傅里叶变换
// 序列长度必须为 2 的幂，返回长度为 N 的复数频谱，未做归一化。
func CalculateFFT(values []float64) ([]complex128, error) {
	if len(values) == 0 {
		return nil, ErrEmptyInput
	}

	if !isPowerOfTwo(len(values)) {
		return nil, wrapError(ErrInvalidLength, "%d, want a power of two", len(values))
	}

	data := make([]complex128, len(values))
	for index, value := range values {
		if !isFinite(value) {
			return nil, wrapError(ErrNonFinite, "values[%d]", index)
		}

		data[index] = complex(value, 0)
	}

	transform(data)
	return data, nil
}

// CalculateIFFT 计算复数序列的逆离散傅里叶变换
// 序列长度必须为 2 的幂，结果按 1/N 归一化，与 CalculateFFT 互为逆变换。
func CalculateIFFT(values []complex128) ([]complex128, error) {
	if len(values) == 0 {
		return nil, ErrEmptyInput
	}

	if !isPowerOfTwo(len(values)) {
		return nil, wrapError(ErrInvalidLength, "%d, want a power of two", len(values))
	}

	data := make([]complex128, len(values))
	for index, value := range values {
		if !isFinite(real(value)) || !isFinite(imag(value)) {
			return nil, wrapError(ErrNonFinite, "values[%d]", index)
		}

		// 逆变换等于先取共轭、再正向变换、最后取共轭并除以 N
		data[index] = complex(real(value), -imag(value))
	}

	transform(data)
	scale := 1 / float64(len(data))
	for index := range data {
		data[index] = complex(real(data[index])*scale, -imag(data[index])*scale)
	}

	return data, nil
}

// transform 就地执行基 2 快速傅里叶变换
func transform(data []complex128) {
	size := len(data)

	// 位反转置换
	for index, reversed := 1, 0; index < size; index++ {
		bit := size >> 1
		for ; reversed&bit != 0; bit >>= 1 {
			reversed ^= bit
		}

		reversed ^= bit
		if index < reversed {
			data[index], data[reversed] = data[reversed], data[index]
		}
	}

	for length := 2; length <= size; length <<= 1 {
		angle := -2 * math.Pi / float64(length)
		root := complex(math.Cos(angle), math.Sin(angle))
		half := length / 2

		for start := 0; start < size; start += length {
			factor := complex(1, 0)
			for offset := 0; offset < half; offset++ {
				even := data[start+offset]
				odd := data[start+offset+half] * factor

				data[start+offset] = even + odd
				data[start+offset+half] = even - odd
				factor *= root
			}
		}
	}
}
