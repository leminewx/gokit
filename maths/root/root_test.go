package root

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// requireClose 断言两个浮点值在给定容差内相等
func requireClose(t *testing.T, label string, got, want, tolerance float64) {
	t.Helper()

	if math.IsNaN(got) || math.Abs(got-want) > tolerance {
		t.Fatalf("%s = %.15g, want %.15g (±%g)", label, got, want, tolerance)
	}
}

func TestOptionValidation(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		want    error
	}{
		{"零容差", []Option{WithTolerance(0)}, ErrInvalidTolerance},
		{"负容差", []Option{WithTolerance(-1e-9)}, ErrInvalidTolerance},
		{"NaN 容差", []Option{WithTolerance(math.NaN())}, ErrInvalidTolerance},
		{"负残差", []Option{WithResidual(-1)}, ErrInvalidResidual},
		{"NaN 残差", []Option{WithResidual(math.NaN())}, ErrInvalidResidual},
		{"零迭代", []Option{WithMaxIterations(0)}, ErrInvalidIterations},
		{"负迭代", []Option{WithMaxIterations(-3)}, ErrInvalidIterations},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateBisection(func(x float64) float64 { return x - 0.5 }, 0, 1, tt.options...)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestNilFunction(t *testing.T) {
	tests := []struct {
		name string
		call func() error
	}{
		{"二分法", func() error { _, err := CalculateBisection(nil, 0, 1); return err }},
		{"牛顿法", func() error { _, err := CalculateNewton(nil, 1); return err }},
		{"弦截法", func() error { _, err := CalculateSecant(nil, 0, 1); return err }},
		{"不动点迭代", func() error { _, err := CalculateFixedPoint(nil, 1); return err }},
		{"Muller 法", func() error { _, err := CalculateMuller(nil, 0, 1, 2); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, ErrNilFunction) {
				t.Fatalf("err = %v, want %v", err, ErrNilFunction)
			}
		})
	}
}

func TestErrorPrefix(t *testing.T) {
	_, err := CalculateBisection(nil, 0, 1)
	if err == nil {
		t.Fatal("want an error, got nil")
	}

	if !strings.HasPrefix(err.Error(), _ERROR_PREFIX) {
		t.Fatalf("error = %q, want prefix %q", err, _ERROR_PREFIX)
	}
}
