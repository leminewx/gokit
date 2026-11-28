package distribution

import (
	"errors"
	"math"
	"testing"
)

func TestNewNormal(t *testing.T) {
	tests := []struct {
		name   string
		mean   float64
		stdDev float64
		want   error
	}{
		{"零标准差", 0, 0, ErrInvalidStdDev},
		{"负标准差", 0, -1, ErrInvalidStdDev},
		{"NaN 标准差", 0, math.NaN(), ErrInvalidStdDev},
		{"Inf 标准差", 0, math.Inf(1), ErrInvalidStdDev},
		{"NaN 均值", math.NaN(), 1, ErrNonFinite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNormal(tt.mean, tt.stdDev)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestNormalStandard(t *testing.T) {
	normal, err := NewNormal(0, 1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if normal.GetMean() != 0 {
		t.Fatalf("GetMean = %g, want 0", normal.GetMean())
	}

	requireClose(t, "方差", normal.GetVariance(), 1, 1e-12)

	density, err := normal.CalculatePDF(0)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "密度", density, 1/_SQRT_TWO_PI, 1e-12)

	distributionFunction, err := normal.CalculateCDF(0)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "分布函数", distributionFunction, 0.5, 1e-15)

	distributionFunction, err = normal.CalculateCDF(1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "分布函数", distributionFunction, 0.8413447460685429, 1e-12)
}

func TestNormalScaled(t *testing.T) {
	normal, err := NewNormal(2, 3)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "方差", normal.GetVariance(), 9, 1e-12)

	density, err := normal.CalculatePDF(2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "密度", density, 1/(3*_SQRT_TWO_PI), 1e-12)

	distributionFunction, err := normal.CalculateCDF(2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "分布函数", distributionFunction, 0.5, 1e-15)

	quantile, err := normal.CalculateQuantile(0.5)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "分位数", quantile, 2, 1e-12)
}

func TestNormalQuantile(t *testing.T) {
	normal, err := NewNormal(0, 1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	quantile, err := normal.CalculateQuantile(0.975)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "分位数", quantile, 1.959963984540054, 1e-9)
}

func TestNormalRoundTrip(t *testing.T) {
	normal, err := NewNormal(5, 2.5)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	for _, probability := range []float64{0.001, 0.01, 0.25, 0.5, 0.975, 0.999} {
		quantile, err := normal.CalculateQuantile(probability)
		if err != nil {
			t.Fatalf("err = %v", err)
		}

		restored, err := normal.CalculateCDF(quantile)
		if err != nil {
			t.Fatalf("err = %v", err)
		}

		requireClose(t, "往返概率", restored, probability, 1e-10)
	}
}

func TestNormalErrors(t *testing.T) {
	normal, err := NewNormal(0, 1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if _, err := normal.CalculatePDF(math.NaN()); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}

	if _, err := normal.CalculateCDF(math.Inf(1)); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}

	for _, probability := range []float64{0, 1, -0.1, 1.1, math.NaN()} {
		if _, err := normal.CalculateQuantile(probability); !errors.Is(err, ErrInvalidProbability) {
			t.Fatalf("probability %g: err = %v, want %v", probability, err, ErrInvalidProbability)
		}
	}
}
