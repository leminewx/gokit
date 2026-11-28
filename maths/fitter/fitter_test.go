package fitter

import (
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
		{"非法阶数", ErrInvalidDegree},
		{"样本不足", ErrTooFewSamples},
		{"负权重", ErrNegativeWeight},
		{"非有限值", ErrNonFinite},
		{"奇异矩阵", ErrSingularSystem},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.HasPrefix(tt.err.Error(), _ERROR_PREFIX) {
				t.Fatalf("%v 缺少错误前缀", tt.err)
			}
		})
	}
}

func TestFitCalculateValue(t *testing.T) {
	fit := Fit{Coefficients: []float64{1, 2, 3}}

	tests := []struct {
		x    float64
		want float64
	}{
		{0, 1},
		{1, 6},
		{2, 17},
		{-2, 9},
	}

	for _, tt := range tests {
		requireClose(t, "拟合值", fit.CalculateValue(tt.x), tt.want, 1e-12)
	}
}
