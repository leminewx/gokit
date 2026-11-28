package maths

import (
	"math"
	"testing"
)

// _EPSILON 为浮点断言容差，覆盖 float32 输入的累积误差。
const _EPSILON = 1e-6

// almostEqual 判断两个浮点数是否在容差内相等。
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= _EPSILON
}

func TestCalculateMean(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		values []float64
		want   float64
	}{
		{"空切片", nil, 0},
		{"单元素", []float64{7}, 7},
		{"正负数混合", []float64{1, -4, 9, 2}, 2},
		{"小数", []float64{1.21, 1.16, 1.42}, 1.2633333333},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if got := CalculateMean(c.values...); !almostEqual(got, c.want) {
				t.Errorf("CalculateMean(%v) = %v, want %v", c.values, got, c.want)
			}
		})
	}
}

func TestCalculateVariance(t *testing.T) {
	t.Parallel()

	values := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	if got, want := CalculateVariance(values), 4.0; !almostEqual(got, want) {
		t.Errorf("CalculateVariance(%v) = %v, want %v（总体方差）", values, got, want)
	}

	// 给定均值时应跳过内部均值计算，结果与自动计算一致
	if got, want := CalculateVariance(values, CalculateMean(values...)), 4.0; !almostEqual(got, want) {
		t.Errorf("CalculateVariance(给定均值) = %v, want %v", got, want)
	}

	if got := CalculateVariance([]int{}); got != 0 {
		t.Errorf("CalculateVariance(空切片) = %v, want 0", got)
	}
}

func TestCalculateStandardDeviation(t *testing.T) {
	t.Parallel()

	if got, want := CalculateStandardDeviation([]float64{2, 4, 4, 4, 5, 5, 7, 9}), 2.0; !almostEqual(got, want) {
		t.Errorf("CalculateStandardDeviation = %v, want %v", got, want)
	}

	// 常数列标准差为 0
	if got := CalculateStandardDeviation([]int32{3, 3, 3}); got != 0 {
		t.Errorf("CalculateStandardDeviation(常数列) = %v, want 0", got)
	}

	if got := CalculateStandardDeviation([]float64{}); got != 0 {
		t.Errorf("CalculateStandardDeviation(空切片) = %v, want 0", got)
	}
}

func TestCalculateMedian(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		values   []float64
		needSort bool
		want     float64
	}{
		{"奇数个已排序", []float64{1, 3, 5}, false, 3},
		{"偶数个已排序", []float64{1, 3, 5, 7}, false, 4},
		{"偶数个求和需提升精度", []float64{math.MaxInt32, math.MaxInt32 + 2}, false, math.MaxInt32 + 1},
		{"乱序且要求排序", []float64{5, 1, 3}, true, 3},
		{"空切片", nil, true, 0},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			values := append([]float64(nil), c.values...)
			if got := CalculateMedian(values, c.needSort); !almostEqual(got, c.want) {
				t.Errorf("CalculateMedian(%v, %v) = %v, want %v", c.values, c.needSort, got, c.want)
			}
		})
	}
}

func TestCalculateAbs(t *testing.T) {
	t.Parallel()

	if got := CalculateAbs(-3.5); got != 3.5 {
		t.Errorf("CalculateAbs(-3.5) = %v, want 3.5", got)
	}
	if got := CalculateAbs(int32(-9)); got != 9 {
		t.Errorf("CalculateAbs(int32(-9)) = %v, want 9", got)
	}
	if got := CalculateAbs(7); got != 7 {
		t.Errorf("CalculateAbs(7) = %v, want 7", got)
	}
}

func TestSum(t *testing.T) {
	t.Parallel()

	if got, want := Sum[int32](1, -4, 9, 2), int32(8); got != want {
		t.Errorf("Sum = %v, want %v", got, want)
	}
	if got := Sum[float64](); got != 0 {
		t.Errorf("Sum(空) = %v, want 0", got)
	}
	if got, want := Sum(1.5, 2.5), 4.0; got != want {
		t.Errorf("Sum(float64) = %v, want %v", got, want)
	}
}

func TestCount(t *testing.T) {
	t.Parallel()

	got := Count(1, -4, 9, 1, 9, 1)
	want := map[int]int{1: 3, -4: 1, 9: 2}
	if len(got) != len(want) {
		t.Fatalf("Count 结果长度 = %d, want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("Count[%v] = %d, want %d", k, got[k], v)
		}
	}

	if got := Count[int](); len(got) != 0 {
		t.Errorf("Count(空) = %v, want 空 map", got)
	}
}

func TestCalculateRange(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		values  []int32
		wantMin int32
		wantMax int32
	}{
		{"正负数混合", []int32{1, -4, 9, 2, 11, -5}, -5, 11},
		{"单元素", []int32{7}, 7, 7},
		{"空切片", nil, 0, 0},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			minValue, maxValue := CalculateRange(c.values...)
			if minValue != c.wantMin || maxValue != c.wantMax {
				t.Errorf("CalculateRange(%v) = (%v, %v), want (%v, %v)",
					c.values, minValue, maxValue, c.wantMin, c.wantMax)
			}
		})
	}
}

func BenchmarkCount(b *testing.B) {
	values := []int32{
		1, -4, 9, 2, 11, -5, 3, 7, -1, 6, 1, -4, 9, 2, 11, -5, 3, 7, -1, 6, 1, -4, 9, 2, 11, -5, 3, 7, -1, 6,
		1, -4, 9, 2, 11, -5, 3, 7, -1, 6, 1, -4, 9, 2, 11, -5, 3, 7, -1, 6, 1, -4, 9, 2, 11, -5, 3, 7, -1, 6, 0, 9,
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Count(values...)
	}
}

func BenchmarkCalculateRange(b *testing.B) {
	values := []int32{
		1, -4, 9, 2, 11, -5, 3, 7, -1, 6, 1, -4, 9, 2, 11, -5, 3, 7, -1, 6, 1, -4, 9, 2, 11, -5, 3, 7, -1, 6,
		1, -4, 9, 2, 11, -5, 3, 7, -1, 6, 1, -4, 9, 2, 11, -5, 3, 7, -1, 6, 1, -4, 9, 2, 11, -5, 3, 7, -1, 6, 0, 9,
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = CalculateRange(values...)
	}
}
