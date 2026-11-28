package filter

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateHampel(t *testing.T) {
	values := []float64{1, 2, 3, 100, 5, 6, 7}

	series, err := CalculateHampel(values, 3, 3)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	want := []float64{1, 2, 3, 5, 5, 6, 7}
	for index := range want {
		requireClose(t, "滤波值", series.Values[index], want[index], 1e-12)
	}

	if series.Replaced != 1 {
		t.Fatalf("Replaced = %d, want 1", series.Replaced)
	}
}

func TestCalculateHampelConstantScale(t *testing.T) {
	// 邻域内 MAD 为 0 时尺度退化为 0，任何偏离中位数的点都会被替换
	values := []float64{5, 5, 5, 50, 5, 5, 5}

	series, err := CalculateHampel(values, 3, 3)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	for _, value := range series.Values {
		requireClose(t, "滤波值", value, 5, 1e-12)
	}

	if series.Replaced != 1 {
		t.Fatalf("Replaced = %d, want 1", series.Replaced)
	}
}

func TestCalculateHampelKeepsEdge(t *testing.T) {
	values := []float64{100, 1, 1, 1, 1, 1, 100}

	series, err := CalculateHampel(values, 5, 3)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if series.Values[0] != 100 || series.Values[len(values)-1] != 100 {
		t.Fatalf("边界值被改写：%v", series.Values)
	}
}

func TestCalculateHampelErrors(t *testing.T) {
	values := []float64{1, 2, 3}

	tests := []struct {
		name      string
		window    int
		threshold float64
		want      error
	}{
		{"偶数窗口", 2, 3, ErrInvalidWindow},
		{"窗口过小", 1, 3, ErrInvalidWindow},
		{"窗口超过长度", 5, 3, ErrInvalidWindow},
		{"零阈值", 3, 0, ErrInvalidThreshold},
		{"负阈值", 3, -1, ErrInvalidThreshold},
		{"NaN 阈值", 3, math.NaN(), ErrInvalidThreshold},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateHampel(values, tt.window, tt.threshold)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}
