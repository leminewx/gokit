package ode

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateFehlbergExponential(t *testing.T) {
	trajectory, err := CalculateFehlberg(exponential, []float64{1}, 0, 1, WithTolerance(1e-10))
	if err != nil {
		t.Fatalf("CalculateFehlberg: %v", err)
	}

	requireClose(t, "y(1)", trajectory.Ys[len(trajectory.Ys)-1][0], math.E, 1e-6)

	if last := trajectory.Xs[len(trajectory.Xs)-1]; last != 1 {
		t.Fatalf("终点 = %g, want 1", last)
	}

	if len(trajectory.Xs) != len(trajectory.Ys) {
		t.Fatalf("Xs 与 Ys 长度不一致: %d != %d", len(trajectory.Xs), len(trajectory.Ys))
	}

	if trajectory.Steps != len(trajectory.Xs)-1 {
		t.Fatalf("Steps = %d, want %d", trajectory.Steps, len(trajectory.Xs)-1)
	}

	for index := 1; index < len(trajectory.Xs); index++ {
		if trajectory.Xs[index] <= trajectory.Xs[index-1] {
			t.Fatalf("Xs 非严格递增: Xs[%d] = %g, Xs[%d] = %g",
				index, trajectory.Xs[index], index-1, trajectory.Xs[index-1])
		}
	}
}

func TestCalculateFehlbergHarmonicOscillator(t *testing.T) {
	trajectory, err := CalculateFehlberg(harmonicOscillator, []float64{1, 0}, 0, math.Pi/2, WithTolerance(1e-10))
	if err != nil {
		t.Fatalf("CalculateFehlberg: %v", err)
	}

	last := trajectory.Ys[len(trajectory.Ys)-1]
	requireClose(t, "cos(π/2)", last[0], 0, 1e-6)
	requireClose(t, "-sin(π/2)", last[1], -1, 1e-6)
}

func TestCalculateFehlbergToleranceControlsStepCount(t *testing.T) {
	calculate := func(tolerance float64) Trajectory {
		trajectory, err := CalculateFehlberg(exponential, []float64{1}, 0, 1,
			WithTolerance(tolerance), WithInitialStep(0.5))
		if err != nil {
			t.Fatalf("CalculateFehlberg(%g): %v", tolerance, err)
		}

		return trajectory
	}

	strict, loose := calculate(1e-12), calculate(1e-3)
	if strict.Steps <= loose.Steps {
		t.Fatalf("严格容差步数 = %d, 宽松容差步数 = %d, want 严格更多", strict.Steps, loose.Steps)
	}

	if strict.Rejected == 0 {
		t.Fatalf("初始步长过大时应出现拒绝，Rejected = 0")
	}
}

func TestCalculateFehlbergHonorsStepRange(t *testing.T) {
	trajectory, err := CalculateFehlberg(exponential, []float64{1}, 0, 1,
		WithTolerance(1e-12), WithStepRange(0.2, 0.3))
	if err != nil {
		t.Fatalf("CalculateFehlberg: %v", err)
	}

	for index := 1; index < len(trajectory.Xs); index++ {
		if width := trajectory.Xs[index] - trajectory.Xs[index-1]; width > 0.3+1e-12 {
			t.Fatalf("步长 %g 超过上界 0.3", width)
		}
	}

	if last := trajectory.Xs[len(trajectory.Xs)-1]; last != 1 {
		t.Fatalf("终点 = %g, want 1", last)
	}
}

func TestCalculateFehlbergNotConverged(t *testing.T) {
	_, err := CalculateFehlberg(exponential, []float64{1}, 0, 1,
		WithTolerance(1e-14), WithMaxIterations(1))
	if !errors.Is(err, ErrNotConverged) {
		t.Fatalf("err = %v, want %v", err, ErrNotConverged)
	}
}

func TestCalculateFehlbergPropagatesEvaluationErrors(t *testing.T) {
	short := func(_ float64, y []float64) []float64 { return []float64{y[0]} }
	if _, err := CalculateFehlberg(short, []float64{1, 1}, 0, 1); !errors.Is(err, ErrStateLengthChanged) {
		t.Fatalf("length change err = %v, want %v", err, ErrStateLengthChanged)
	}

	blowup := func(_ float64, _ []float64) []float64 { return []float64{math.NaN()} }
	if _, err := CalculateFehlberg(blowup, []float64{1}, 0, 1); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("non-finite err = %v, want %v", err, ErrNonFinite)
	}
}
