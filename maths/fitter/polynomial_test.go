package fitter

import (
	"errors"
	"math"
	"testing"
)

func TestCalculatePolynomialLine(t *testing.T) {
	samples := []Sample{
		{X: 0, Y: 1, Weight: 1},
		{X: 1, Y: 3, Weight: 1},
		{X: 2, Y: 5, Weight: 1},
	}

	fit, err := CalculatePolynomial(samples, 1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if len(fit.Coefficients) != 2 {
		t.Fatalf("len = %d, want 2", len(fit.Coefficients))
	}

	requireClose(t, "常数项", fit.Coefficients[0], 1, 1e-10)
	requireClose(t, "一次项", fit.Coefficients[1], 2, 1e-10)
	requireClose(t, "R²", fit.RSquared, 1, 1e-10)

	if fit.WeightedResidual > 1e-20 {
		t.Fatalf("WeightedResidual = %g, want ~0", fit.WeightedResidual)
	}

	if fit.Samples != 3 {
		t.Fatalf("Samples = %d, want 3", fit.Samples)
	}
}

func TestCalculatePolynomialQuadratic(t *testing.T) {
	samples := []Sample{
		{X: 0, Y: 0, Weight: 1},
		{X: 1, Y: 1, Weight: 1},
		{X: 2, Y: 4, Weight: 1},
		{X: 3, Y: 9, Weight: 1},
	}

	fit, err := CalculatePolynomial(samples, 2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	want := []float64{0, 0, 1}
	for index := range want {
		requireClose(t, "系数", fit.Coefficients[index], want[index], 1e-10)
	}

	requireClose(t, "R²", fit.RSquared, 1, 1e-10)
}

func TestCalculatePolynomialWeighting(t *testing.T) {
	// 三次重复观测中权重最大的样本把常数拟合拉向 Y = 3
	samples := []Sample{
		{X: 0, Y: 0, Weight: 1},
		{X: 1, Y: 1, Weight: 1},
		{X: 1, Y: 3, Weight: 9},
	}

	fit, err := CalculatePolynomial(samples, 0)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "常数项", fit.Coefficients[0], 28.0/11, 1e-12)
	requireClose(t, "R²", fit.RSquared, 0, 1e-12)
	requireClose(t, "加权残差", fit.WeightedResidual, fit.WeightedTotal, 1e-12)
}

func TestCalculatePolynomialZeroWeight(t *testing.T) {
	samples := []Sample{
		{X: 0, Y: 0, Weight: 1},
		{X: 1, Y: 1, Weight: 1},
		{X: 2, Y: 100, Weight: 0},
	}

	fit, err := CalculatePolynomial(samples, 1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "常数项", fit.Coefficients[0], 0, 1e-12)
	requireClose(t, "一次项", fit.Coefficients[1], 1, 1e-12)

	if fit.Samples != 2 {
		t.Fatalf("Samples = %d, want 2", fit.Samples)
	}
}

func TestCalculatePolynomialRSquared(t *testing.T) {
	// 常数模型只能取均值，残差平方和与总平方和相等，R² = 0
	samples := []Sample{
		{X: 0, Y: 0, Weight: 1},
		{X: 1, Y: 2, Weight: 1},
		{X: 2, Y: 1, Weight: 1},
	}

	fit, err := CalculatePolynomial(samples, 0)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "常数项", fit.Coefficients[0], 1, 1e-12)
	requireClose(t, "R²", fit.RSquared, 0, 1e-12)
	requireClose(t, "加权残差", fit.WeightedResidual, 2, 1e-12)
}

func TestCalculatePolynomialErrors(t *testing.T) {
	samples := []Sample{
		{X: 0, Y: 0, Weight: 1},
		{X: 1, Y: 1, Weight: 1},
	}

	tests := []struct {
		name    string
		samples []Sample
		degree  int
		want    error
	}{
		{"空输入", nil, 1, ErrEmptyInput},
		{"负阶数", samples, -1, ErrInvalidDegree},
		{"阶数过高", samples, 13, ErrInvalidDegree},
		{"样本不足", samples, 2, ErrTooFewSamples},
		{"负权重", []Sample{{X: 0, Y: 0, Weight: -1}, {X: 1, Y: 1, Weight: 1}}, 0, ErrNegativeWeight},
		{"NaN 自变量", []Sample{{X: math.NaN(), Y: 0, Weight: 1}}, 0, ErrNonFinite},
		{"Inf 因变量", []Sample{{X: 0, Y: math.Inf(1), Weight: 1}}, 0, ErrNonFinite},
		{"NaN 权重", []Sample{{X: 0, Y: 0, Weight: math.NaN()}}, 0, ErrNonFinite},
		{"列线性相关", []Sample{{X: 1, Y: 1, Weight: 1}, {X: 1, Y: 2, Weight: 1}}, 1, ErrSingularSystem},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculatePolynomial(tt.samples, tt.degree)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCalculatePolynomialDoesNotAliasInput(t *testing.T) {
	samples := []Sample{
		{X: 0, Y: 0, Weight: 1},
		{X: 1, Y: 1, Weight: 1},
		{X: 2, Y: 4, Weight: 1},
	}

	fit, err := CalculatePolynomial(samples, 2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	fit.Coefficients[0] = 100
	if samples[0].Y != 0 {
		t.Fatalf("结果与输入共享底层数组")
	}
}
