package differentiator

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
		{"空输入", ErrEmptyInput},
		{"长度不一致", ErrLengthMismatch},
		{"样本不足", ErrTooFewSamples},
		{"横坐标非递增", ErrNotStrictlyIncreasing},
		{"非法阶数", ErrInvalidOrder},
		{"非法窗口", ErrInvalidWindow},
		{"越界", ErrOutOfRange},
		{"非有限值", ErrNonFinite},
		{"奇异方程", ErrSingularSystem},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.HasPrefix(tt.err.Error(), _ERROR_PREFIX) {
				t.Fatalf("%v 缺少错误前缀", tt.err)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	_, err := NewSampler([]float64{0, 1}, []float64{0, 1, 2})
	if !errors.Is(err, ErrLengthMismatch) {
		t.Fatalf("err = %v, want %v", err, ErrLengthMismatch)
	}

	if !strings.Contains(err.Error(), "2 != 3") {
		t.Fatalf("err = %v, 缺少上下文", err)
	}
}
