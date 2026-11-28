package quadrature

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateGaussLegendreExactness(t *testing.T) {
	// n 点 Gauss-Legendre 对 2n-1 次多项式精确
	tests := []struct {
		name     string
		exponent int
		order    int
	}{
		{"2 点/3 次", 3, 2},
		{"3 点/5 次", 5, 3},
		{"4 点/7 次", 7, 4},
		{"5 点/9 次", 9, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			function := func(x float64) float64 { return math.Pow(x, float64(tt.exponent)) }
			want := 1.0 / float64(tt.exponent+1)

			result, err := CalculateGaussLegendre(function, 0, 1, WithOrder(tt.order), WithPanels(1))
			if err != nil {
				t.Fatalf("CalculateGaussLegendre: %v", err)
			}

			requireClose(t, "Value", result.Value, want, 1e-15)
		})
	}
}

func TestCalculateGaussLegendreTranscendental(t *testing.T) {
	tests := []struct {
		name      string
		function  Function
		a, b      float64
		want      float64
		tolerance float64
	}{
		{"exp", math.Exp, 0, 1, math.E - 1, 1e-14},
		{"sin", math.Sin, 0, math.Pi, 2, 1e-12},
		{"ln", math.Log, 1, 2, 2*math.Log(2) - 1, 1e-12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CalculateGaussLegendre(tt.function, tt.a, tt.b, WithOrder(5), WithPanels(4))
			if err != nil {
				t.Fatalf("CalculateGaussLegendre: %v", err)
			}

			requireClose(t, "Value", result.Value, tt.want, tt.tolerance)
		})
	}
}

func TestCalculateGaussLegendreErrorEstimation(t *testing.T) {
	function := func(x float64) float64 { return math.Cos(10 * x) }

	plain, err := CalculateGaussLegendre(function, 0, 1, WithOrder(4), WithPanels(2))
	if err != nil {
		t.Fatalf("CalculateGaussLegendre: %v", err)
	}

	if plain.ErrorEstimate != 0 {
		t.Fatalf("ErrorEstimate = %g, want 0 (默认不估计)", plain.ErrorEstimate)
	}

	estimated, err := CalculateGaussLegendre(function, 0, 1,
		WithOrder(4), WithPanels(2), WithErrorEstimation())
	if err != nil {
		t.Fatalf("CalculateGaussLegendre: %v", err)
	}

	if estimated.ErrorEstimate <= 0 {
		t.Fatalf("ErrorEstimate = %g, want > 0", estimated.ErrorEstimate)
	}

	requireClose(t, "Value", estimated.Value, plain.Value, 1e-15)
}

func TestCalculateGaussLegendreNonFinite(t *testing.T) {
	_, err := CalculateGaussLegendre(func(x float64) float64 { return math.NaN() }, 0, 1)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}
