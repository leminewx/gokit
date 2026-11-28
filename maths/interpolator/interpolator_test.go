package interpolator

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

// samplesOf 生成 [start, stop] 上等距分布的 count 个采样点及其函数值
func samplesOf(function func(float64) float64, start, stop float64, count int) ([]float64, []float64) {
	xs := make([]float64, count)
	ys := make([]float64, count)
	for index := range xs {
		xs[index] = start + (stop-start)*float64(index)/float64(count-1)
		ys[index] = function(xs[index])
	}

	return xs, ys
}

func TestSampleValidation(t *testing.T) {
	tests := []struct {
		name string
		xs   []float64
		ys   []float64
		want error
	}{
		{"长度不一致", []float64{0, 1, 2}, []float64{0, 1}, ErrLengthMismatch},
		{"样本过少", []float64{0}, []float64{1}, ErrTooFewSamples},
		{"空样本", nil, nil, ErrTooFewSamples},
		{"横坐标重复", []float64{0, 1, 1}, []float64{0, 1, 2}, ErrNotStrictlyIncreasing},
		{"横坐标递减", []float64{0, 2, 1}, []float64{0, 1, 2}, ErrNotStrictlyIncreasing},
		{"含 NaN", []float64{0, math.NaN()}, []float64{0, 1}, ErrNonFinite},
		{"含 Inf", []float64{0, 1}, []float64{0, math.Inf(1)}, ErrNonFinite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewCubicSpline(tt.xs, tt.ys, BOUNDARY_NATURAL); !errors.Is(err, tt.want) {
				t.Fatalf("NewCubicSpline err = %v, want %v", err, tt.want)
			}

			if _, err := NewHermite(tt.xs, tt.ys, tt.ys); !errors.Is(err, tt.want) {
				t.Fatalf("NewHermite err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestBoundaryValidation(t *testing.T) {
	xs, ys := samplesOf(math.Sin, 0, 2*math.Pi, 5)

	tests := []struct {
		name      string
		xs, ys    []float64
		condition BoundaryCondition
		options   []Option
		want      error
	}{
		{"夹持缺少导数", xs, ys, BOUNDARY_CLAMPED, nil, ErrMissingDerivative},
		{"周期端点不等", xs, []float64{0, 1, 2, 3, 4}, BOUNDARY_PERIODIC, nil, ErrInvalidBoundary},
		{"周期样本过少", []float64{0, 1, 2}, []float64{0, 1, 0}, BOUNDARY_PERIODIC, nil, ErrInvalidBoundary},
		{"非扭结样本过少", []float64{0, 1, 2}, []float64{0, 1, 4}, BOUNDARY_NOT_A_KNOT, nil, ErrInvalidBoundary},
		{"未知边界条件", xs, ys, BoundaryCondition(9), nil, ErrInvalidBoundary},
		{"未知外推策略", xs, ys, BOUNDARY_NATURAL, []Option{WithExtrapolation(Extrapolation(9))}, ErrInvalidBoundary},
		{"非周期边界用环绕", xs, ys, BOUNDARY_NATURAL, []Option{WithExtrapolation(EXTRAPOLATION_WRAP)}, ErrInvalidBoundary},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCubicSpline(tt.xs, tt.ys, tt.condition, tt.options...)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestClampedDerivativesNonFinite(t *testing.T) {
	xs, ys := samplesOf(math.Sin, 0, 2*math.Pi, 5)
	_, err := NewCubicSpline(xs, ys, BOUNDARY_CLAMPED,
		WithClampedDerivatives(math.NaN(), 1))
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}

func TestErrorPrefix(t *testing.T) {
	_, err := NewCubicSpline([]float64{0}, []float64{0}, BOUNDARY_NATURAL)
	if err == nil {
		t.Fatal("want an error, got nil")
	}

	if !strings.HasPrefix(err.Error(), _ERROR_PREFIX) {
		t.Fatalf("error = %q, want prefix %q", err, _ERROR_PREFIX)
	}
}
