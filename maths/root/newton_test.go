package root

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateNewton(t *testing.T) {
	tests := []struct {
		name      string
		function  Function
		initial   float64
		want      float64
		tolerance float64
	}{
		{"x²-2", func(x float64) float64 { return x*x - 2 }, 1.5, math.Sqrt2, 1e-12},
		{"cos x - x", func(x float64) float64 { return math.Cos(x) - x }, 0.5, 0.7390851332151607, 1e-12},
		{"x³-2x-5", func(x float64) float64 { return x*x*x - 2*x - 5 }, 2, 2.0945514815423265, 1e-11},
		{"sqrt(1+x²) 的极小点导数根", func(x float64) float64 { return x / math.Sqrt(1+x*x) }, 0.1, 0, 1e-12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			solution, err := CalculateNewton(tt.function, tt.initial)
			if err != nil {
				t.Fatalf("CalculateNewton: %v", err)
			}

			requireClose(t, "Root", solution.Root, tt.want, tt.tolerance)
			requireClose(t, "Value", solution.Value, 0, 1e-9)

			if solution.Iterations > 20 {
				t.Fatalf("Iterations = %d, want <= 20", solution.Iterations)
			}
		})
	}
}

func TestCalculateNewtonWithExplicitDerivative(t *testing.T) {
	solution, err := CalculateNewton(
		func(x float64) float64 { return x*x - 2 },
		1.5,
		WithDerivative(func(x float64) float64 { return 2 * x }),
	)
	if err != nil {
		t.Fatalf("CalculateNewton: %v", err)
	}

	requireClose(t, "Root", solution.Root, math.Sqrt2, 1e-14)
}

func TestCalculateNewtonQuadraticConvergence(t *testing.T) {
	// 二次收敛：3 次迭代即可把误差压到 1e-5 以下
	solution, err := CalculateNewton(
		func(x float64) float64 { return x*x - 2 },
		1.5,
		WithDerivative(func(x float64) float64 { return 2 * x }),
		WithMaxIterations(3),
		WithTolerance(1e-5),
	)
	if err != nil {
		t.Fatalf("CalculateNewton: %v", err)
	}

	if math.Abs(solution.Root-math.Sqrt2) > 1e-5 {
		t.Fatalf("Root = %.15g, want within 1e-5 of √2 after 3 iterations", solution.Root)
	}
}

func TestCalculateNewtonZeroDerivative(t *testing.T) {
	// 常数函数的差分导数恒为 0
	_, err := CalculateNewton(func(x float64) float64 { return 1 }, 0.5)
	if !errors.Is(err, ErrZeroDerivative) {
		t.Fatalf("err = %v, want %v", err, ErrZeroDerivative)
	}
}

func TestCalculateNewtonNonFinite(t *testing.T) {
	_, err := CalculateNewton(func(x float64) float64 { return math.NaN() }, 1)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}

func TestCalculateNewtonNonFiniteInitial(t *testing.T) {
	_, err := CalculateNewton(func(x float64) float64 { return x }, math.Inf(1))
	if !errors.Is(err, ErrInvalidInitial) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidInitial)
	}
}

func TestCalculateNewtonNotConverged(t *testing.T) {
	_, err := CalculateNewton(func(x float64) float64 { return x*x - 2 }, 1.5,
		WithTolerance(1e-15), WithMaxIterations(1))
	if !errors.Is(err, ErrNotConverged) {
		t.Fatalf("err = %v, want %v", err, ErrNotConverged)
	}
}
