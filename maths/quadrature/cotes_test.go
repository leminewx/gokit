package quadrature

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateCompositeCotesExactness(t *testing.T) {
	// 单面板时，n 点 Newton-Cotes 对 n-1 次多项式精确（奇数节点可达 n 次）
	tests := []struct {
		name     string
		function Function
		order    int
		want     float64
	}{
		{"梯形 x", func(x float64) float64 { return x }, 2, 0.5},
		{"Simpson x³", func(x float64) float64 { return x * x * x }, 3, 0.25},
		{"Simpson 3/8 x³", func(x float64) float64 { return x * x * x }, 4, 0.25},
		{"Boole x⁵", func(x float64) float64 { return math.Pow(x, 5) }, 5, 1.0 / 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CalculateCompositeCotes(tt.function, 0, 1, WithOrder(tt.order), WithPanels(1))
			if err != nil {
				t.Fatalf("CalculateCompositeCotes: %v", err)
			}

			requireClose(t, "Value", result.Value, tt.want, 1e-14)
		})
	}
}

func TestCalculateCompositeCotesTrapezoidError(t *testing.T) {
	// 单面板梯形对 x² 的误差恰为 1/6（0.5 - 1/3），用于确认权重与宽度缩放正确
	result, err := CalculateCompositeCotes(func(x float64) float64 { return x * x }, 0, 1,
		WithOrder(2), WithPanels(1))
	if err != nil {
		t.Fatalf("CalculateCompositeCotes: %v", err)
	}

	requireClose(t, "Value", result.Value, 0.5, 0)

	if result.ErrorEstimate != 0 {
		t.Fatalf("ErrorEstimate = %g, want 0 (未启用估计)", result.ErrorEstimate)
	}
}

func TestCalculateCompositeCotesRefined(t *testing.T) {
	result, err := CalculateCompositeCotes(math.Sin, 0, math.Pi, WithOrder(3), WithPanels(32))
	if err != nil {
		t.Fatalf("CalculateCompositeCotes: %v", err)
	}

	requireClose(t, "Value", result.Value, 2, 1e-6)
}

func TestCalculateCompositeCotesWithErrorEstimation(t *testing.T) {
	result, err := CalculateCompositeCotes(math.Sin, 0, math.Pi,
		WithOrder(3), WithPanels(4), WithErrorEstimation())
	if err != nil {
		t.Fatalf("CalculateCompositeCotes: %v", err)
	}

	if result.ErrorEstimate <= 0 {
		t.Fatalf("ErrorEstimate = %g, want > 0", result.ErrorEstimate)
	}

	// 估计值应与真实误差同量级（相差不超过两个数量级）
	actual := math.Abs(result.Value - 2)
	if actual > 0 && (result.ErrorEstimate > 100*actual || result.ErrorEstimate < actual/100) {
		t.Fatalf("ErrorEstimate = %g, actual error = %g, want same order of magnitude", result.ErrorEstimate, actual)
	}
}

func TestCalculateCompositeCotesNonFinite(t *testing.T) {
	_, err := CalculateCompositeCotes(func(x float64) float64 { return math.NaN() }, 0, 1)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}
