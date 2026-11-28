package set

import (
	"iter"
	"sync"
)

var _ Set[int] = (*SafeSet[int])(nil)

// SafeSet 定义基于 sync.RWMutex 的并发安全的集合（组合非并发安全的集合，锁粒度最小）
type SafeSet[T comparable] struct {
	set  UnsafeSet[T]
	lock sync.RWMutex
}

// NewSafeSet 新建一个基于 sync.RWMutex 的并发安全的集合
func NewSafeSet[T comparable](elems ...T) Set[T] {
	set := make(UnsafeSet[T], len(elems))
	set.Add(elems...)

	return &SafeSet[T]{set: set}
}

// Add 添加元素到集合
func (own *SafeSet[T]) Add(elems ...T) {
	own.lock.Lock()
	own.set.Add(elems...)
	own.lock.Unlock()
}

// Delete 从集合中删除元素
func (own *SafeSet[T]) Delete(elems ...T) {
	own.lock.Lock()
	own.set.Delete(elems...)
	own.lock.Unlock()
}

// HasAll 判断所有指定元素是否存在于集合
func (own *SafeSet[T]) HasAll(elems ...T) bool {
	own.lock.RLock()
	ok := own.set.HasAll(elems...)
	own.lock.RUnlock()
	return ok
}

// HasAny 判断任一指定元素是存在于集合
func (own *SafeSet[T]) HasAny(elems ...T) bool {
	own.lock.RLock()
	ok := own.set.HasAny(elems...)
	own.lock.RUnlock()
	return ok
}

// GetMap 获取集合所有元素的映射
func (own *SafeSet[T]) GetMap() map[T]struct{} {
	own.lock.RLock()
	elems := own.set.GetMap()
	own.lock.RUnlock()
	return elems
}

// GetSlice 获取集合所有元素的切片
func (own *SafeSet[T]) GetSlice() []T {
	own.lock.RLock()
	elems := own.set.GetSlice()
	own.lock.RUnlock()
	return elems
}

// GetLength 获取集合长度
func (own *SafeSet[T]) GetLength() int {
	own.lock.RLock()
	length := own.set.GetLength()
	own.lock.RUnlock()
	return length
}

// IsEmpty 判断集合是否为空
func (own *SafeSet[T]) IsEmpty() bool {
	return own.GetLength() == 0
}

// IsEqualTo 判断两集合的元素是否完全相同
func (own *SafeSet[T]) IsEqualTo(other Set[T]) bool {
	own.lock.RLock()
	ok := own.set.IsEqualTo(other)
	own.lock.RUnlock()
	return ok
}

// IsSubsetOf 判断当前集合是否是指定集合的子集
func (own *SafeSet[T]) IsSubsetOf(other Set[T]) bool {
	own.lock.RLock()
	ok := own.set.IsSubsetOf(other)
	own.lock.RUnlock()
	return ok
}

// IsSupersetOf 判断当前集合是否为指定集合的超集
func (own *SafeSet[T]) IsSupersetOf(other Set[T]) bool {
	own.lock.RLock()
	ok := own.set.IsSupersetOf(other)
	own.lock.RUnlock()
	return ok
}

// Union 求并集（不修改原集合，生成一个新集合）
func (own *SafeSet[T]) Union(other Set[T]) Set[T] {
	own.lock.RLock()
	set := own.set.Union(other)
	own.lock.RUnlock()
	return set
}

// Intersect 求交集（不修改原集合，生成一个新集合）
func (own *SafeSet[T]) Intersect(other Set[T]) Set[T] {
	own.lock.RLock()
	set := own.set.Intersect(other)
	own.lock.RUnlock()
	return set
}

// Complement 求差集（当前集合 - other 集合，不修改原集合，生成一个新集合）
func (own *SafeSet[T]) Complement(other Set[T]) Set[T] {
	own.lock.RLock()
	set := own.set.Complement(other)
	own.lock.RUnlock()
	return set
}

// Extract 提取符合条件的集合元素
func (own *SafeSet[T]) Extract(fn func(elem T) bool) []T {
	own.lock.RLock()
	elems := own.set.Extract(fn)
	own.lock.RUnlock()
	return elems
}

// Filter 过滤集合，删除并返回过滤的元素
func (own *SafeSet[T]) Filter(fn func(elem T) bool) []T {
	own.lock.Lock()
	elems := own.set.Filter(fn)
	own.lock.Unlock()
	return elems
}

// Range 遍历并获取每个元素
func (own *SafeSet[T]) Range(fn func(elem T)) {
	own.lock.RLock()
	own.set.Range(fn)
	own.lock.RUnlock()
}

// RangeUntil 遍历所有元素，当 fn 返回 false 时停止
func (own *SafeSet[T]) RangeUntil(fn func(elem T) bool) {
	own.lock.RLock()
	own.set.RangeUntil(fn)
	own.lock.RUnlock()
}

// GetAll 返回遍历集合的迭代器，可在 for-range 中直接使用
//   - 遍历前先做快照，避免在 yield 期间持有读锁
func (own *SafeSet[T]) GetAll() iter.Seq[T] {
	return func(yield func(elem T) bool) {
		for _, elem := range own.GetSlice() {
			if !yield(elem) {
				return
			}
		}
	}
}

// Clone 克隆当前集合
func (own *SafeSet[T]) Clone() Set[T] {
	own.lock.RLock()
	set := own.set.clone()
	own.lock.RUnlock()

	return &SafeSet[T]{set: set}
}

func (own *SafeSet[T]) Clear() {
	own.lock.Lock()
	own.set.Clear()
	own.lock.Unlock()
}
