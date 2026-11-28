package distribution

import (
	"errors"
	"math"
	"testing"
)

func TestPoissonPMF(t *testing.T) {
	poisson, err := NewPoisson(3)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "均值", poisson.GetMean(), 3, 1e-12)
	requireClose(t, "方差", poisson.GetVariance(), 3, 1e-12)

	tests := []struct {
		name string
		k    int
		want float64
	}{
		{"负值", -1, 0},
		{"零", 0, math.Exp(-3)},
		{"一", 1, 3 * math.Exp(-3)},
		{"三", 3, 4.5 * math.Exp(-3)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireClose(t, "概率质量", poisson.CalculatePMF(tt.k), tt.want, 1e-15)
		})
	}

	total := 0.0
	for k := 0; k <= 100; k++ {
		total += poisson.CalculatePMF(k)
	}

	requireClose(t, "总概率", total, 1, 1e-12)
}

func TestPoissonCDF(t *testing.T) {
	poisson, err := NewPoisson(3)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	tests := []struct {
		name string
		k    int
		want float64
	}{
		{"负值", -1, 0},
		{"零", 0, math.Exp(-3)},
		{"三", 3, 13 * math.Exp(-3)},
		{"远离均值", 100, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireClose(t, "分布函数", poisson.CalculateCDF(tt.k), tt.want, 1e-15)
		})
	}
}

func TestPoissonQuantile(t *testing.T) {
	poisson, err := NewPoisson(3)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	tests := []struct {
		name        string
		probability float64
		want        int
	}{
		{"中位数", 0.5, 3},
		{"九成分位", 0.9, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quantile, err := poisson.CalculateQuantile(tt.probability)
			if err != nil {
				t.Fatalf("err = %v", err)
			}

			if quantile != tt.want {
				t.Fatalf("分位数 = %d, want %d", quantile, tt.want)
			}

			if poisson.CalculateCDF(quantile) < tt.probability {
				t.Fatalf("CDF(%d) 未达到 %g", quantile, tt.probability)
			}

			if quantile > 0 && poisson.CalculateCDF(quantile-1) >= tt.probability {
				t.Fatalf("分位数不是最小的 k")
			}
		})
	}
}

func TestPoissonErrors(t *testing.T) {
	tests := []struct {
		name string
		mean float64
	}{
		{"零均值", 0},
		{"负均值", -1},
		{"NaN 均值", math.NaN()},
		{"Inf 均值", math.Inf(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPoisson(tt.mean)
			if !errors.Is(err, ErrInvalidMean) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidMean)
			}
		})
	}

	poisson, err := NewPoisson(1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	for _, probability := range []float64{0, 1, -1, math.NaN()} {
		if _, err := poisson.CalculateQuantile(probability); !errors.Is(err, ErrInvalidProbability) {
			t.Fatalf("probability %g: err = %v, want %v", probability, err, ErrInvalidProbability)
		}
	}
}
