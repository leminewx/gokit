package knn

import "container/heap"

// maxHeap 最大堆实现（用于保持最小的 k 个距离）
type maxHeap struct {
	k     int
	items []neighbor
}

// 编译期断言：Len/Less/Swap/Push/Pop 是 heap.Interface 钉死的方法名
var _ heap.Interface = (*maxHeap)(nil)

func (own maxHeap) Len() int {
	return len(own.items)
}

func (own maxHeap) Less(i, j int) bool {
	return own.items[i].distance > own.items[j].distance
}

func (own maxHeap) Swap(i, j int) {
	own.items[i], own.items[j] = own.items[j], own.items[i]
}

func (own *maxHeap) Push(x any) {
	own.items = append(own.items, x.(neighbor))
}

func (own *maxHeap) Pop() any {
	old := own.items
	n := len(old)
	x := old[n-1]
	own.items = old[0 : n-1]
	return x
}

func (own *maxHeap) Peek() neighbor {
	return own.items[0]
}
