package spectrum

import (
	"errors"
	"testing"
)

func TestCalculateWindow(t *testing.T) {
	tests := []struct {
		name   string
		window Window
		length int
		want   []float64
	}{
		{"矩形窗", WINDOW_RECTANGULAR, 4, []float64{1, 1, 1, 1}},
		{"Hann 窗", WINDOW_HANN, 4, []float64{0, 0.5, 1, 0.5}},
		{"Hamming 窗", WINDOW_HAMMING, 4, []float64{0.08, 0.54, 1, 0.54}},
		{"Blackman 窗", WINDOW_BLACKMAN, 4, []float64{0, 0.34, 1, 0.34}},
		{"长度为 1", WINDOW_HANN, 1, []float64{1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coefficients, err := CalculateWindow(tt.window, tt.length)
			if err != nil {
				t.Fatalf("err = %v", err)
			}

			if len(coefficients) != tt.length {
				t.Fatalf("len = %d, want %d", len(coefficients), tt.length)
			}

			for index := range tt.want {
				requireClose(t, "系数", coefficients[index], tt.want[index], 1e-12)
			}
		})
	}
}

func TestCalculateWindowErrors(t *testing.T) {
	tests := []struct {
		name   string
		window Window
		length int
		want   error
	}{
		{"零长度", WINDOW_HANN, 0, ErrInvalidLength},
		{"负长度", WINDOW_HANN, -4, ErrInvalidLength},
		{"负窗编号", Window(-1), 4, ErrInvalidWindow},
		{"未知窗", Window(99), 4, ErrInvalidWindow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateWindow(tt.window, tt.length)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}
