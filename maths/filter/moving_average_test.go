package filter

import (
	"errors"
	"testing"
)

func TestCalculateMovingAverage(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}

	tests := []struct {
		name   string
		window int
		want   []float64
	}{
		{"窗口 1", 1, []float64{1, 2, 3, 4, 5}},
		{"窗口 3", 3, []float64{1.5, 2, 3, 4, 4.5}},
		{"窗口 5", 5, []float64{2, 2.5, 3, 3.5, 4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			series, err := CalculateMovingAverage(values, tt.window)
			if err != nil {
				t.Fatalf("err = %v", err)
			}

			if len(series.Values) != len(values) {
				t.Fatalf("len = %d, want %d", len(series.Values), len(values))
			}

			for index := range tt.want {
				requireClose(t, "平滑值", series.Values[index], tt.want[index], 1e-12)
			}
		})
	}
}

func TestCalculateMovingAverageSpreadsSpike(t *testing.T) {
	values := []float64{0, 0, 9, 0, 0}

	series, err := CalculateMovingAverage(values, 3)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	want := []float64{0, 3, 3, 3, 0}
	for index := range want {
		requireClose(t, "平滑值", series.Values[index], want[index], 1e-12)
	}
}

func TestCalculateMovingAverageErrors(t *testing.T) {
	values := []float64{1, 2, 3}

	tests := []struct {
		name   string
		window int
	}{
		{"零窗口", 0},
		{"负窗口", -2},
		{"窗口超过长度", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateMovingAverage(values, tt.window)
			if !errors.Is(err, ErrInvalidWindow) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidWindow)
			}
		})
	}
}

func TestCalculateMovingAverageNoAliasing(t *testing.T) {
	values := []float64{1, 2, 3}

	series, err := CalculateMovingAverage(values, 2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	series.Values[0] = 100
	if values[0] != 1 {
		t.Fatalf("结果与输入共享底层数组")
	}
}
