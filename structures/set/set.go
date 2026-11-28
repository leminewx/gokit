package set

import "iter"

var empty = struct{}{}

// Set 定义集合的接口类型
type Set[T comparable] interface {
	// Add 添加元素到集合
	Add(...T)
	// Delete 从集合中删除元素
	Delete(...T)
	// HasAll 判断所有指定元素是否存在于集合
	HasAll(...T) bool
	// HasAny 判断任一指定元素是否存在于集合
	HasAny(...T) bool
	// GetMap 获取集合所有元素的映射副本
	GetMap() map[T]struct{}
	// GetSlice 获取集合所有元素的切片副本
	GetSlice() []T
	// GetLength 获取集合长度
	GetLength() int

	// IsEmpty 判断集合是否为空
	IsEmpty() bool
	// IsEqualTo 判断两集合的元素是否完全相同
	IsEqualTo(Set[T]) bool
	// IsSubsetOf 判断当前集合是否为指定集合的子集
	IsSubsetOf(Set[T]) bool
	// IsSupersetOf 判断当前集合是否为指定集合的超集
	IsSupersetOf(Set[T]) bool

	// Union 求并集（不修改原集合，生成一个新集合）
	Union(Set[T]) Set[T]
	// Intersect 求交集（不修改原集合，生成一个新集合）
	Intersect(Set[T]) Set[T]
	// Complement 求差集（当前集合 - 另一集合，不修改原集合，生成一个新集合）
	Complement(Set[T]) Set[T]

	// Extract 提取符合条件的集合元素（不删除源数据）
	Extract(func(elem T) bool) []T
	// Filter 过滤集合（删除源数据）
	Filter(func(elem T) bool) []T
	// Range 遍历每个元素
	Range(func(elem T))
	// RangeUntil 遍历所有元素，当 fn 返回 false 时停止
	RangeUntil(func(elem T) bool)
	// GetAll 返回遍历集合的迭代器，可在 for-range 中直接使用
	GetAll() iter.Seq[T]
	// Clone 克隆当前集合
	Clone() Set[T]
	// Clear 清空集合
	Clear()
}

// ConvertToUnsafeSet 将指定集合转换为 UnsafeSet
func ConvertToUnsafeSet[T comparable](set Set[T]) Set[T] {
	return NewUnsafeSet(set.GetSlice()...)
}

// ConvertToSafeSet 将指定集合转换为 SafeSet
func ConvertToSafeSet[T comparable](set Set[T]) Set[T] {
	return NewSafeSet(set.GetSlice()...)
}

// ConvertToSyncSet 将指定集合转换为 SyncSet
func ConvertToSyncSet[T comparable](set Set[T]) Set[T] {
	return NewSyncSet(set.GetSlice()...)
}
