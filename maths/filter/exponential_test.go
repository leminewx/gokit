package filter

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateExponential(t *testing.T) {
	values := []float64{1, 2, 3}

	series, err := CalculateExponential(values, 0.5)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	want := []float64{1, 1.5, 2.25}
	for index := range want {
		requireClose(t, "平滑值", series.Values[index], want[index], 1e-12)
	}
}

func TestCalculateExponentialAlphaOne(t *testing.T) {
	values := []float64{4, 1, 7, 2}

	series, err := CalculateExponential(values, 1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	for index := range values {
		requireClose(t, "平滑值", series.Values[index], values[index], 1e-12)
	}
}

func TestCalculateExponentialAttenuatesNoise(t *testing.T) {
	values := []float64{0, 10, 0, 10, 0, 10, 0}

	series, err := CalculateExponential(values, 0.2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	for index := 1; index < len(values); index++ {
		if math.Abs(series.Values[index]-series.Values[index-1]) >= 10 {
			t.Fatalf("第 %d 个样本跳变过大：%g", index, series.Values[index])
		}
	}
}

func TestCalculateExponentialErrors(t *testing.T) {
	values := []float64{1, 2, 3}

	tests := []struct {
		name  string
		alpha float64
	}{
		{"零系数", 0},
		{"负系数", -0.5},
		{"超过 1", 1.5},
		{"NaN 系数", math.NaN()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateExponential(values, tt.alpha)
			if !errors.Is(err, ErrInvalidAlpha) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidAlpha)
			}
		})
	}
}
