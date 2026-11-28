package queue

import (
	"container/heap"
	"testing"
)

func TestTaskHeapOrdersByExecuteAt(t *testing.T) {
	t.Parallel()

	tasks := &taskHeap{}
	heap.Init(tasks)

	for _, executeAt := range []int64{50, 10, 30, 20, 40} {
		heap.Push(tasks, &task{executeAt: executeAt})
	}

	last := int64(-1)
	for tasks.Len() > 0 {
		task := heap.Pop(tasks).(*task)
		if task.executeAt < last {
			t.Fatalf("弹出顺序错误：%d 在 %d 之后", task.executeAt, last)
		}
		last = task.executeAt

		if task.index != -1 {
			t.Errorf("弹出后 index = %d, want -1", task.index)
		}
	}
}

func TestTaskHeapIndexMaintained(t *testing.T) {
	t.Parallel()

	tasks := &taskHeap{}
	heap.Init(tasks)

	first := &task{executeAt: 100}
	second := &task{executeAt: 1}
	heap.Push(tasks, first)
	heap.Push(tasks, second)

	if first.index != 1 {
		t.Errorf("first.index = %d, want 1", first.index)
	}
	if second.index != 0 {
		t.Errorf("second.index = %d, want 0", second.index)
	}

	// 调整堆后索引必须同步更新
	first.executeAt = 0
	heap.Fix(tasks, first.index)
	if (*tasks)[0] != first {
		t.Error("调整优先级后堆顶不是 first")
	}
	if first.index != 0 {
		t.Errorf("Fix 后 first.index = %d, want 0", first.index)
	}
}

func TestTaskCancel(t *testing.T) {
	t.Parallel()

	task := &task{}
	if task.canceled.Load() {
		t.Error("新建任务默认已被取消")
	}

	task.canceled.Store(true)
	if !task.canceled.Load() {
		t.Error("取消标记未生效")
	}
}
