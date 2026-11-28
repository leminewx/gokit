package stream

import (
	"context"
	"testing"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

func TestBatch(t *testing.T) {
	s := NewStreamOf(1, 2, 3, 4, 5)
	batched := Batch(s, 2)
	got := collect(batched)
	want := [][]int{{1, 2}, {3, 4}, {5}}
	if len(got) != len(want) {
		t.Fatalf("Batch length = %d, want %d", len(got), len(want))
	}
	for i := 0; i < len(got); i++ {
		if !equalSlice(got[i], want[i]) {
			t.Errorf("Batch[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestBatchWithTimeout(t *testing.T) {
	s := NewStreamOf(1, 2, 3)
	batched := Batch(s, 5, 100*time.Millisecond)
	got := collect(batched)
	if len(got) != 1 || len(got[0]) != 3 {
		t.Errorf("Batch with timeout = %v, want [[1,2,3]]", got)
	}
}

// TestBatchWithClock 验证按数量与按超时两种批处理触发方式
func TestBatchWithClock(t *testing.T) {
	t.Parallel()

	const interval = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fake := timex.NewManual(testEpoch)
	source, s := testSource[int](ctx, fake)
	batched := Batch(s, 10, interval)

	batches := make(chan []int, 8)
	go func() {
		for {
			item, ok := batched.Read()
			if !ok {
				return
			}

			batches <- item.GetValue()
		}
	}()

	// 未达到数量阈值，由超时触发
	source <- NewItem(1)
	source <- NewItem(2)
	if batch := nextBatch(t, fake, interval, batches); !equalSlice(batch, []int{1, 2}) {
		t.Fatalf("超时批处理结果错误: %v", batch)
	}

	// 达到数量阈值后立即输出
	for i := 3; i <= 12; i++ {
		source <- NewItem(i)
	}
	want := []int{3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	if batch := nextBatch(t, fake, interval, batches); !equalSlice(batch, want) {
		t.Fatalf("数量批处理结果错误: %v", batch)
	}
}

// testEpoch 测试时钟的起点
var testEpoch = time.Unix(1700000000, 0).UTC()

// nextBatch 反复推进时钟直到产出下一批数据
func nextBatch(t *testing.T, fake *timex.Manual, interval time.Duration, batches <-chan []int) []int {
	t.Helper()

	deadline := time.After(2 * time.Second)
	for {
		fake.Advance(interval)

		select {
		case batch := <-batches:
			return batch
		case <-deadline:
			t.Fatal("等待批处理输出超时")
		case <-time.After(5 * time.Millisecond):
		}
	}
}
