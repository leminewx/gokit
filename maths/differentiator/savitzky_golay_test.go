package differentiator

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateSavitzkyGolayLinear(t *testing.T) {
	values := make([]float64, 9)
	for index := range values {
		values[index] = 3*float64(index) + 1
	}

	series, err := CalculateSavitzkyGolay(values, 5, 2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if series.Window != 5 || series.Order != 2 {
		t.Fatalf("Window/Order = %d/%d, want 5/2", series.Window, series.Order)
	}

	half := 2
	for index := range values {
		requireClose(t, "平滑值", series.Values[index], values[index], 1e-12)

		if index >= half && index+half < len(values) {
			requireClose(t, "导数值", series.Derivatives[index], 3, 1e-12)
		} else if series.Derivatives[index] != 0 {
			t.Fatalf("边界导数 = %g, want 0", series.Derivatives[index])
		}
	}
}

func TestCalculateSavitzkyGolayQuadratic(t *testing.T) {
	values := make([]float64, 11)
	for index := range values {
		values[index] = float64(index * index)
	}

	series, err := CalculateSavitzkyGolay(values, 7, 2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	half := 3
	for index := half; index+half < len(values); index++ {
		requireClose(t, "平滑值", series.Values[index], values[index], 1e-10)
		requireClose(t, "导数值", series.Derivatives[index], 2*float64(index), 1e-9)
	}
}

func TestCalculateSavitzkyGolaySmoothingOnly(t *testing.T) {
	values := []float64{1, 1, 1, 1, 1, 1, 1}

	series, err := CalculateSavitzkyGolay(values, 3, 0)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	for index := range values {
		requireClose(t, "平滑值", series.Values[index], 1, 1e-12)
		if series.Derivatives[index] != 0 {
			t.Fatalf("导数 = %g, want 0", series.Derivatives[index])
		}
	}
}

func TestCalculateSavitzkyGolayDenoise(t *testing.T) {
	// 叠加交替噪声后，平滑结果的波动应明显小于原始序列
	values := []float64{0, 10, 0, 10, 0, 10, 0, 10, 0}
	series, err := CalculateSavitzkyGolay(values, 5, 2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	spread := 0.0
	for index := 2; index+2 < len(values); index++ {
		spread = math.Max(spread, math.Abs(series.Values[index]-values[index]))
	}

	if spread < 1 {
		t.Fatalf("平滑幅度 %g 过小，疑似未做局部拟合", spread)
	}

	for index := 2; index+2 < len(values); index++ {
		if math.Abs(series.Values[index]) > 10 {
			t.Fatalf("平滑值超出原始值域：%g", series.Values[index])
		}
	}
}

func TestCalculateSavitzkyGolayErrors(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		window int
		order  int
		want   error
	}{
		{"空输入", nil, 3, 1, ErrEmptyInput},
		{"偶数窗口", []float64{1, 2, 3, 4}, 4, 2, ErrInvalidWindow},
		{"窗口过小", []float64{1, 2, 3, 4}, 1, 0, ErrInvalidWindow},
		{"窗口过大", []float64{1, 2, 3}, 33, 2, ErrInvalidWindow},
		{"负阶数", []float64{1, 2, 3, 4, 5}, 5, -1, ErrInvalidOrder},
		{"阶数过高", []float64{1, 2, 3, 4, 5, 6, 7}, 5, 7, ErrInvalidOrder},
		{"阶数不小于窗口", []float64{1, 2, 3, 4, 5}, 5, 5, ErrInvalidOrder},
		{"样本少于窗口", []float64{1, 2, 3, 4}, 5, 2, ErrTooFewSamples},
		{"含 NaN", []float64{1, math.NaN(), 3, 4, 5}, 3, 1, ErrNonFinite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateSavitzkyGolay(tt.values, tt.window, tt.order)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCalculateSavitzkyGolayNoAliasing(t *testing.T) {
	values := []float64{0, 1, 4, 9, 16, 25, 36}

	series, err := CalculateSavitzkyGolay(values, 3, 2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	series.Values[0] = 100
	if values[0] != 0 {
		t.Fatalf("结果与输入共享底层数组")
	}
}
