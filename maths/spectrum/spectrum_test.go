package spectrum

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// requireClose 断言两个浮点值在给定容差内相等
func requireClose(t *testing.T, label string, got, want, tolerance float64) {
	t.Helper()

	if math.IsNaN(got) || math.Abs(got-want) > tolerance {
		t.Fatalf("%s = %.15g, want %.15g (±%g)", label, got, want, tolerance)
	}
}

func TestErrorPrefix(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"空输入", ErrEmptyInput},
		{"非法长度", ErrInvalidLength},
		{"非法窗函数", ErrInvalidWindow},
		{"非法采样率", ErrInvalidSampleRate},
		{"非有限值", ErrNonFinite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.HasPrefix(tt.err.Error(), _ERROR_PREFIX) {
				t.Fatalf("%v 缺少错误前缀", tt.err)
			}
		})
	}
}

func TestCalculateSpectrumCosine(t *testing.T) {
	const size = 64
	const peak = 8

	values := make([]float64, size)
	for index := range values {
		values[index] = math.Cos(2 * math.Pi * peak * float64(index) / size)
	}

	tests := []struct {
		name   string
		window Window
	}{
		{"矩形窗", WINDOW_RECTANGULAR},
		{"Hann 窗", WINDOW_HANN},
		{"Hamming 窗", WINDOW_HAMMING},
		{"Blackman 窗", WINDOW_BLACKMAN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spectrum, err := CalculateSpectrum(values, size, tt.window)
			if err != nil {
				t.Fatalf("err = %v", err)
			}

			if len(spectrum.Frequencies) != size/2+1 || len(spectrum.Amplitudes) != size/2+1 {
				t.Fatalf("谱长度 = %d/%d, want %d", len(spectrum.Frequencies), len(spectrum.Amplitudes), size/2+1)
			}

			requireClose(t, "分辨率", spectrum.Resolution, 1, 1e-12)
			requireClose(t, "峰值频率", spectrum.Frequencies[peak], peak, 1e-12)
			requireClose(t, "峰值幅度", spectrum.Amplitudes[peak], 1, 1e-12)

			if spectrum.Amplitudes[0] > 1e-12 {
				t.Fatalf("直流分量 = %g, want ~0", spectrum.Amplitudes[0])
			}
		})
	}
}

func TestCalculateSpectrumDC(t *testing.T) {
	spectrum, err := CalculateSpectrum([]float64{1, 1, 1, 1}, 4, WINDOW_RECTANGULAR)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "直流幅度", spectrum.Amplitudes[0], 1, 1e-12)

	for index := 1; index < len(spectrum.Amplitudes); index++ {
		if spectrum.Amplitudes[index] > 1e-12 {
			t.Fatalf("频点 %d 幅度 = %g, want ~0", index, spectrum.Amplitudes[index])
		}
	}
}

func TestCalculateSpectrumNyquist(t *testing.T) {
	// 交替序列的能量全部落在 Nyquist 频点，单边谱不应翻倍
	spectrum, err := CalculateSpectrum([]float64{1, -1, 1, -1}, 4, WINDOW_RECTANGULAR)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "Nyquist 幅度", spectrum.Amplitudes[2], 1, 1e-12)

	if spectrum.Amplitudes[0] > 1e-12 {
		t.Fatalf("直流分量 = %g, want ~0", spectrum.Amplitudes[0])
	}
}

func TestCalculateSpectrumErrors(t *testing.T) {
	tests := []struct {
		name       string
		values     []float64
		sampleRate float64
		window     Window
		want       error
	}{
		{"空输入", nil, 1, WINDOW_HANN, ErrEmptyInput},
		{"长度非 2 的幂", []float64{1, 2, 3}, 1, WINDOW_HANN, ErrInvalidLength},
		{"零采样率", []float64{1, 2}, 0, WINDOW_HANN, ErrInvalidSampleRate},
		{"负采样率", []float64{1, 2}, -1, WINDOW_HANN, ErrInvalidSampleRate},
		{"NaN 采样率", []float64{1, 2}, math.NaN(), WINDOW_HANN, ErrInvalidSampleRate},
		{"非法窗", []float64{1, 2}, 1, Window(99), ErrInvalidWindow},
		{"含 NaN", []float64{1, math.NaN()}, 1, WINDOW_HANN, ErrNonFinite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateSpectrum(tt.values, tt.sampleRate, tt.window)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCalculateSpectrumDoesNotModifyInput(t *testing.T) {
	values := []float64{1, 2, 3, 4}

	if _, err := CalculateSpectrum(values, 4, WINDOW_HANN); err != nil {
		t.Fatalf("err = %v", err)
	}

	want := []float64{1, 2, 3, 4}
	for index := range want {
		if values[index] != want[index] {
			t.Fatalf("输入被改写：%v", values)
		}
	}
}
