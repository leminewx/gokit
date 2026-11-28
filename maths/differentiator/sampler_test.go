package differentiator

import (
	"errors"
	"math"
	"testing"
)

func TestNewSampler(t *testing.T) {
	tests := []struct {
		name string
		xs   []float64
		ys   []float64
		want error
	}{
		{"长度不一致", []float64{0, 1}, []float64{0, 1, 4}, ErrLengthMismatch},
		{"样本不足", []float64{0, 1}, []float64{0, 1}, ErrTooFewSamples},
		{"横坐标重复", []float64{0, 1, 1, 2}, []float64{0, 1, 1, 4}, ErrNotStrictlyIncreasing},
		{"横坐标递减", []float64{0, 2, 1}, []float64{0, 4, 1}, ErrNotStrictlyIncreasing},
		{"空输入", nil, nil, ErrTooFewSamples},
		{"含 NaN", []float64{0, 1, math.NaN()}, []float64{0, 1, 2}, ErrNonFinite},
		{"含 Inf", []float64{0, 1, 2}, []float64{0, math.Inf(1), 2}, ErrNonFinite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSampler(tt.xs, tt.ys)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestSamplerCalculateDerivative(t *testing.T) {
	xs := []float64{0, 1, 2, 4}
	ys := []float64{0, 1, 4, 16}

	sampler, err := NewSampler(xs, ys)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	tests := []struct {
		name  string
		x     float64
		order int
		want  float64
	}{
		{"左端点", 0, 1, 0},
		{"内部点", 0.5, 1, 1},
		{"样本点", 2, 1, 4},
		{"右端点", 4, 1, 8},
		{"二阶导左端", 0, 2, 2},
		{"二阶导内部", 1.5, 2, 2},
		{"二阶导右端", 4, 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			derivative, err := sampler.CalculateDerivativeOrder(tt.x, tt.order)
			if err != nil {
				t.Fatalf("err = %v", err)
			}

			// 二次多项式被三点插值精确复现，误差只来自舍入
			requireClose(t, "导数值", derivative.Value, tt.want, 1e-12)
			if derivative.Window != 3 {
				t.Fatalf("Window = %d, want 3", derivative.Window)
			}
		})
	}
}

func TestSamplerCalculateDerivativeAveraging(t *testing.T) {
	// 三次函数上两侧二次拟合的结果不同，取平均并以差值的一半作为误差指示
	xs := []float64{0, 1, 2, 3, 4}
	ys := make([]float64, len(xs))
	for index := range xs {
		ys[index] = xs[index] * xs[index] * xs[index]
	}

	sampler, err := NewSampler(xs, ys)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	derivative, err := sampler.CalculateDerivative(1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	// 左侧模板给出 4，右侧模板给出 1，真值 3 落在两者之间
	requireClose(t, "导数值", derivative.Value, 2.5, 1e-12)
	requireClose(t, "误差指示", derivative.ErrorEstimate, 1.5, 1e-12)
}

func TestSamplerCalculateDerivativeErrors(t *testing.T) {
	sampler, err := NewSampler([]float64{0, 1, 2}, []float64{0, 1, 4})
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if _, err := sampler.CalculateDerivativeOrder(1, 3); !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidOrder)
	}

	if _, err := sampler.CalculateDerivative(-0.1); !errors.Is(err, ErrOutOfRange) {
		t.Fatalf("err = %v, want %v", err, ErrOutOfRange)
	}

	if _, err := sampler.CalculateDerivative(2.1); !errors.Is(err, ErrOutOfRange) {
		t.Fatalf("err = %v, want %v", err, ErrOutOfRange)
	}

	if _, err := sampler.CalculateDerivative(math.NaN()); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}

func TestSamplerImmutability(t *testing.T) {
	xs := []float64{0, 1, 2}
	ys := []float64{0, 1, 4}

	sampler, err := NewSampler(xs, ys)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	xs[1] = 100
	ys[2] = 1000

	derivative, err := sampler.CalculateDerivative(1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "导数值", derivative.Value, 2, 1e-12)
}
