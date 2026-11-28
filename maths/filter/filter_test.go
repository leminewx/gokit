package filter

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
		{"空输入", ErrEmptyInput},
		{"非法窗口", ErrInvalidWindow},
		{"非法阈值", ErrInvalidThreshold},
		{"非法系数", ErrInvalidAlpha},
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

func TestInputValidation(t *testing.T) {
	calls := []struct {
		name string
		call func(values []float64) error
	}{
		{"滑动平均", func(values []float64) error {
			_, err := CalculateMovingAverage(values, 1)
			return err
		}},
		{"中值滤波", func(values []float64) error {
			_, err := CalculateMedian(values, 3)
			return err
		}},
		{"Hampel", func(values []float64) error {
			_, err := CalculateHampel(values, 3, 3)
			return err
		}},
		{"指数平滑", func(values []float64) error {
			_, err := CalculateExponential(values, 0.5)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name+"空输入", func(t *testing.T) {
			if err := call.call(nil); !errors.Is(err, ErrEmptyInput) {
				t.Fatalf("err = %v, want %v", err, ErrEmptyInput)
			}
		})

		t.Run(call.name+"含 NaN", func(t *testing.T) {
			if err := call.call([]float64{1, math.NaN(), 3}); !errors.Is(err, ErrNonFinite) {
				t.Fatalf("err = %v, want %v", err, ErrNonFinite)
			}
		})

		t.Run(call.name+"含 Inf", func(t *testing.T) {
			if err := call.call([]float64{1, math.Inf(-1), 3}); !errors.Is(err, ErrNonFinite) {
				t.Fatalf("err = %v, want %v", err, ErrNonFinite)
			}
		})
	}
}
