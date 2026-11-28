package learning

import (
	"math"
	"testing"
)

func TestCalculateEuclideanDistance(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    []float64
		b    []float64
		want float64
	}{
		{name: "同一点", a: []float64{1, 2}, b: []float64{1, 2}, want: 0},
		{name: "二维直角边", a: []float64{0, 0}, b: []float64{3, 4}, want: 5},
		{name: "一维", a: []float64{-1}, b: []float64{2}, want: 3},
		{name: "三维", a: []float64{0, 0, 0}, b: []float64{1, 2, 2}, want: 3},
	}

	for _, tc := range cases {
		if got := CalculateEuclideanDistance(tc.a, tc.b); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("%s: CalculateEuclideanDistance(%v, %v) = %v, want %v", tc.name, tc.a, tc.b, got, tc.want)
		}
	}

	if got := CalculateEuclideanDistance([]int{0, 0}, []int{3, 4}); math.Abs(got-5) > 1e-9 {
		t.Errorf("CalculateEuclideanDistance(int) = %v, want 5", got)
	}
}

func TestCalculateEuclideanDistanceDimensionMismatch(t *testing.T) {
	t.Parallel()

	got := CalculateEuclideanDistance([]float64{1, 2}, []float64{1})
	if !math.IsInf(got, 1) {
		t.Errorf("维度不一致时 = %v, want +Inf", got)
	}
}

func BenchmarkCalculateEuclideanDistance(b *testing.B) {
	a := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	target := []float64{8, 7, 6, 5, 4, 3, 2, 1}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CalculateEuclideanDistance(a, target)
	}
}
