package distribution

import (
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
		{"非法标准差", ErrInvalidStdDev},
		{"非法区间", ErrInvalidRange},
		{"非法速率", ErrInvalidRate},
		{"非法均值", ErrInvalidMean},
		{"非法概率", ErrInvalidProbability},
		{"未收敛", ErrNotConverged},
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

func TestValidateProbability(t *testing.T) {
	tests := []struct {
		name        string
		probability float64
		want        bool
	}{
		{"正常值", 0.5, true},
		{"下边界", 0, false},
		{"上边界", 1, false},
		{"负值", -0.1, false},
		{"超过 1", 1.1, false},
		{"NaN", math.NaN(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProbability(tt.probability)
			if (err == nil) != tt.want {
				t.Fatalf("err = %v, want valid = %v", err, tt.want)
			}
		})
	}
}
