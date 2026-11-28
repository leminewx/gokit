package root

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateSecant(t *testing.T) {
	tests := []struct {
		name      string
		function  Function
		a, b      float64
		want      float64
		tolerance float64
	}{
		{"cos x - x", func(x float64) float64 { return math.Cos(x) - x }, 0, 1, 0.7390851332151607, 1e-12},
		{"x²-2", func(x float64) float64 { return x*x - 2 }, 1, 2, math.Sqrt2, 1e-12},
		{"x³-2x-5", func(x float64) float64 { return x*x*x - 2*x - 5 }, 2, 3, 2.0945514815423265, 1e-11},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			solution, err := CalculateSecant(tt.function, tt.a, tt.b)
			if err != nil {
				t.Fatalf("CalculateSecant: %v", err)
			}

			requireClose(t, "Root", solution.Root, tt.want, tt.tolerance)
			requireClose(t, "Value", solution.Value, 0, 1e-9)
		})
	}
}

func TestCalculateSecantRejectsInvalidInterval(t *testing.T) {
	tests := []struct {
		name string
		a, b float64
	}{
		{"两端点相同", 1, 1},
		{"a 非有限", math.Inf(-1), 1},
		{"b 为 NaN", 0, math.NaN()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateSecant(func(x float64) float64 { return x }, tt.a, tt.b)
			if !errors.Is(err, ErrInvalidInterval) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidInterval)
			}
		})
	}
}

func TestCalculateSecantEqualValues(t *testing.T) {
	// 关于 1.5 对称的两个点函数值相同，弦截斜率退化
	_, err := CalculateSecant(func(x float64) float64 { return (x - 1.5) * (x - 1.5) }, 1, 2)
	if !errors.Is(err, ErrNoSignChange) {
		t.Fatalf("err = %v, want %v", err, ErrNoSignChange)
	}
}

func TestCalculateSecantNonFinite(t *testing.T) {
	_, err := CalculateSecant(func(x float64) float64 { return math.Inf(1) }, 0, 1)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}

func TestCalculateSecantNotConverged(t *testing.T) {
	_, err := CalculateSecant(func(x float64) float64 { return x*x - 2 }, 1, 2,
		WithTolerance(1e-15), WithMaxIterations(1))
	if !errors.Is(err, ErrNotConverged) {
		t.Fatalf("err = %v, want %v", err, ErrNotConverged)
	}
}
