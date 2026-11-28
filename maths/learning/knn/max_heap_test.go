package knn

import (
	"container/heap"
	"testing"
)

func TestMaxHeapPopOrder(t *testing.T) {
	t.Parallel()

	hp := &maxHeap{items: []neighbor{}}
	heap.Init(hp)

	for _, distance := range []float64{0.5, 0.1, 0.9, 0.3} {
		heap.Push(hp, neighbor{distance: distance})
	}

	last := 1.0
	for hp.Len() > 0 {
		item := heap.Pop(hp).(neighbor)
		if item.distance > last {
			t.Fatalf("弹出顺序错误：%v 在 %v 之后", item.distance, last)
		}
		last = item.distance
	}
}

func TestMaxHeapKeepsNearest(t *testing.T) {
	t.Parallel()

	k := 3
	hp := &maxHeap{k: k, items: []neighbor{}}
	heap.Init(hp)

	distances := []float64{5, 1, 4, 2, 3}
	for i, distance := range distances {
		heap.Push(hp, neighbor{index: i, distance: distance})
		if hp.Len() > k {
			heap.Pop(hp) // 弹出最大距离，保留最近的 k 个
		}
	}

	if hp.Len() != k {
		t.Fatalf("堆大小 = %d, want %d", hp.Len(), k)
	}

	// 堆顶必须是已保留元素中的最大距离
	if got := hp.Peek().distance; got != 3 {
		t.Errorf("Peek() = %v, want 3", got)
	}
}
