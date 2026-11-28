package set

import (
	"iter"
	"sync"
)

var _ Set[int] = (*SyncSet[int])(nil)

// SyncSet 定义基于 sync.Map 的并发安全的集合（组合 sync.Map，适用于读多写少）
type SyncSet[T comparable] struct {
	dict sync.Map
}

// NewSyncSet 新建一个基于 sync.Map 的并发安全的集合（适用于读多写少）
func NewSyncSet[T comparable](elems ...T) Set[T] {
	own := &SyncSet[T]{}
	own.Add(elems...)
	return own
}

// Add 添加元素到集合
func (own *SyncSet[T]) Add(elems ...T) {
	for _, elem := range elems {
		own.dict.Store(elem, empty)
	}
}

// Delete 从集合中删除元素
func (own *SyncSet[T]) Delete(elems ...T) {
	for _, elem := range elems {
		own.dict.Delete(elem)
	}
}

// HasAll 判断所有指定元素是否存在于集合
func (own *SyncSet[T]) HasAll(elems ...T) bool {
	for _, elem := range elems {
		if _, ok := own.dict.Load(elem); !ok {
			return false
		}
	}

	return true
}

// HasAny 判断任一指定元素是存在于集合
func (own *SyncSet[T]) HasAny(elems ...T) bool {
	for _, elem := range elems {
		if _, ok := own.dict.Load(elem); ok {
			return true
		}
	}

	return false
}

// GetMap 获取集合的字典格式
func (own *SyncSet[T]) GetMap() map[T]struct{} {
	elems := make(map[T]struct{})
	own.dict.Range(func(key, _ any) bool {
		elems[key.(T)] = empty
		return true
	})

	return elems
}

// GetSlice 获取集合的所有元素
func (own *SyncSet[T]) GetSlice() []T {
	elems := make([]T, 0)
	own.dict.Range(func(key, _ any) bool {
		elems = append(elems, key.(T))
		return true
	})

	return elems
}

// GetLength 获取集合长度
func (own *SyncSet[T]) GetLength() int {
	var length int
	own.dict.Range(func(_, _ any) bool {
		length++
		return true
	})

	return length
}

// IsEmpty 判断集合是否为空
func (own *SyncSet[T]) IsEmpty() bool {
	var isEmpty = true
	own.dict.Range(func(_, _ any) bool {
		isEmpty = false
		return false // 发现一个元素立即停止
	})

	return isEmpty
}

// IsEqualTo 判断两集合的元素是否完全相同
func (own *SyncSet[T]) IsEqualTo(other Set[T]) bool {
	if own.GetLength() != other.GetLength() {
		return false
	}

	// 长度相等时，检查当前集合是否包含另一集合的所有元素（或互为子集）
	return own.IsSubsetOf(other)
}

// IsSubsetOf 判断当前集合是否是指定集合的子集
func (own *SyncSet[T]) IsSubsetOf(other Set[T]) bool {
	// 空集是任何集合的子集
	if own.IsEmpty() {
		return true
	}
	if own.GetLength() > other.GetLength() {
		return false
	}

	foundAll := true
	own.dict.Range(func(key, _ any) bool {
		if !other.HasAny(key.(T)) {
			foundAll = false
			return false // 停止遍历
		}

		return true
	})

	return foundAll
}

// IsSupersetOf 判断当前集合是否为指定集合的超集
func (own *SyncSet[T]) IsSupersetOf(other Set[T]) bool {
	if own.GetLength() < other.GetLength() {
		return false
	}

	foundAll := true
	other.RangeUntil(func(elem T) bool {
		if _, ok := own.dict.Load(elem); !ok {
			foundAll = false
			return false
		}

		return true
	})

	return foundAll
}

// Union 求并集（不修改原集合，生成一个新集合）
func (own *SyncSet[T]) Union(other Set[T]) Set[T] {
	elems := own.GetSlice()
	return NewSyncSet(append(elems, other.GetSlice()...)...)
}

// Intersect 求交集（不修改原集合，生成一个新集合）
func (own *SyncSet[T]) Intersect(other Set[T]) Set[T] {
	ownSet := NewSyncSet(own.GetSlice()...)

	// 遍历较小的集合，减少检查次数
	if own.GetLength() < other.GetLength() {
		ownSet.Clear()
		own.dict.Range(func(key, _ any) bool {
			elem := key.(T)
			if other.HasAny(elem) {
				ownSet.Add(elem)
			}

			return true
		})

		return ownSet
	}

	ownSet.Clear()
	other.Range(func(elem T) {
		if _, ok := own.dict.Load(elem); ok {
			ownSet.Add(elem)
		}
	})

	return ownSet
}

// Complement 求差集（当前集合 - other 集合，不修改原集合，生成一个新集合）
func (own *SyncSet[T]) Complement(other Set[T]) Set[T] {
	elems := make([]T, 0)
	own.dict.Range(func(key, _ any) bool {
		elem := key.(T)
		if !other.HasAny(elem) {
			elems = append(elems, elem)
		}

		return true
	})

	return NewSyncSet(elems...)
}

// Extract 提取符合条件的集合元素
func (own *SyncSet[T]) Extract(fn func(elem T) bool) []T {
	elems := make([]T, 0)
	own.dict.Range(func(key, _ any) bool {
		elem := key.(T)
		if fn(elem) {
			elems = append(elems, elem)
		}

		return true
	})

	return elems
}

// Filter 过滤集合（删除源数据）
func (own *SyncSet[T]) Filter(fn func(elem T) bool) []T {
	// 符合条件的元素（将被删除并返回）
	removed := make([]T, 0)
	own.dict.Range(func(key, _ any) bool {
		elem := key.(T)
		if fn(elem) {
			removed = append(removed, elem)
		}

		return true
	})

	// 删除符合条件的元素，保留不符合条件的
	own.Delete(removed...)
	return removed
}

// Range 遍历并获取每个元素
func (own *SyncSet[T]) Range(fn func(elem T)) {
	own.dict.Range(func(key, _ any) bool {
		fn(key.(T))
		return true
	})
}

// RangeUntil 遍历所有元素，当 fn 返回 false 时停止
func (own *SyncSet[T]) RangeUntil(fn func(elem T) bool) {
	own.dict.Range(func(key, _ any) bool {
		return fn(key.(T))
	})
}

// GetAll 返回遍历集合的迭代器，可在 for-range 中直接使用
//   - sync.Map 的 Range 不持有全局锁，遍历结果弱一致
func (own *SyncSet[T]) GetAll() iter.Seq[T] {
	return func(yield func(elem T) bool) {
		own.dict.Range(func(key, _ any) bool {
			return yield(key.(T))
		})
	}
}

// Clone 克隆当前集合
func (own *SyncSet[T]) Clone() Set[T] {
	return NewSyncSet(own.GetSlice()...)
}

// Clear 清空集合
func (own *SyncSet[T]) Clear() {
	own.dict.Clear()
}
