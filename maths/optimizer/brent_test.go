package optimizer

import (
	"errors"
	"math"
	"testing"
)

func TestMinimizeBrentQuadratic(t *testing.T) {
	solution, err := MinimizeBrent(func(x float64) float64 { return (x - 2) * (x - 2) }, -5, 5)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "极小点", solution.X, 2, 1e-6)
	requireClose(t, "极小值", solution.Value, 0, 1e-12)

	if solution.Iterations <= 0 || solution.Evaluations <= 0 {
		t.Fatalf("Iterations/Evaluations = %d/%d, want > 0", solution.Iterations, solution.Evaluations)
	}
}

func TestMinimizeBrentCosine(t *testing.T) {
	// cos(x) 在 [0, 7] 上于 π 处取得最小值 -1
	solution, err := MinimizeBrent(math.Cos, 0, 7, WithTolerance(1e-10))
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "极小点", solution.X, math.Pi, 1e-6)
	requireClose(t, "极小值", solution.Value, -1, 1e-12)
}

func TestMinimizeBrentQuartic(t *testing.T) {
	solution, err := MinimizeBrent(func(x float64) float64 {
		difference := x + 1
		return difference * difference * difference * difference
	}, -4, 2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "极小点", solution.X, -1, 1e-5)
	if solution.Value > 1e-12 {
		t.Fatalf("极小值 = %g, want ~0", solution.Value)
	}
}

func TestMinimizeBrentNotConverged(t *testing.T) {
	_, err := MinimizeBrent(func(x float64) float64 { return (x - 2) * (x - 2) }, -5, 5, WithMaxIterations(1))
	if !errors.Is(err, ErrNotConverged) {
		t.Fatalf("err = %v, want %v", err, ErrNotConverged)
	}
}

func TestMinimizeBrentErrors(t *testing.T) {
	function := func(x float64) float64 { return x * x }

	tests := []struct {
		name  string
		lower float64
		upper float64
		want  error
	}{
		{"区间反向", 1, -1, ErrInvalidRange},
		{"区间退化", 1, 1, ErrInvalidRange},
		{"下界非有限", math.Inf(-1), 1, ErrInvalidRange},
		{"上界 NaN", -1, math.NaN(), ErrInvalidRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MinimizeBrent(function, tt.lower, tt.upper)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}

	_, err := MinimizeBrent(func(float64) float64 { return math.NaN() }, 0, 1)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}
