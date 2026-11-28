package interpolator

import (
	"errors"
	"math"
	"testing"
)

func TestNewHermiteInterpolatesSin(t *testing.T) {
	xs, ys := samplesOf(math.Sin, 0, 2*math.Pi, 17)
	derivatives := make([]float64, len(xs))
	for index, x := range xs {
		derivatives[index] = math.Cos(x)
	}

	interpolator, err := NewHermite(xs, ys, derivatives)
	if err != nil {
		t.Fatalf("NewHermite: %v", err)
	}

	for index := range xs {
		value, err := interpolator.Calculate(xs[index])
		if err != nil {
			t.Fatalf("Calculate(%g): %v", xs[index], err)
		}

		requireClose(t, "节点值", value, ys[index], 1e-12)

		derivative, err := interpolator.CalculateDerivative(xs[index])
		if err != nil {
			t.Fatalf("CalculateDerivative(%g): %v", xs[index], err)
		}

		requireClose(t, "节点导数", derivative, derivatives[index], 1e-10)
	}

	value, err := interpolator.Calculate(1)
	if err != nil {
		t.Fatalf("Calculate(1): %v", err)
	}

	requireClose(t, "区间内精度", value, math.Sin(1), 1e-4)
}

func TestNewHermiteReproducesCubic(t *testing.T) {
	cubic := func(x float64) float64 { return x*x*x - 2*x + 1 }
	slope := func(x float64) float64 { return 3*x*x - 2 }

	xs, ys := samplesOf(cubic, -1, 2, 4)
	derivatives := make([]float64, len(xs))
	for index, x := range xs {
		derivatives[index] = slope(x)
	}

	interpolator, err := NewHermite(xs, ys, derivatives)
	if err != nil {
		t.Fatalf("NewHermite: %v", err)
	}

	for _, x := range []float64{-0.4, 0.5, 1.9} {
		value, err := interpolator.Calculate(x)
		if err != nil {
			t.Fatalf("Calculate(%g): %v", x, err)
		}

		requireClose(t, "三次多项式", value, cubic(x), 1e-9)

		derivative, err := interpolator.CalculateDerivative(x)
		if err != nil {
			t.Fatalf("CalculateDerivative(%g): %v", x, err)
		}

		requireClose(t, "三次多项式导数", derivative, slope(x), 1e-8)
	}
}

func TestNewHermiteRejectsInvalidDerivatives(t *testing.T) {
	tests := []struct {
		name        string
		derivatives []float64
		want        error
	}{
		{"长度不一致", []float64{1, 1}, ErrLengthMismatch},
		{"含 NaN", []float64{1, math.NaN(), 1}, ErrNonFinite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewHermite([]float64{0, 1, 2}, []float64{0, 1, 2}, tt.derivatives)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestHermiteRejectsOutOfRange(t *testing.T) {
	interpolator, err := NewHermite([]float64{0, 1, 2}, []float64{0, 1, 4}, []float64{0, 2, 4})
	if err != nil {
		t.Fatalf("NewHermite: %v", err)
	}

	if _, err := interpolator.Calculate(-0.5); !errors.Is(err, ErrOutOfRange) {
		t.Fatalf("Calculate err = %v, want %v", err, ErrOutOfRange)
	}

	if _, err := interpolator.CalculateDerivative(2.5); !errors.Is(err, ErrOutOfRange) {
		t.Fatalf("CalculateDerivative err = %v, want %v", err, ErrOutOfRange)
	}

	if _, err := interpolator.Calculate(math.NaN()); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("Calculate err = %v, want %v", err, ErrNonFinite)
	}
}

func TestHermiteCubicAccuracy(t *testing.T) {
	// x² 的导数线性，Hermite 分段三次对四次以下多项式在节点稀疏时也有较好精度
	xs := []float64{0, 0.5, 1}
	ys := []float64{0, 0.25, 1}
	derivatives := []float64{0, 1, 2}

	interpolator, err := NewHermite(xs, ys, derivatives)
	if err != nil {
		t.Fatalf("NewHermite: %v", err)
	}

	for _, x := range []float64{0.25, 0.75} {
		value, err := interpolator.Calculate(x)
		if err != nil {
			t.Fatalf("Calculate(%g): %v", x, err)
		}

		requireClose(t, "二次多项式", value, x*x, 1e-12)
	}
}
