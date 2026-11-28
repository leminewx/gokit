package distribution

import (
	"errors"
	"math"
	"testing"
)

func TestExponential(t *testing.T) {
	exponential, err := NewExponential(2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "均值", exponential.GetMean(), 0.5, 1e-12)
	requireClose(t, "方差", exponential.GetVariance(), 0.25, 1e-12)

	tests := []struct {
		name string
		x    float64
		pdf  float64
		cdf  float64
	}{
		{"负半轴", -1, 0, 0},
		{"零点", 0, 2, 0},
		{"正半轴", 1, 2 * math.Exp(-2), 1 - math.Exp(-2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			density, err := exponential.CalculatePDF(tt.x)
			if err != nil {
				t.Fatalf("err = %v", err)
			}

			requireClose(t, "密度", density, tt.pdf, 1e-12)

			distributionFunction, err := exponential.CalculateCDF(tt.x)
			if err != nil {
				t.Fatalf("err = %v", err)
			}

			requireClose(t, "分布函数", distributionFunction, tt.cdf, 1e-12)
		})
	}
}

func TestExponentialQuantile(t *testing.T) {
	exponential, err := NewExponential(2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	quantile, err := exponential.CalculateQuantile(0.5)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "中位数", quantile, math.Log(2)/2, 1e-12)

	restored, err := exponential.CalculateCDF(quantile)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "往返概率", restored, 0.5, 1e-12)
}

func TestExponentialErrors(t *testing.T) {
	tests := []struct {
		name string
		rate float64
	}{
		{"零速率", 0},
		{"负速率", -1},
		{"NaN 速率", math.NaN()},
		{"Inf 速率", math.Inf(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewExponential(tt.rate)
			if !errors.Is(err, ErrInvalidRate) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidRate)
			}
		})
	}

	exponential, err := NewExponential(1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if _, err := exponential.CalculateCDF(math.NaN()); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}

	if _, err := exponential.CalculateQuantile(1); !errors.Is(err, ErrInvalidProbability) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidProbability)
	}
}
