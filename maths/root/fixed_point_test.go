package root

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateFixedPoint(t *testing.T) {
	solution, err := CalculateFixedPoint(math.Cos, 0.5)
	if err != nil {
		t.Fatalf("CalculateFixedPoint: %v", err)
	}

	requireClose(t, "Root", solution.Root, 0.7390851332151607, 1e-8)

	if math.Abs(solution.Value) > 1e-8 {
		t.Fatalf("Value = %g, want |g(x)-x| <= 1e-8", solution.Value)
	}
}

func TestCalculateFixedPointWithAitken(t *testing.T) {
	plain, err := CalculateFixedPoint(math.Cos, 0.5)
	if err != nil {
		t.Fatalf("CalculateFixedPoint: %v", err)
	}

	accelerated, err := CalculateFixedPoint(math.Cos, 0.5, WithAitken())
	if err != nil {
		t.Fatalf("CalculateFixedPoint with Aitken: %v", err)
	}

	requireClose(t, "Root", accelerated.Root, 0.7390851332151607, 1e-8)

	// Aitken 加速应显著减少迭代次数
	if accelerated.Iterations >= plain.Iterations {
		t.Fatalf("Iterations with Aitken = %d, want < %d (without)", accelerated.Iterations, plain.Iterations)
	}
}

func TestCalculateFixedPointDiverges(t *testing.T) {
	_, err := CalculateFixedPoint(func(x float64) float64 { return x + 1 }, 0)
	if !errors.Is(err, ErrNotConverged) {
		t.Fatalf("err = %v, want %v", err, ErrNotConverged)
	}
}

func TestCalculateFixedPointNonFinite(t *testing.T) {
	_, err := CalculateFixedPoint(func(x float64) float64 { return math.Inf(1) }, 0)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}

func TestCalculateFixedPointNonFiniteInitial(t *testing.T) {
	_, err := CalculateFixedPoint(math.Cos, math.NaN())
	if !errors.Is(err, ErrInvalidInitial) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidInitial)
	}
}
