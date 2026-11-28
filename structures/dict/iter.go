package dict

import "iter"

// ExtractKeys 从键值对迭代器中提取键迭代器
//   - 常与 GetItems 配合使用，例如：for key := range dict.ExtractKeys(d.GetItems())
func ExtractKeys[K comparable, V any](seq iter.Seq2[K, V]) iter.Seq[K] {
	return func(yield func(key K) bool) {
		for key := range seq {
			if !yield(key) {
				return
			}
		}
	}
}

// ExtractValues 从键值对迭代器中提取值迭代器
//   - 常与 GetItems 配合使用，例如：for val := range dict.ExtractValues(d.GetItems())
func ExtractValues[K comparable, V any](seq iter.Seq2[K, V]) iter.Seq[V] {
	return func(yield func(val V) bool) {
		for _, val := range seq {
			if !yield(val) {
				return
			}
		}
	}
}

// Collect 将键值对迭代器收集为非并发安全的字典
func Collect[K comparable, V any](seq iter.Seq2[K, V]) *UnsafeDict[K, V] {
	items := NewUnsafeDict[K, V]()
	for key, val := range seq {
		items.Set(key, val)
	}

	return items
}
