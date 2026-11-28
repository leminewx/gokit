package activator

import (
	"math"
	"testing"
)

func TestSoftmaxEmpty(t *testing.T) {
	t.Parallel()

	if got := CalculateSoftmax(nil); len(got) != 0 {
		t.Errorf("CalculateSoftmax(nil) 长度 = %d, want 0", len(got))
	}
	if got := CalculateSoftmax([]float64{}); len(got) != 0 {
		t.Errorf("CalculateSoftmax(空切片) 长度 = %d, want 0", len(got))
	}
}

func TestSoftmaxSumToOne(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []float64
	}{
		{name: "single", in: []float64{7}},
		{name: "uniform", in: []float64{1, 1, 1}},
		{name: "ascending", in: []float64{1, 2, 3}},
		{name: "large values", in: []float64{1000, 1001, 999}},
		{name: "negative values", in: []float64{-5, -10, -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			probes := CalculateSoftmax(tt.in)
			if len(probes) != len(tt.in) {
				t.Fatalf("CalculateSoftmax 长度 = %d, want %d", len(probes), len(tt.in))
			}

			sum := 0.0
			for _, val := range probes {
				if val < 0 || val > 1 {
					t.Errorf("概率越界：%v", val)
				}
				sum += val
			}
			if math.Abs(sum-1) > 1e-9 {
				t.Errorf("概率和 = %v, want 1", sum)
			}
		})
	}
}

func TestSoftmaxKeepsOrdering(t *testing.T) {
	t.Parallel()

	probes := CalculateSoftmax([]float64{1, 2, 3})
	if probes[2] <= probes[1] || probes[1] <= probes[0] {
		t.Errorf("CalculateSoftmax 未保持单调性：%v", probes)
	}
}

func TestSoftmaxAllNegInf(t *testing.T) {
	t.Parallel()

	probes := CalculateSoftmax([]float64{math.Inf(-1), math.Inf(-1)})
	if len(probes) != 2 {
		t.Fatalf("长度 = %d, want 2", len(probes))
	}
	for i, val := range probes {
		if math.Abs(val-0.5) > 1e-9 {
			t.Errorf("probes[%d] = %v, want 0.5", i, val)
		}
	}
}

func TestSoftmaxBackward(t *testing.T) {
	t.Parallel()

	probes := []float64{0.1, 0.6, 0.3}
	grad := CalculateSoftmaxBackward(probes, 1)
	want := []float64{0.1, -0.4, 0.3}
	for i := range want {
		if math.Abs(grad[i]-want[i]) > 1e-9 {
			t.Errorf("grad[%d] = %v, want %v", i, grad[i], want[i])
		}
	}
}

func TestSoftmaxBackwardInvalidLabel(t *testing.T) {
	t.Parallel()

	for _, label := range []int{-1, 3} {
		grad := CalculateSoftmaxBackward([]float64{0.2, 0.5, 0.3}, label)
		for i, val := range grad {
			if val != 0 {
				t.Errorf("label=%d 时 grad[%d] = %v, want 0", label, i, val)
			}
		}
	}
}

func BenchmarkSoftmax(b *testing.B) {
	inputs := []float64{1, 2, 3, 4, 5}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = CalculateSoftmax(inputs)
	}
}
