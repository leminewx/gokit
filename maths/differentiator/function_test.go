package differentiator

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateFunctionDerivative(t *testing.T) {
	tests := []struct {
		name      string
		function  Function
		x         float64
		want      float64
		tolerance float64
	}{
		{"sin 在 1 处", math.Sin, 1, math.Cos(1), 1e-9},
		{"sin 在 0 处", math.Sin, 0, 1, 1e-9},
		{"exp 在 2 处", math.Exp, 2, math.Exp(2), 1e-8},
		{"三次多项式", func(x float64) float64 { return x * x * x }, 2, 12, 1e-8},
		{"线性函数", func(x float64) float64 { return 3*x - 7 }, -4, 3, 1e-12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			derivative, err := CalculateFunctionDerivative(tt.function, tt.x)
			if err != nil {
				t.Fatalf("err = %v", err)
			}

			requireClose(t, "导数值", derivative.Value, tt.want, tt.tolerance)
			if derivative.Window != _FIVE_POINT_WINDOW {
				t.Fatalf("Window = %d, want %d", derivative.Window, _FIVE_POINT_WINDOW)
			}

			if derivative.ErrorEstimate < 0 {
				t.Fatalf("ErrorEstimate = %g, want >= 0", derivative.ErrorEstimate)
			}
		})
	}
}

func TestCalculateFunctionSecondDerivative(t *testing.T) {
	tests := []struct {
		name      string
		function  Function
		x         float64
		want      float64
		tolerance float64
	}{
		{"sin 在 1 处", math.Sin, 1, -math.Sin(1), 1e-7},
		{"二次多项式", func(x float64) float64 { return 2*x*x + x }, 3, 4, 1e-8},
		{"三次多项式", func(x float64) float64 { return x * x * x }, 2, 12, 1e-7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			derivative, err := CalculateFunctionSecondDerivative(tt.function, tt.x)
			if err != nil {
				t.Fatalf("err = %v", err)
			}

			requireClose(t, "二阶导数值", derivative.Value, tt.want, tt.tolerance)
		})
	}
}

func TestCalculateFunctionDerivativeErrors(t *testing.T) {
	_, err := CalculateFunctionDerivative(nil, 0)
	if !errors.Is(err, ErrNilFunction) {
		t.Fatalf("err = %v, want %v", err, ErrNilFunction)
	}

	_, err = CalculateFunctionDerivative(math.Sin, math.NaN())
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}

	nonFinite := func(float64) float64 { return math.Inf(1) }
	_, err = CalculateFunctionDerivative(nonFinite, 0)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}

func TestCalculateFunctionDerivativeFinite(t *testing.T) {
	// 多项式导数必须为有限值，且误差指示应为很小的量
	derivative, err := CalculateFunctionDerivative(func(x float64) float64 { return 5*x*x*x*x - 2*x }, 1.5)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if math.IsInf(derivative.Value, 0) || math.IsNaN(derivative.Value) {
		t.Fatalf("Value = %g, want finite", derivative.Value)
	}

	if derivative.ErrorEstimate > 1e-6 {
		t.Fatalf("ErrorEstimate = %g, want <= 1e-6", derivative.ErrorEstimate)
	}
}
