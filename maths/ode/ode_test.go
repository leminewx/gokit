package ode

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func requireClose(t *testing.T, label string, got, want, tolerance float64) {
	t.Helper()

	if math.Abs(got-want) > tolerance {
		t.Fatalf("%s = %g, want %g ± %g", label, got, want, tolerance)
	}
}

// exponential 对应 y' = y
func exponential(_ float64, y []float64) []float64 {
	return []float64{y[0]}
}

// harmonicOscillator 把 y” = -y 降阶为 [y0' = y1, y1' = -y0]
func harmonicOscillator(_ float64, y []float64) []float64 {
	return []float64{y[1], -y[0]}
}

func TestCalculateRungeKutta4Validation(t *testing.T) {
	tests := []struct {
		name     string
		function Function
		initial  []float64
		start    float64
		stop     float64
		step     float64
		want     error
	}{
		{"未提供函数", nil, []float64{1}, 0, 1, 0.1, ErrNilFunction},
		{"初始状态为空", exponential, nil, 0, 1, 0.1, ErrEmptyState},
		{"初始状态含 NaN", exponential, []float64{math.NaN()}, 0, 1, 0.1, ErrNonFinite},
		{"区间反向", exponential, []float64{1}, 1, 0, 0.1, ErrInvalidRange},
		{"区间为零", exponential, []float64{1}, 0, 0, 0.1, ErrInvalidRange},
		{"区间端点非有限", exponential, []float64{1}, 0, math.Inf(1), 0.1, ErrInvalidRange},
		{"步长为零", exponential, []float64{1}, 0, 1, 0, ErrInvalidStep},
		{"步长为负", exponential, []float64{1}, 0, 1, -0.1, ErrInvalidStep},
		{"步长非有限", exponential, []float64{1}, 0, 1, math.Inf(1), ErrInvalidStep},
		{"步数过多", exponential, []float64{1}, 0, 1, 1e-9, ErrTooManySteps},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateRungeKutta4(tt.function, tt.initial, tt.start, tt.stop, tt.step)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCalculateFehlbergValidation(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		want    error
	}{
		{"容差为零", []Option{WithTolerance(0)}, ErrInvalidTolerance},
		{"容差非有限", []Option{WithTolerance(math.NaN())}, ErrInvalidTolerance},
		{"初始步长为负", []Option{WithInitialStep(-1)}, ErrInvalidStep},
		{"初始步长非有限", []Option{WithInitialStep(math.Inf(1))}, ErrInvalidStep},
		{"步长区间反向", []Option{WithStepRange(1, 0.5)}, ErrInvalidStepRange},
		{"步长下界为负", []Option{WithStepRange(-1, 1)}, ErrInvalidStepRange},
		{"步长上界非有限", []Option{WithStepRange(0, math.Inf(1))}, ErrInvalidStepRange},
		{"迭代上限为零", []Option{WithMaxIterations(0)}, ErrInvalidIterations},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateFehlberg(exponential, []float64{1}, 0, 1, tt.options...)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCalculateFehlbergRejectsInvalidInput(t *testing.T) {
	if _, err := CalculateFehlberg(nil, []float64{1}, 0, 1); !errors.Is(err, ErrNilFunction) {
		t.Fatalf("nil function err = %v, want %v", err, ErrNilFunction)
	}

	if _, err := CalculateFehlberg(exponential, []float64{}, 0, 1); !errors.Is(err, ErrEmptyState) {
		t.Fatalf("empty state err = %v, want %v", err, ErrEmptyState)
	}

	if _, err := CalculateFehlberg(exponential, []float64{1}, 1, 1); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("empty range err = %v, want %v", err, ErrInvalidRange)
	}
}

func TestTrajectoryKeepsInputUntouched(t *testing.T) {
	initial := []float64{1, 0}
	snapshot := append([]float64(nil), initial...)

	trajectories := map[string]Trajectory{}
	trajectory, err := CalculateFehlberg(harmonicOscillator, initial, 0, 1)
	if err != nil {
		t.Fatalf("CalculateFehlberg: %v", err)
	}
	trajectories["Fehlberg"] = trajectory

	trajectory, err = CalculateRungeKutta4(harmonicOscillator, initial, 0, 1, 0.1)
	if err != nil {
		t.Fatalf("CalculateRungeKutta4: %v", err)
	}
	trajectories["RungeKutta4"] = trajectory

	for name, result := range trajectories {
		for index := range snapshot {
			if initial[index] != snapshot[index] {
				t.Fatalf("%s 改写了初始状态: %v, want %v", name, initial, snapshot)
			}
		}

		if len(result.Xs) != len(result.Ys) {
			t.Fatalf("%s Xs 与 Ys 长度不一致: %d != %d", name, len(result.Xs), len(result.Ys))
		}

		if result.Steps != len(result.Xs)-1 {
			t.Fatalf("%s Steps = %d, want %d", name, result.Steps, len(result.Xs)-1)
		}

		// 各行必须独立分配，改写首行不得影响次行
		result.Ys[0][0] = 100
		if result.Ys[1][0] == 100 {
			t.Fatalf("%s Ys 各行共享底层数组", name)
		}
	}
}

func TestErrorPrefix(t *testing.T) {
	for _, err := range []error{ErrNilFunction, ErrInvalidStep, ErrNotConverged} {
		if !strings.HasPrefix(err.Error(), "gokit.maths.ode: ") {
			t.Fatalf("%q 缺少统一前缀", err)
		}
	}
}
