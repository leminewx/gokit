package queue

import (
	"container/heap"
	"sync/atomic"
	"time"
)

type (
	// task 定义定时任务的数据结构（内部类型，由 AddTask 创建）
	task struct {
		id        int64         // 任务唯一ID
		job       func()        // 任务函数
		executeAt int64         // 定时任务执行时间戳(纳秒)
		canceled  atomic.Bool   // 用于标记任务已被取消(原子操作)
		period    time.Duration // 重复执行间隔(0表示不重复)
		remaining int           // 剩余执行次数(-1表示一直重复；>0表示指定次数)
		index     int           // 在堆中的索引
	}

	// taskHeap 定义任务最小堆的数据结构
	taskHeap []*task
)

var _ heap.Interface = (*taskHeap)(nil)

// Len 实现 heap.Interface 接口
func (own *taskHeap) Len() int {
	return len(*own)
}

// Less 实现 heap.Interface 接口
func (own *taskHeap) Less(i, j int) bool {
	return (*own)[i].executeAt < (*own)[j].executeAt
}

// Swap 实现 heap.Interface 接口
func (own *taskHeap) Swap(i, j int) {
	h := *own
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

// Push 压入数据
func (own *taskHeap) Push(x any) {
	n := len(*own)
	task := x.(*task)
	task.index = n
	*own = append(*own, task)
}

// Pop 弹出数据
func (own *taskHeap) Pop() any {
	old := *own
	n := len(old)
	task := old[n-1]
	old[n-1] = nil // 避免内存泄漏
	task.index = -1
	*own = old[0 : n-1]
	return task
}
