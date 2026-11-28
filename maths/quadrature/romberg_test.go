package quadrature

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateRomberg(t *testing.T) {
	tests := []struct {
		name      string
		function  Function
		a, b      float64
		want      float64
		tolerance float64
	}{
		{"x²", func(x float64) float64 { return x * x }, 0, 1, 1.0 / 3, 1e-12},
		{"sin", math.Sin, 0, math.Pi, 2, 1e-11},
		{"1/(1+x)", func(x float64) float64 { return 1 / (1 + x) }, 0, 1, math.Ln2, 1e-11},
		{"4/(1+x²)", func(x float64) float64 { return 4 / (1 + x*x) }, 0, 1, math.Pi, 1e-11},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CalculateRomberg(tt.function, tt.a, tt.b)
			if err != nil {
				t.Fatalf("CalculateRomberg: %v", err)
			}

			requireClose(t, "Value", result.Value, tt.want, tt.tolerance)

			if result.ErrorEstimate < 0 {
				t.Fatalf("ErrorEstimate = %g, want >= 0", result.ErrorEstimate)
			}

			if result.Iterations < 2 {
				t.Fatalf("Iterations = %d, want >= 2", result.Iterations)
			}
		})
	}
}

func TestCalculateRombergRefinesWithTolerance(t *testing.T) {
	function := func(x float64) float64 { return math.Cos(10 * x) }

	loose, err := CalculateRomberg(function, 0, 1, WithTolerance(1e-3))
	if err != nil {
		t.Fatalf("CalculateRomberg with 1e-3: %v", err)
	}

	strict, err := CalculateRomberg(function, 0, 1, WithTolerance(1e-12))
	if err != nil {
		t.Fatalf("CalculateRomberg with 1e-12: %v", err)
	}

	// 更严格的容差必须消耗更多行，且估计值更小
	if strict.Iterations <= loose.Iterations {
		t.Fatalf("Iterations strict = %d, want > %d (loose)", strict.Iterations, loose.Iterations)
	}

	if strict.ErrorEstimate > loose.ErrorEstimate {
		t.Fatalf("ErrorEstimate strict = %g, want <= %g (loose)", strict.ErrorEstimate, loose.ErrorEstimate)
	}
}

func TestCalculateRombergNotConverged(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
	}{
		{"仅一行", []Option{WithMaxIterations(1)}},
		{"两行且容差过严", []Option{WithMaxIterations(2), WithTolerance(1e-16)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateRomberg(func(x float64) float64 { return math.Exp(x) }, 0, 1, tt.options...)
			if !errors.Is(err, ErrNotConverged) {
				t.Fatalf("err = %v, want %v", err, ErrNotConverged)
			}
		})
	}
}

func TestCalculateRombergNonFinite(t *testing.T) {
	_, err := CalculateRomberg(func(x float64) float64 { return math.NaN() }, 0, 1)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}
