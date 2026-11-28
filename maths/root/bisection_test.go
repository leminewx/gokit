package root

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateBisection(t *testing.T) {
	tests := []struct {
		name      string
		function  Function
		a, b      float64
		want      float64
		tolerance float64
	}{
		{"x²-2", func(x float64) float64 { return x*x - 2 }, 1, 2, math.Sqrt2, 2e-9},
		{"cos x - x", func(x float64) float64 { return math.Cos(x) - x }, 0, 1, 0.7390851332151607, 2e-9},
		{"x³-2x-5", func(x float64) float64 { return x*x*x - 2*x - 5 }, 2, 3, 2.0945514815423265, 2e-9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			solution, err := CalculateBisection(tt.function, tt.a, tt.b)
			if err != nil {
				t.Fatalf("CalculateBisection: %v", err)
			}

			requireClose(t, "Root", solution.Root, tt.want, tt.tolerance)
			requireClose(t, "Value", solution.Value, 0, 1e-8)

			if solution.Iterations <= 0 {
				t.Fatalf("Iterations = %d, want > 0", solution.Iterations)
			}
		})
	}
}

func TestCalculateBisectionAtEndpoint(t *testing.T) {
	// 左端点恰为根时直接返回，不进行迭代
	solution, err := CalculateBisection(func(x float64) float64 { return x - 1 }, 1, 2)
	if err != nil {
		t.Fatalf("CalculateBisection: %v", err)
	}

	requireClose(t, "Root", solution.Root, 1, 0)
	requireClose(t, "Value", solution.Value, 0, 0)

	if solution.Iterations != 0 {
		t.Fatalf("Iterations = %d, want 0", solution.Iterations)
	}
}

func TestCalculateBisectionRejectsInvalidInterval(t *testing.T) {
	tests := []struct {
		name string
		a, b float64
		want error
	}{
		{"端点相同", 1, 1, ErrInvalidInterval},
		{"区间反向", 2, 1, ErrInvalidInterval},
		{"左端点非有限", math.Inf(1), 1, ErrInvalidInterval},
		{"右端点非有限", 0, math.NaN(), ErrInvalidInterval},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateBisection(func(x float64) float64 { return x }, tt.a, tt.b)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCalculateBisectionWithoutSignChange(t *testing.T) {
	_, err := CalculateBisection(func(x float64) float64 { return x*x + 1 }, -1, 1)
	if !errors.Is(err, ErrNoSignChange) {
		t.Fatalf("err = %v, want %v", err, ErrNoSignChange)
	}
}

func TestCalculateBisectionNonFinite(t *testing.T) {
	_, err := CalculateBisection(func(x float64) float64 { return math.NaN() }, 0, 1)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}

func TestCalculateBisectionNotConverged(t *testing.T) {
	_, err := CalculateBisection(func(x float64) float64 { return x*x - 2 }, 1, 2,
		WithTolerance(1e-15), WithMaxIterations(2))
	if !errors.Is(err, ErrNotConverged) {
		t.Fatalf("err = %v, want %v", err, ErrNotConverged)
	}
}

func TestCalculateBisectionWithResidual(t *testing.T) {
	// 残差容差可让迭代在 |f(x)| 足够小时提前结束
	solution, err := CalculateBisection(func(x float64) float64 { return x*x - 2 }, 1, 2,
		WithTolerance(1e-15), WithResidual(1e-6), WithMaxIterations(100))
	if err != nil {
		t.Fatalf("CalculateBisection: %v", err)
	}

	if math.Abs(solution.Value) > 1e-6 {
		t.Fatalf("Value = %g, want |f| <= 1e-6", solution.Value)
	}
}
