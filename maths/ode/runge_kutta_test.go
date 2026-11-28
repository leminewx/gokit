package ode

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateRungeKutta4Exponential(t *testing.T) {
	trajectory, err := CalculateRungeKutta4(exponential, []float64{1}, 0, 1, 0.1)
	if err != nil {
		t.Fatalf("CalculateRungeKutta4: %v", err)
	}

	if len(trajectory.Xs) != 11 || trajectory.Steps != 10 {
		t.Fatalf("len(Xs) = %d, Steps = %d, want 11, 10", len(trajectory.Xs), trajectory.Steps)
	}

	if trajectory.Xs[len(trajectory.Xs)-1] != 1 {
		t.Fatalf("终点 = %g, want 1", trajectory.Xs[len(trajectory.Xs)-1])
	}

	requireClose(t, "y(1)", trajectory.Ys[len(trajectory.Ys)-1][0], math.E, 1e-5)
}

func TestCalculateRungeKutta4TruncatesLastStep(t *testing.T) {
	trajectory, err := CalculateRungeKutta4(exponential, []float64{1}, 0, 1, 0.3)
	if err != nil {
		t.Fatalf("CalculateRungeKutta4: %v", err)
	}

	want := []float64{0, 0.3, 0.6, 0.9, 1}
	if len(trajectory.Xs) != len(want) {
		t.Fatalf("len(Xs) = %d, want %d", len(trajectory.Xs), len(want))
	}

	for index := range want {
		requireClose(t, "节点", trajectory.Xs[index], want[index], 1e-12)
	}

	requireClose(t, "y(1)", trajectory.Ys[len(trajectory.Ys)-1][0], math.E, 1e-3)
}

func TestCalculateRungeKutta4HarmonicOscillator(t *testing.T) {
	trajectory, err := CalculateRungeKutta4(harmonicOscillator, []float64{1, 0}, 0, math.Pi/2, 0.01)
	if err != nil {
		t.Fatalf("CalculateRungeKutta4: %v", err)
	}

	last := trajectory.Ys[len(trajectory.Ys)-1]
	requireClose(t, "cos(π/2)", last[0], 0, 1e-7)
	requireClose(t, "-sin(π/2)", last[1], -1, 1e-7)
}

func TestCalculateRungeKutta4RefinementOrder(t *testing.T) {
	calculate := func(step float64) float64 {
		trajectory, err := CalculateRungeKutta4(exponential, []float64{1}, 0, 1, step)
		if err != nil {
			t.Fatalf("CalculateRungeKutta4(%g): %v", step, err)
		}

		return math.Abs(trajectory.Ys[len(trajectory.Ys)-1][0] - math.E)
	}

	coarse, fine := calculate(0.1), calculate(0.05)
	if coarse == 0 || fine == 0 {
		t.Fatalf("误差为零，无法判定收敛阶: %g, %g", coarse, fine)
	}

	// 四阶方法的全局误差随步长减半应缩小约 16 倍
	ratio := coarse / fine
	if ratio < 10 || ratio > 25 {
		t.Fatalf("误差比 = %g, want 10..25", ratio)
	}
}

func TestCalculateRungeKutta4StepOneStep(t *testing.T) {
	// y' = -2xy，精确解 y = exp(-x²)
	function := func(x float64, y []float64) []float64 {
		return []float64{-2 * x * y[0]}
	}

	next, err := CalculateRungeKutta4Step(function, 0, []float64{1}, 0.1)
	if err != nil {
		t.Fatalf("CalculateRungeKutta4Step: %v", err)
	}

	requireClose(t, "单步", next[0], math.Exp(-0.01), 1e-6)

	// 返回值不得与输入共享底层数组
	state := []float64{1}
	if _, err := CalculateRungeKutta4Step(function, 0, state, 0.1); err != nil {
		t.Fatalf("CalculateRungeKutta4Step: %v", err)
	}

	if state[0] != 1 {
		t.Fatalf("初始状态被改写: %v", state)
	}
}

func TestCalculateRungeKutta4StepValidation(t *testing.T) {
	if _, err := CalculateRungeKutta4Step(nil, 0, []float64{1}, 0.1); !errors.Is(err, ErrNilFunction) {
		t.Fatalf("nil function err = %v, want %v", err, ErrNilFunction)
	}

	if _, err := CalculateRungeKutta4Step(exponential, 0, nil, 0.1); !errors.Is(err, ErrEmptyState) {
		t.Fatalf("empty state err = %v, want %v", err, ErrEmptyState)
	}

	if _, err := CalculateRungeKutta4Step(exponential, math.NaN(), []float64{1}, 0.1); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("non-finite x err = %v, want %v", err, ErrNonFinite)
	}

	if _, err := CalculateRungeKutta4Step(exponential, 0, []float64{1}, 0); !errors.Is(err, ErrInvalidStep) {
		t.Fatalf("zero step err = %v, want %v", err, ErrInvalidStep)
	}

	short := func(_ float64, y []float64) []float64 { return []float64{y[0]} }
	if _, err := CalculateRungeKutta4Step(short, 0, []float64{1, 1}, 0.1); !errors.Is(err, ErrStateLengthChanged) {
		t.Fatalf("length change err = %v, want %v", err, ErrStateLengthChanged)
	}

	blowup := func(_ float64, _ []float64) []float64 { return []float64{math.Inf(1)} }
	if _, err := CalculateRungeKutta4Step(blowup, 0, []float64{1}, 0.1); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("non-finite derivative err = %v, want %v", err, ErrNonFinite)
	}
}
