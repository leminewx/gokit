package interpolator

import (
	"errors"
	"math"
	"testing"
)

func TestCubicSplineInterpolatesNodes(t *testing.T) {
	cubic := func(x float64) float64 { return x*x*x - 2*x + 1 }
	slope := func(x float64) float64 { return 3*x*x - 2 }
	xs, ys := samplesOf(cubic, -1, 2, 6)

	tests := []struct {
		name      string
		condition BoundaryCondition
		options   []Option
	}{
		{"自然边界", BOUNDARY_NATURAL, nil},
		{"夹持边界", BOUNDARY_CLAMPED, []Option{WithClampedDerivatives(slope(xs[0]), slope(xs[len(xs)-1]))}},
		{"非扭结边界", BOUNDARY_NOT_A_KNOT, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spline, err := NewCubicSpline(xs, ys, tt.condition, tt.options...)
			if err != nil {
				t.Fatalf("NewCubicSpline: %v", err)
			}

			for index := range xs {
				value, err := spline.Calculate(xs[index])
				if err != nil {
					t.Fatalf("Calculate(%g): %v", xs[index], err)
				}

				requireClose(t, "节点值", value, ys[index], 1e-10)
			}
		})
	}
}

func TestCubicSplineReproducesCubic(t *testing.T) {
	cubic := func(x float64) float64 { return x*x*x - 2*x + 1 }
	slope := func(x float64) float64 { return 3*x*x - 2 }
	xs, ys := samplesOf(cubic, -1, 2, 6)

	tests := []struct {
		name      string
		condition BoundaryCondition
		options   []Option
	}{
		{"夹持边界", BOUNDARY_CLAMPED, []Option{WithClampedDerivatives(slope(xs[0]), slope(xs[len(xs)-1]))}},
		{"非扭结边界", BOUNDARY_NOT_A_KNOT, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spline, err := NewCubicSpline(xs, ys, tt.condition, tt.options...)
			if err != nil {
				t.Fatalf("NewCubicSpline: %v", err)
			}

			for _, x := range []float64{-0.7, 0.3, 0.85, 1.99} {
				value, err := spline.Calculate(x)
				if err != nil {
					t.Fatalf("Calculate(%g): %v", x, err)
				}

				requireClose(t, "三次多项式", value, cubic(x), 1e-9)

				derivative, err := spline.CalculateDerivative(x)
				if err != nil {
					t.Fatalf("CalculateDerivative(%g): %v", x, err)
				}

				requireClose(t, "三次多项式导数", derivative, slope(x), 1e-8)
			}
		})
	}
}

func TestCubicSplineNaturalBoundary(t *testing.T) {
	xs, ys := samplesOf(math.Sin, 0, 2*math.Pi, 7)
	spline, err := NewCubicSpline(xs, ys, BOUNDARY_NATURAL)
	if err != nil {
		t.Fatalf("NewCubicSpline: %v", err)
	}

	for _, x := range []float64{xs[0], xs[len(xs)-1]} {
		curvature, err := spline.CalculateSecondDerivative(x)
		if err != nil {
			t.Fatalf("CalculateSecondDerivative(%g): %v", x, err)
		}

		requireClose(t, "自然边界二阶导数", curvature, 0, 1e-12)
	}
}

func TestCubicSplineClampedBoundary(t *testing.T) {
	xs, ys := samplesOf(math.Sin, 0, 2*math.Pi, 7)
	left, right := math.Cos(xs[0]), math.Cos(xs[len(xs)-1])

	spline, err := NewCubicSpline(xs, ys, BOUNDARY_CLAMPED, WithClampedDerivatives(left, right))
	if err != nil {
		t.Fatalf("NewCubicSpline: %v", err)
	}

	for index, want := range map[int]float64{0: left, len(xs) - 1: right} {
		derivative, err := spline.CalculateDerivative(xs[index])
		if err != nil {
			t.Fatalf("CalculateDerivative(%g): %v", xs[index], err)
		}

		requireClose(t, "夹持边界导数", derivative, want, 1e-10)
	}
}

func TestCubicSplineSecondDerivativeContinuous(t *testing.T) {
	xs, ys := samplesOf(math.Sin, 0, 2*math.Pi, 9)
	spline, err := NewCubicSpline(xs, ys, BOUNDARY_NATURAL)
	if err != nil {
		t.Fatalf("NewCubicSpline: %v", err)
	}

	const offset = 1e-9
	for index := 1; index < len(xs)-1; index++ {
		left, err := spline.CalculateSecondDerivative(xs[index] - offset)
		if err != nil {
			t.Fatalf("CalculateSecondDerivative: %v", err)
		}

		right, err := spline.CalculateSecondDerivative(xs[index] + offset)
		if err != nil {
			t.Fatalf("CalculateSecondDerivative: %v", err)
		}

		requireClose(t, "二阶导数跳变", right, left, 1e-6)
	}
}

func TestCubicSplineDefaultRejectsExtrapolation(t *testing.T) {
	xs, ys := samplesOf(math.Sin, 0, 2*math.Pi, 9)
	spline, err := NewCubicSpline(xs, ys, BOUNDARY_NATURAL)
	if err != nil {
		t.Fatalf("NewCubicSpline: %v", err)
	}

	if _, err := spline.Calculate(-0.5); !errors.Is(err, ErrOutOfRange) {
		t.Fatalf("Calculate err = %v, want %v", err, ErrOutOfRange)
	}

	if _, err := spline.CalculateDerivative(7); !errors.Is(err, ErrOutOfRange) {
		t.Fatalf("CalculateDerivative err = %v, want %v", err, ErrOutOfRange)
	}

	if _, err := spline.CalculateSecondDerivative(7); !errors.Is(err, ErrOutOfRange) {
		t.Fatalf("CalculateSecondDerivative err = %v, want %v", err, ErrOutOfRange)
	}
}

func TestCubicSplineExtrapolationPolicies(t *testing.T) {
	// 线性数据在自然边界下二阶导数恒为 0，样条退化为直线，便于验证外推策略
	xs, ys := samplesOf(func(x float64) float64 { return 2*x + 1 }, 0, 2, 5)

	tests := []struct {
		name      string
		policy    Extrapolation
		wantValue float64
		wantSlope float64
	}{
		{"线性外推", EXTRAPOLATION_LINEAR, 6, 2},
		{"三次外推", EXTRAPOLATION_CUBIC, 6, 2},
		{"常值延拓", EXTRAPOLATION_CLAMP, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spline, err := NewCubicSpline(xs, ys, BOUNDARY_NATURAL, WithExtrapolation(tt.policy))
			if err != nil {
				t.Fatalf("NewCubicSpline: %v", err)
			}

			value, err := spline.Calculate(2.5)
			if err != nil {
				t.Fatalf("Calculate: %v", err)
			}

			requireClose(t, "右侧外推值", value, tt.wantValue, 1e-9)

			derivative, err := spline.CalculateDerivative(2.5)
			if err != nil {
				t.Fatalf("CalculateDerivative: %v", err)
			}

			requireClose(t, "右侧外推导数", derivative, tt.wantSlope, 1e-9)

			curvature, err := spline.CalculateSecondDerivative(2.5)
			if err != nil {
				t.Fatalf("CalculateSecondDerivative: %v", err)
			}

			requireClose(t, "右侧外推二阶导数", curvature, 0, 1e-9)
		})
	}

	// 左侧外推：y(-1) = -1
	spline, err := NewCubicSpline(xs, ys, BOUNDARY_NATURAL, WithExtrapolation(EXTRAPOLATION_LINEAR))
	if err != nil {
		t.Fatalf("NewCubicSpline: %v", err)
	}

	value, err := spline.Calculate(-1)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	requireClose(t, "左侧外推值", value, -1, 1e-9)

	clamped, err := NewCubicSpline(xs, ys, BOUNDARY_NATURAL, WithExtrapolation(EXTRAPOLATION_CLAMP))
	if err != nil {
		t.Fatalf("NewCubicSpline: %v", err)
	}

	leftValue, err := clamped.Calculate(-1)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	requireClose(t, "左侧常值", leftValue, 1, 1e-12)
}

func TestCubicSplineCubicExtrapolationExact(t *testing.T) {
	cubic := func(x float64) float64 { return x*x*x - 2*x + 1 }
	slope := func(x float64) float64 { return 3*x*x - 2 }
	xs, ys := samplesOf(cubic, 0, 2, 5)

	spline, err := NewCubicSpline(xs, ys, BOUNDARY_CLAMPED,
		WithClampedDerivatives(slope(xs[0]), slope(xs[len(xs)-1])),
		WithExtrapolation(EXTRAPOLATION_CUBIC))
	if err != nil {
		t.Fatalf("NewCubicSpline: %v", err)
	}

	value, err := spline.Calculate(2.5)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	requireClose(t, "三次外推", value, cubic(2.5), 1e-9)
}

func TestCubicSplinePeriodicWrap(t *testing.T) {
	xs, ys := samplesOf(math.Sin, 0, 2*math.Pi, 33)

	spline, err := NewCubicSpline(xs, ys, BOUNDARY_PERIODIC, WithExtrapolation(EXTRAPOLATION_WRAP))
	if err != nil {
		t.Fatalf("NewCubicSpline: %v", err)
	}

	tests := []struct {
		name string
		x    float64
		want float64
	}{
		{"右侧环绕", 2*math.Pi + 0.5, math.Sin(0.5)},
		{"左侧环绕", -0.5, math.Sin(-0.5)},
		{"跨两个周期", 4*math.Pi + 1, math.Sin(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := spline.Calculate(tt.x)
			if err != nil {
				t.Fatalf("Calculate: %v", err)
			}

			requireClose(t, "环绕值", value, tt.want, 5e-5)
		})
	}

	// 未启用环绕时越界必须报错
	strict, err := NewCubicSpline(xs, ys, BOUNDARY_PERIODIC)
	if err != nil {
		t.Fatalf("NewCubicSpline: %v", err)
	}

	if _, err := strict.Calculate(2*math.Pi + 0.5); !errors.Is(err, ErrOutOfRange) {
		t.Fatalf("Calculate err = %v, want %v", err, ErrOutOfRange)
	}
}

func TestCubicSplineNonFinite(t *testing.T) {
	xs, ys := samplesOf(math.Sin, 0, 1, 5)
	spline, err := NewCubicSpline(xs, ys, BOUNDARY_NATURAL)
	if err != nil {
		t.Fatalf("NewCubicSpline: %v", err)
	}

	if _, err := spline.Calculate(math.NaN()); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("Calculate err = %v, want %v", err, ErrNonFinite)
	}

	if _, err := spline.CalculateDerivative(math.Inf(1)); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("CalculateDerivative err = %v, want %v", err, ErrNonFinite)
	}

	if _, err := spline.CalculateSecondDerivative(math.Inf(-1)); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("CalculateSecondDerivative err = %v, want %v", err, ErrNonFinite)
	}
}
