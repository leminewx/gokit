package distribution

import (
	"errors"
	"math"
	"testing"
)

func TestUniform(t *testing.T) {
	uniform, err := NewUniform(0, 4)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "均值", uniform.GetMean(), 2, 1e-12)
	requireClose(t, "方差", uniform.GetVariance(), 4.0/3, 1e-12)

	tests := []struct {
		name string
		x    float64
		pdf  float64
		cdf  float64
	}{
		{"区间左侧", -1, 0, 0},
		{"区间内部", 2, 0.25, 0.5},
		{"下界", 0, 0.25, 0},
		{"上界", 4, 0.25, 1},
		{"区间右侧", 5, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			density, err := uniform.CalculatePDF(tt.x)
			if err != nil {
				t.Fatalf("err = %v", err)
			}

			requireClose(t, "密度", density, tt.pdf, 1e-12)

			distributionFunction, err := uniform.CalculateCDF(tt.x)
			if err != nil {
				t.Fatalf("err = %v", err)
			}

			requireClose(t, "分布函数", distributionFunction, tt.cdf, 1e-12)
		})
	}
}

func TestUniformQuantile(t *testing.T) {
	uniform, err := NewUniform(-2, 6)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	for _, probability := range []float64{0.1, 0.25, 0.5, 0.9} {
		quantile, err := uniform.CalculateQuantile(probability)
		if err != nil {
			t.Fatalf("err = %v", err)
		}

		restored, err := uniform.CalculateCDF(quantile)
		if err != nil {
			t.Fatalf("err = %v", err)
		}

		requireClose(t, "往返概率", restored, probability, 1e-12)
	}
}

func TestUniformErrors(t *testing.T) {
	tests := []struct {
		name  string
		lower float64
		upper float64
	}{
		{"区间退化", 1, 1},
		{"区间反向", 2, 1},
		{"下界非有限", math.Inf(-1), 1},
		{"上界 NaN", 1, math.NaN()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewUniform(tt.lower, tt.upper)
			if !errors.Is(err, ErrInvalidRange) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidRange)
			}
		})
	}

	uniform, err := NewUniform(0, 1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if _, err := uniform.CalculatePDF(math.NaN()); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}

	if _, err := uniform.CalculateQuantile(0); !errors.Is(err, ErrInvalidProbability) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidProbability)
	}
}
