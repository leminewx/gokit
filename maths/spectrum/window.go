package spectrum

import "math"

// CalculateWindow 生成指定类型的周期窗系数
//   - 长度为 N 的窗按 w[n] = f(2πn/N) 生成，适合直接用于 N 点 DFT；
//   - length 必须为正，length 为 1 时返回 [1]，避免退化窗把信号整体置零；
//   - 矩形窗全部为 1，Hann / Hamming / Blackman 的系数见各自定义。
func CalculateWindow(window Window, length int) ([]float64, error) {
	if length <= 0 {
		return nil, wrapError(ErrInvalidLength, "%d", length)
	}

	if window < WINDOW_RECTANGULAR || window > WINDOW_BLACKMAN {
		return nil, wrapError(ErrInvalidWindow, "%d", int(window))
	}

	coefficients := make([]float64, length)
	if window == WINDOW_RECTANGULAR || length == 1 {
		for index := range coefficients {
			coefficients[index] = 1
		}

		return coefficients, nil
	}

	for index := range coefficients {
		phase := 2 * math.Pi * float64(index) / float64(length)
		switch window {
		case WINDOW_HANN:
			coefficients[index] = 0.5 - 0.5*math.Cos(phase)
		case WINDOW_HAMMING:
			coefficients[index] = 0.54 - 0.46*math.Cos(phase)
		case WINDOW_BLACKMAN:
			coefficients[index] = 0.42 - 0.5*math.Cos(phase) + 0.08*math.Cos(2*phase)
		}
	}

	return coefficients, nil
}

// CalculateSpectrum 计算实数序列的单边幅度谱
//   - 先按 window 加窗，再做 FFT，返回 N/2+1 个频点；
//   - 幅度按窗函数相干增益（系数之和）归一化，除直流与 Nyquist 外的频点翻倍；
//   - sampleRate 为正，频率分辨率为 sampleRate / N。
func CalculateSpectrum(values []float64, sampleRate float64, window Window) (Spectrum, error) {
	if len(values) == 0 {
		return Spectrum{}, ErrEmptyInput
	}

	if !isPowerOfTwo(len(values)) {
		return Spectrum{}, wrapError(ErrInvalidLength, "%d, want a power of two", len(values))
	}

	if !isFinite(sampleRate) || sampleRate <= 0 {
		return Spectrum{}, wrapError(ErrInvalidSampleRate, "%g", sampleRate)
	}

	coefficients, err := CalculateWindow(window, len(values))
	if err != nil {
		return Spectrum{}, err
	}

	data := make([]complex128, len(values))
	gain := 0.0
	for index, value := range values {
		if !isFinite(value) {
			return Spectrum{}, wrapError(ErrNonFinite, "values[%d]", index)
		}

		data[index] = complex(value*coefficients[index], 0)
		gain += coefficients[index]
	}

	transform(data)
	resolution := sampleRate / float64(len(values))
	size := len(values)/2 + 1
	spectrum := Spectrum{
		Frequencies: make([]float64, size),
		Amplitudes:  make([]float64, size),
		Resolution:  resolution,
	}

	for index := 0; index < size; index++ {
		spectrum.Frequencies[index] = float64(index) * resolution

		amplitude := math.Hypot(real(data[index]), imag(data[index])) / gain
		if index > 0 && index != len(values)/2 {
			amplitude *= 2
		}

		spectrum.Amplitudes[index] = amplitude
	}

	return spectrum, nil
}
