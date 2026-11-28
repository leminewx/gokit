package stream

import "testing"

// TestTransformKeepsZeroValue 验证零值不会被 Transform 静默丢弃
func TestTransformKeepsZeroValue(t *testing.T) {
	t.Parallel()

	got := collect(NewStreamOf(1, 2, 3).Transform(func(item Item[int]) int {
		if item.GetValue() == 2 {
			return 0
		}

		return item.GetValue()
	}))

	if !equalSlice(got, []int{1, 0, 3}) {
		t.Fatalf("Transform 零值处理错误: %v", got)
	}
}

// TestTransformCallback 验证收尾回调在流结束时执行
func TestTransformCallback(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	stream := NewStreamOf(1, 2).Transform(func(item Item[int]) int {
		return item.GetValue()
	}, func() { close(done) })

	stream.Wait()

	select {
	case <-done:
	default:
		t.Fatal("Transform 收尾回调未执行")
	}
}

// TestMapZeroValueAndIndex 验证 Map 保留零值且索引从 0 连续递增
func TestMapZeroValueAndIndex(t *testing.T) {
	t.Parallel()

	mapped := Map(NewStreamOf(1, 2, 0), func(item Item[int]) int {
		return item.GetValue() * 10
	})

	var (
		values  []int
		indexes []uint64
	)
	mapped.Consume(func(item Item[int]) {
		values = append(values, item.GetValue())
		indexes = append(indexes, item.GetIndex())
	})

	if !equalSlice(values, []int{10, 20, 0}) {
		t.Fatalf("Map 结果错误: %v", values)
	}
	if !equalSlice(indexes, []uint64{0, 1, 2}) {
		t.Fatalf("Map 索引错误: %v", indexes)
	}
}
