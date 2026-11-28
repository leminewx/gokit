package set

import "iter"

var _ Set[int] = UnsafeSet[int]{}

// UnsafeSet 定义非并发安全的集合类型
type UnsafeSet[T comparable] map[T]struct{}

// NewUnsafeSet 新建一个非并发安全的集合
func NewUnsafeSet[T comparable](elems ...T) Set[T] {
	set := make(UnsafeSet[T])
	set.Add(elems...)
	return &set
}

// Add 添加元素到集合
func (own UnsafeSet[T]) Add(elems ...T) {
	for _, elem := range elems {
		own[elem] = empty
	}
}

// Delete 从集合中删除元素
func (own UnsafeSet[T]) Delete(elems ...T) {
	for _, elem := range elems {
		delete(own, elem)
	}
}

// HasAll 判断所有指定元素是否存在于集合
func (own UnsafeSet[T]) HasAll(elems ...T) bool {
	for _, elem := range elems {
		if _, ok := own[elem]; !ok {
			return false
		}
	}

	return true
}

// HasAny 判断任一指定元素是存在于集合
func (own UnsafeSet[T]) HasAny(elems ...T) bool {
	for _, elem := range elems {
		if _, ok := own[elem]; ok {
			return true
		}
	}

	return false
}

// GetMap 获取集合所有元素的映射
func (own UnsafeSet[T]) GetMap() map[T]struct{} {
	elems := make(map[T]struct{}, len(own))
	for elem := range own {
		elems[elem] = empty
	}

	return elems
}

// GetSlice 获取集合所有元素的切片
func (own UnsafeSet[T]) GetSlice() []T {
	elems := make([]T, 0, len(own))
	for elem := range own {
		elems = append(elems, elem)
	}

	return elems
}

// GetLength 获取集合长度
func (own UnsafeSet[T]) GetLength() int {
	return len(own)
}

// IsEmpty 判断集合是否为空
func (own UnsafeSet[T]) IsEmpty() bool {
	return len(own) == 0
}

// IsEqualTo 判断两集合的元素是否完全相同
func (own UnsafeSet[T]) IsEqualTo(other Set[T]) bool {
	if own.GetLength() != other.GetLength() {
		return false
	}

	// 长度相等时，子集关系等价于相等
	return own.IsSubsetOf(other)
}

// IsSubsetOf 判断当前集合是否是指定集合的子集
func (own UnsafeSet[T]) IsSubsetOf(other Set[T]) bool {
	if own.GetLength() > other.GetLength() {
		return false
	}

	// 检查单个元素是否在另一集合中
	for elem := range own {
		if !other.HasAny(elem) {
			return false
		}
	}

	return true
}

// IsSupersetOf 判断当前集合是否为指定集合的超集
func (own UnsafeSet[T]) IsSupersetOf(other Set[T]) bool {
	if own.GetLength() < other.GetLength() {
		return false
	}

	foundAll := true
	other.RangeUntil(func(elem T) bool {
		if _, ok := own[elem]; !ok {
			foundAll = false
			return false
		}

		return true
	})

	return foundAll
}

// Union 求并集（不修改原集合，生成一个新集合）
func (own UnsafeSet[T]) Union(other Set[T]) Set[T] {
	// 克隆当前集合，再添加另一集合的所有元素
	set := own.Clone()
	set.Add(other.GetSlice()...)
	return set
}

// Intersect 求交集（不修改原集合，生成一个新集合）
func (own UnsafeSet[T]) Intersect(other Set[T]) Set[T] {
	// 遍历较小的集合，减少检查次数
	if own.GetLength() < other.GetLength() {
		// 交换角色，确保遍历较小的
		return other.Intersect(own)
	}

	set := NewUnsafeSet[T]()
	for _, elem := range other.GetSlice() {
		if _, ok := own[elem]; ok {
			set.Add(elem)
		}
	}

	return set
}

// Complement 求差集（当前集合 - other 集合，不修改原集合，生成一个新集合）
func (own UnsafeSet[T]) Complement(other Set[T]) Set[T] {
	set := NewUnsafeSet[T]()
	for elem := range own {
		if !other.HasAny(elem) {
			set.Add(elem)
		}
	}

	return set
}

// Extract 提取符合条件的集合元素
func (own UnsafeSet[T]) Extract(fn func(elem T) bool) []T {
	elems := make([]T, 0, len(own))
	for elem := range own {
		if fn(elem) {
			elems = append(elems, elem)
		}
	}

	return elems
}

// Filter 过滤集合，删除符合条件的元素并返回被删除的元素
func (own UnsafeSet[T]) Filter(fn func(elem T) bool) []T {
	// 符合条件的元素（将被删除并返回）
	removed := make([]T, 0)
	for elem := range own {
		if fn(elem) {
			removed = append(removed, elem)
		}
	}

	// 删除符合条件的元素，保留不符合条件的
	own.Delete(removed...)
	return removed
}

// Range 遍历所有元素
func (own UnsafeSet[T]) Range(fn func(elem T)) {
	for elem := range own {
		fn(elem)
	}
}

// RangeUntil 遍历所有元素，当 fn 返回 false 时停止
func (own UnsafeSet[T]) RangeUntil(fn func(elem T) bool) {
	for elem := range own {
		if !fn(elem) {
			break
		}
	}
}

// GetAll 返回遍历集合的迭代器，可在 for-range 中直接使用
func (own UnsafeSet[T]) GetAll() iter.Seq[T] {
	return func(yield func(elem T) bool) {
		for elem := range own {
			if !yield(elem) {
				return
			}
		}
	}
}

// clone 返回集合的浅拷贝（供组合类型复用）
func (own UnsafeSet[T]) clone() UnsafeSet[T] {
	elems := make(UnsafeSet[T], len(own))
	for elem := range own {
		elems[elem] = empty
	}

	return elems
}

// Clone 克隆当前集合
func (own UnsafeSet[T]) Clone() Set[T] {
	return own.clone()
}

// Clear 清空集合
func (own UnsafeSet[T]) Clear() {
	clear(own)
}
