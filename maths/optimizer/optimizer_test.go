package optimizer

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

func TestErrorPrefix(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"nil 函数", ErrNilFunction},
		{"空初值", ErrEmptyInitial},
		{"非法区间", ErrInvalidRange},
		{"非法容差", ErrInvalidTolerance},
		{"非法迭代", ErrInvalidIterations},
		{"未收敛", ErrNotConverged},
		{"非有限值", ErrNonFinite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.HasPrefix(tt.err.Error(), _ERROR_PREFIX) {
				t.Fatalf("%v 缺少错误前缀", tt.err)
			}
		})
	}
}

func TestNilFunction(t *testing.T) {
	if _, err := MinimizeBrent(nil, 0, 1); !errors.Is(err, ErrNilFunction) {
		t.Fatalf("err = %v, want %v", err, ErrNilFunction)
	}

	if _, err := MinimizeNelderMead(nil, []float64{0}); !errors.Is(err, ErrNilFunction) {
		t.Fatalf("err = %v, want %v", err, ErrNilFunction)
	}
}

func TestOptionValidation(t *testing.T) {
	function := func(x float64) float64 { return x * x }

	tests := []struct {
		name    string
		options []Option
		want    error
	}{
		{"零容差", []Option{WithTolerance(0)}, ErrInvalidTolerance},
		{"负容差", []Option{WithTolerance(-1e-8)}, ErrInvalidTolerance},
		{"NaN 容差", []Option{WithTolerance(math.NaN())}, ErrInvalidTolerance},
		{"零迭代", []Option{WithMaxIterations(0)}, ErrInvalidIterations},
		{"负迭代", []Option{WithMaxIterations(-1)}, ErrInvalidIterations},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MinimizeBrent(function, -1, 1, tt.options...)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}
