package optimizer

import (
	"errors"
	"math"
	"testing"
)

// rosenbrock 二维 Rosenbrock 函数，全局极小点在 (1, 1)
func rosenbrock(point []float64) float64 {
	difference := 1 - point[0]
	second := point[1] - point[0]*point[0]
	return difference*difference + 100*second*second
}

func TestMinimizeNelderMeadQuadratic(t *testing.T) {
	solution, err := MinimizeNelderMead(func(point []float64) float64 {
		difference := point[0] - 2
		return difference * difference
	}, []float64{0})
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "极小点", solution.X[0], 2, 1e-5)
	if solution.Value > 1e-12 {
		t.Fatalf("极小值 = %g, want ~0", solution.Value)
	}

	if len(solution.X) != 1 {
		t.Fatalf("len = %d, want 1", len(solution.X))
	}
}

func TestMinimizeNelderMeadRosenbrock(t *testing.T) {
	solution, err := MinimizeNelderMead(rosenbrock, []float64{-1.2, 1}, WithTolerance(1e-10), WithMaxIterations(10000))
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "x", solution.X[0], 1, 1e-4)
	requireClose(t, "y", solution.X[1], 1, 1e-4)

	if solution.Value > 1e-8 {
		t.Fatalf("极小值 = %g, want <= 1e-8", solution.Value)
	}
}

func TestMinimizeNelderMeadNotConverged(t *testing.T) {
	_, err := MinimizeNelderMead(rosenbrock, []float64{-1.2, 1}, WithMaxIterations(1))
	if !errors.Is(err, ErrNotConverged) {
		t.Fatalf("err = %v, want %v", err, ErrNotConverged)
	}
}

func TestMinimizeNelderMeadErrors(t *testing.T) {
	if _, err := MinimizeNelderMead(rosenbrock, nil); !errors.Is(err, ErrEmptyInitial) {
		t.Fatalf("err = %v, want %v", err, ErrEmptyInitial)
	}

	if _, err := MinimizeNelderMead(rosenbrock, []float64{0, math.NaN()}); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}

	if _, err := MinimizeNelderMead(func([]float64) float64 { return math.Inf(1) }, []float64{0}); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}

func TestMinimizeNelderMeadDoesNotModifyInitial(t *testing.T) {
	initial := []float64{3, -2}

	if _, err := MinimizeNelderMead(rosenbrock, initial); err != nil {
		t.Fatalf("err = %v", err)
	}

	if initial[0] != 3 || initial[1] != -2 {
		t.Fatalf("初值被改写：%v", initial)
	}
}
