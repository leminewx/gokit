package quadrature

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
		{"负容差", []Option{WithTolerance(-1)}, ErrInvalidTolerance},
		{"NaN 容差", []Option{WithTolerance(math.NaN())}, ErrInvalidTolerance},
		{"零迭代", []Option{WithMaxIterations(0)}, ErrInvalidIterations},
		{"零分区", []Option{WithPanels(0)}, ErrInvalidPanels},
		{"节点数过小", []Option{WithOrder(1)}, ErrInvalidOrder},
		{"节点数过大", []Option{WithOrder(6)}, ErrInvalidOrder},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateRomberg(func(x float64) float64 { return x }, 0, 1, tt.options...)
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
		{"Romberg", func() error { _, err := CalculateRomberg(nil, 0, 1); return err }},
		{"复化 Cotes", func() error { _, err := CalculateCompositeCotes(nil, 0, 1); return err }},
		{"自适应 Cotes", func() error { _, err := CalculateAdaptiveCotes(nil, 0, 1); return err }},
		{"Gauss-Legendre", func() error { _, err := CalculateGaussLegendre(nil, 0, 1); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, ErrNilFunction) {
				t.Fatalf("err = %v, want %v", err, ErrNilFunction)
			}
		})
	}
}

func TestInvalidInterval(t *testing.T) {
	tests := []struct {
		name string
		a, b float64
	}{
		{"端点相同", 1, 1},
		{"区间反向", 2, 1},
		{"左端点非有限", math.Inf(-1), 1},
		{"右端点非有限", 0, math.NaN()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateRomberg(func(x float64) float64 { return x }, tt.a, tt.b)
			if !errors.Is(err, ErrInvalidInterval) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidInterval)
			}
		})
	}
}

func TestErrorPrefix(t *testing.T) {
	_, err := CalculateRomberg(nil, 0, 1)
	if err == nil {
		t.Fatal("want an error, got nil")
	}

	if !strings.HasPrefix(err.Error(), _ERROR_PREFIX) {
		t.Fatalf("error = %q, want prefix %q", err, _ERROR_PREFIX)
	}
}
