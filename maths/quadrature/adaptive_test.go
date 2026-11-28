package quadrature

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateAdaptiveCotes(t *testing.T) {
	tests := []struct {
		name      string
		function  Function
		a, b      float64
		want      float64
		tolerance float64
	}{
		{"4/(1+x²)", func(x float64) float64 { return 4 / (1 + x*x) }, 0, 1, math.Pi, 1e-10},
		{"exp", math.Exp, 0, 1, math.E - 1, 1e-10},
		{"x⁴", func(x float64) float64 { return math.Pow(x, 4) }, 0, 1, 0.2, 1e-12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CalculateAdaptiveCotes(tt.function, tt.a, tt.b)
			if err != nil {
				t.Fatalf("CalculateAdaptiveCotes: %v", err)
			}

			requireClose(t, "Value", result.Value, tt.want, tt.tolerance)

			if result.ErrorEstimate < 0 || result.ErrorEstimate > _DEFAULT_TOLERANCE {
				t.Fatalf("ErrorEstimate = %g, want within [0, %g]", result.ErrorEstimate, _DEFAULT_TOLERANCE)
			}

			if result.Iterations < 1 {
				t.Fatalf("Iterations = %d, want >= 1", result.Iterations)
			}
		})
	}
}

func TestCalculateAdaptiveCotesSubdividesSingularEndpoint(t *testing.T) {
	// 端点导数无界（sqrt 在 0 处），自适应法必须细分出多个叶子才能逼近
	result, err := CalculateAdaptiveCotes(math.Sqrt, 0, 1, WithTolerance(1e-9))
	if err != nil {
		t.Fatalf("CalculateAdaptiveCotes: %v", err)
	}

	requireClose(t, "Value", result.Value, 2.0/3, 1e-5)

	if result.Iterations <= 1 {
		t.Fatalf("Iterations = %d, want > 1", result.Iterations)
	}
}

func TestCalculateAdaptiveCotesRespectsDepthLimit(t *testing.T) {
	// 深度上限为 1 时只允许一次细分，结果精度明显低于默认深度
	shallow, err := CalculateAdaptiveCotes(func(x float64) float64 { return 1 / (1 + 100*x*x) }, 0, 1,
		WithTolerance(1e-12), WithMaxIterations(1))
	if err != nil {
		t.Fatalf("CalculateAdaptiveCotes: %v", err)
	}

	deep, err := CalculateAdaptiveCotes(func(x float64) float64 { return 1 / (1 + 100*x*x) }, 0, 1,
		WithTolerance(1e-12))
	if err != nil {
		t.Fatalf("CalculateAdaptiveCotes: %v", err)
	}

	if deep.Iterations <= shallow.Iterations {
		t.Fatalf("Iterations deep = %d, want > %d (shallow)", deep.Iterations, shallow.Iterations)
	}

	// 二者都应逼近解析结果 arctan(10)/10
	exact := math.Atan(10) / 10
	if math.Abs(deep.Value-exact) > math.Abs(shallow.Value-exact) {
		t.Fatalf("deep error %g, want <= shallow error %g", math.Abs(deep.Value-exact), math.Abs(shallow.Value-exact))
	}
}

func TestCalculateAdaptiveCotesNonFinite(t *testing.T) {
	_, err := CalculateAdaptiveCotes(func(x float64) float64 { return math.Inf(1) }, 0, 1)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}
