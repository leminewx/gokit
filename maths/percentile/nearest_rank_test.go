package percentile

import (
	"errors"
	"testing"
)

// TestNearestRankPercentile 最近秩口径：第 p 百分位取 ceil(n*p/100)-1 的样本值。
func TestNearestRankPercentile(t *testing.T) {
	t.Parallel()

	calculator := NewNearestRank()
	for value := 1.0; value <= 10; value++ {
		calculator.Add(value)
	}

	cases := []struct {
		p    float64
		want float64
	}{
		{0, 1},
		{10, 1},   // ceil(1)-1 = 0
		{50, 5},   // ceil(5)-1 = 4
		{90, 9},   // ceil(9)-1 = 8
		{99, 10},  // ceil(9.9)-1 = 9
		{100, 10}, // ceil(10)-1 = 9
	}
	for _, item := range cases {
		got, err := calculator.CalculatePercentile(item.p)
		if err != nil {
			t.Errorf("P%v 返回错误: %v", item.p, err)
			continue
		}
		if got != item.want {
			t.Errorf("P%v = %v, want %v", item.p, got, item.want)
		}
	}
}

// TestNearestRankUnsortedInput 乱序写入后仍能给出正确分位（内部懒排序）。
func TestNearestRankUnsortedInput(t *testing.T) {
	t.Parallel()

	calculator := NewNearestRank()
	for _, value := range []float64{4, 1, 3, 2} {
		calculator.Add(value)
	}

	// 4 个样本的 P50 取第 2 个（下标 1），与 R-7 插值（2.5）不同
	got, err := calculator.CalculatePercentile(50)
	if err != nil {
		t.Fatalf("P50 返回错误: %v", err)
	}
	if got != 2 {
		t.Errorf("P50 = %v, want 2", got)
	}

	if count := calculator.GetCount(); count != 4 {
		t.Errorf("Count = %d, want 4", count)
	}
}

// TestNearestRankErrors 空样本与越界百分位必须返回可判定错误。
func TestNearestRankErrors(t *testing.T) {
	t.Parallel()

	calculator := NewNearestRank()
	if _, err := calculator.CalculatePercentile(50); !errors.Is(err, ErrNoDataPoints) {
		t.Errorf("空样本错误 = %v, want ErrNoDataPoints", err)
	}

	calculator.Add(1)
	if _, err := calculator.CalculatePercentile(-1); !errors.Is(err, ErrPercentileOutOfRange) {
		t.Errorf("P-1 错误 = %v, want ErrPercentileOutOfRange", err)
	}
	if _, err := calculator.CalculatePercentile(101); !errors.Is(err, ErrPercentileOutOfRange) {
		t.Errorf("P101 错误 = %v, want ErrPercentileOutOfRange", err)
	}
}

// TestNearestRankMerge 同类型可合并且保留全部样本；不同类型拒绝合并。
func TestNearestRankMerge(t *testing.T) {
	t.Parallel()

	left := NewNearestRank()
	right := NewNearestRank()
	left.Add(10)
	right.Add(20)
	right.Add(30)

	if err := left.Merge(right); err != nil {
		t.Fatalf("Merge 失败: %v", err)
	}
	if count := left.GetCount(); count != 3 {
		t.Errorf("合并后 Count = %d, want 3", count)
	}

	got, err := left.CalculatePercentile(100)
	if err != nil {
		t.Fatalf("P100 返回错误: %v", err)
	}
	if got != 30 {
		t.Errorf("P100 = %v, want 30", got)
	}

	if err := left.Merge(NewExact()); !errors.Is(err, ErrNotMergeable) {
		t.Errorf("跨实现合并错误 = %v, want ErrNotMergeable", err)
	}
}

// TestNearestRankConcurrentAdd 并发写入不丢样本（配合 -race 检查数据竞争）。
func TestNearestRankConcurrentAdd(t *testing.T) {
	t.Parallel()

	const workers, perWorker = 8, 100
	calculator := NewNearestRank()

	done := make(chan struct{})
	for worker := 0; worker < workers; worker++ {
		go func(base int) {
			defer func() { done <- struct{}{} }()
			for offset := 0; offset < perWorker; offset++ {
				calculator.Add(float64(base + offset))
			}
		}(worker * perWorker)
	}
	for worker := 0; worker < workers; worker++ {
		<-done
	}

	if count := calculator.GetCount(); count != workers*perWorker {
		t.Errorf("Count = %d, want %d", count, workers*perWorker)
	}
}

// TestCalculateNearestRankOnSorted 已排序切片直接计算，免去实例开销。
func TestCalculateNearestRankOnSorted(t *testing.T) {
	t.Parallel()

	sorted := []float64{5, 10, 15, 20, 25}
	cases := []struct {
		p    float64
		want float64
	}{
		{0, 5},
		{20, 5},
		{40, 10},
		{100, 25},
	}
	for _, item := range cases {
		got, err := CalculateNearestRank(sorted, item.p)
		if err != nil {
			t.Errorf("P%v 返回错误: %v", item.p, err)
			continue
		}
		if got != item.want {
			t.Errorf("P%v = %v, want %v", item.p, got, item.want)
		}
	}

	if _, err := CalculateNearestRank(nil, 50); !errors.Is(err, ErrNoDataPoints) {
		t.Errorf("空切片错误 = %v, want ErrNoDataPoints", err)
	}
	if _, err := CalculateNearestRank(sorted, 100.1); !errors.Is(err, ErrPercentileOutOfRange) {
		t.Errorf("越界错误 = %v, want ErrPercentileOutOfRange", err)
	}
}
