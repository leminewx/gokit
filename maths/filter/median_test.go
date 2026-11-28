package filter

import (
	"errors"
	"testing"
)

func TestCalculateMedian(t *testing.T) {
	values := []float64{1, 10, 2, 3, 100, 4, 5}

	series, err := CalculateMedian(values, 3)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	want := []float64{1, 2, 3, 3, 4, 5, 5}
	for index := range want {
		requireClose(t, "中值", series.Values[index], want[index], 1e-12)
	}

	if series.Replaced != 0 {
		t.Fatalf("Replaced = %d, want 0", series.Replaced)
	}
}

func TestCalculateMedianPreservesEdge(t *testing.T) {
	values := []float64{9, 1, 1, 1, 1, 1, 9}

	series, err := CalculateMedian(values, 5)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if series.Values[0] != 9 || series.Values[len(values)-1] != 9 {
		t.Fatalf("边界值被改写：%v", series.Values)
	}

	for index := 2; index < len(values)-2; index++ {
		requireClose(t, "中值", series.Values[index], 1, 1e-12)
	}
}

func TestCalculateMedianErrors(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		window int
	}{
		{"偶数窗口", []float64{1, 2, 3, 4}, 4},
		{"窗口过小", []float64{1, 2, 3}, 1},
		{"窗口超过长度", []float64{1, 2, 3}, 5},
		{"窗口等于长度但不是奇数", []float64{1, 2, 3, 4}, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateMedian(tt.values, tt.window)
			if !errors.Is(err, ErrInvalidWindow) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidWindow)
			}
		})
	}
}

func TestCalculateMedianNoAliasing(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}

	series, err := CalculateMedian(values, 3)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	series.Values[0] = 100
	if values[0] != 1 {
		t.Fatalf("结果与输入共享底层数组")
	}
}
