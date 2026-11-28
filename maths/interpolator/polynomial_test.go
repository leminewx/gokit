package interpolator

import (
	"errors"
	"math"
	"testing"
)

func TestNewLagrangeRecoversQuadratic(t *testing.T) {
	// y = x² - 3x + 2 在 0、1、2 处的值为 2、0、0
	polynomial, err := NewLagrange([]float64{0, 1, 2}, []float64{2, 0, 0})
	if err != nil {
		t.Fatalf("NewLagrange: %v", err)
	}

	if degree := polynomial.GetDegree(); degree != 2 {
		t.Fatalf("GetDegree() = %d, want 2", degree)
	}

	coefficients := polynomial.GetCoefficients()
	want := []float64{2, -3, 1}
	for index := range want {
		requireClose(t, "coefficient", coefficients[index], want[index], 1e-12)
	}

	for _, x := range []float64{-1, 0.5, 1.7, 4} {
		requireClose(t, "Calculate", polynomial.Calculate(x), x*x-3*x+2, 1e-12)
	}

	requireClose(t, "CalculateDerivative", polynomial.CalculateDerivative(2), 1, 1e-12)
}

func TestNewNewtonMatchesLagrange(t *testing.T) {
	xs := []float64{0, 0.5, 1, 1.5, 2}
	ys := []float64{1, 2, 0, 3, 5}

	lagrange, err := NewLagrange(xs, ys)
	if err != nil {
		t.Fatalf("NewLagrange: %v", err)
	}

	newton, err := NewNewton(xs, ys)
	if err != nil {
		t.Fatalf("NewNewton: %v", err)
	}

	for index := 0; index < len(xs); index++ {
		requireClose(t, "插值节点", lagrange.Calculate(xs[index]), ys[index], 1e-12)
		requireClose(t, "插值节点", newton.Calculate(xs[index]), ys[index], 1e-12)
	}

	for _, x := range []float64{0.25, 0.75, 1.25, 1.75, 3} {
		requireClose(t, "两种形式取值", newton.Calculate(x), lagrange.Calculate(x), 1e-9)
		requireClose(t, "两种形式导数", newton.CalculateDerivative(x), lagrange.CalculateDerivative(x), 1e-8)
	}
}

func TestNewNewtonRecoversCubic(t *testing.T) {
	xs := []float64{-1, 0, 1, 2}
	ys := make([]float64, len(xs))
	for index, x := range xs {
		ys[index] = x*x*x + 1
	}

	polynomial, err := NewNewton(xs, ys)
	if err != nil {
		t.Fatalf("NewNewton: %v", err)
	}

	coefficients := polynomial.GetCoefficients()
	want := []float64{1, 0, 0, 1}
	for index := range want {
		requireClose(t, "coefficient", coefficients[index], want[index], 1e-10)
	}
}

func TestPolynomialReturnsCopy(t *testing.T) {
	polynomial, err := NewNewton([]float64{0, 1}, []float64{1, 3})
	if err != nil {
		t.Fatalf("NewNewton: %v", err)
	}

	coefficients := polynomial.GetCoefficients()
	coefficients[0] = 100

	requireClose(t, "Calculate", polynomial.Calculate(0), 1, 1e-12)
}

func TestPolynomialValidation(t *testing.T) {
	tests := []struct {
		name string
		xs   []float64
		ys   []float64
		want error
	}{
		{"长度不一致", []float64{0, 1}, []float64{1}, ErrLengthMismatch},
		{"样本过少", []float64{0}, []float64{1}, ErrTooFewSamples},
		{"横坐标重复", []float64{0, 0}, []float64{1, 2}, ErrNotStrictlyIncreasing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewNewton(tt.xs, tt.ys); !errors.Is(err, tt.want) {
				t.Fatalf("NewNewton err = %v, want %v", err, tt.want)
			}

			if _, err := NewLagrange(tt.xs, tt.ys); !errors.Is(err, tt.want) {
				t.Fatalf("NewLagrange err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestPolynomialRejectsTooManySamples(t *testing.T) {
	xs := make([]float64, 65)
	ys := make([]float64, 65)
	for index := range xs {
		xs[index] = float64(index)
		ys[index] = math.Sin(xs[index])
	}

	if _, err := NewNewton(xs, ys); !errors.Is(err, ErrTooManySamples) {
		t.Fatalf("err = %v, want %v", err, ErrTooManySamples)
	}
}
