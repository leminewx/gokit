package set

import "iter"

// CollectSet 根据迭代器构建一个非并发安全的集合
//   - 常与 GetAll 配合使用，例如：set.CollectSet(s.GetAll())
func CollectSet[T comparable](seq iter.Seq[T]) Set[T] {
	elemSet := NewUnsafeSet[T]()
	for elem := range seq {
		elemSet.Add(elem)
	}

	return elemSet
}

// Union 求多个集合的并集（不修改原集合，生成一个新集合）
func Union[T comparable](sets ...Set[T]) Set[T] {
	length := 0
	for _, elemSet := range sets {
		length += elemSet.GetLength()
	}

	elems := make([]T, 0, length)
	for _, elemSet := range sets {
		elems = append(elems, elemSet.GetSlice()...)
	}

	return NewUnsafeSet(elems...)
}

// Intersect 求多个集合的交集（不修改原集合，生成一个新集合）
//   - 以第一个集合为基准，任意集合为空时结果必然为空集
func Intersect[T comparable](first Set[T], rest ...Set[T]) Set[T] {
	result := first.Clone()
	for _, elemSet := range rest {
		result = result.Intersect(elemSet)
		if result.IsEmpty() {
			break
		}
	}

	return result
}
