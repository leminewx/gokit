package dict

import "iter"

var _ Dict[int, int] = &UnsafeDict[int, int]{}

// UnsafeDict 定义非并发安全的字典类型
type UnsafeDict[K comparable, V any] map[K]V

// NewUnsafeDict 新建一个非并发安全的字典
func NewUnsafeDict[K comparable, V any]() *UnsafeDict[K, V] {
	dict := UnsafeDict[K, V]{}
	return &dict
}

// Set 设置键值对
func (own *UnsafeDict[K, V]) Set(key K, val V) {
	own.ensure()
	(*own)[key] = val
}

// Update 更新字典
func (own *UnsafeDict[K, V]) Update(items map[K]V) {
	if len(items) == 0 {
		return
	}

	own.ensure()
	for key, val := range items {
		(*own)[key] = val
	}
}

// ensure 保证底层 map 已初始化，以支持零值直接使用
func (own *UnsafeDict[K, V]) ensure() {
	if *own == nil {
		*own = make(map[K]V)
	}
}

// Delete 删除键值对
func (own *UnsafeDict[K, V]) Delete(keys ...K) {
	for _, key := range keys {
		delete(*own, key)
	}
}

// HasAll 判断所有指定键值对是否存在
func (own *UnsafeDict[K, V]) HasAll(keys ...K) bool {
	for _, key := range keys {
		if _, ok := (*own)[key]; !ok {
			return false
		}
	}

	return true
}

// HasAny 判断任一指定元素是存在
func (own *UnsafeDict[K, V]) HasAny(keys ...K) bool {
	for _, key := range keys {
		if _, ok := (*own)[key]; ok {
			return true
		}
	}

	return false
}

// Get 查询键值对
func (own *UnsafeDict[K, V]) Get(key K) (V, bool) {
	val, ok := (*own)[key]
	return val, ok
}

// GetLength 获取字典长度
func (own *UnsafeDict[K, V]) GetLength() int {
	return len(*own)
}

// GetKeys 获取字典中所有的键
func (own *UnsafeDict[K, V]) GetKeys() []K {
	keys := make([]K, 0, len(*own))
	for key := range *own {
		keys = append(keys, key)
	}

	return keys
}

// GetValues 获取字典中所有的值
func (own *UnsafeDict[K, V]) GetValues() []V {
	values := make([]V, 0, len(*own))
	for _, val := range *own {
		values = append(values, val)
	}

	return values
}

// GetMap 获取Map
func (own *UnsafeDict[K, V]) GetMap() map[K]V {
	items := make(map[K]V)
	for key, val := range *own {
		items[key] = val
	}

	return items
}

// Range 遍历字典
func (own *UnsafeDict[K, V]) Range(fn func(key K, val V)) {
	for key, val := range *own {
		fn(key, val)
	}
}

// GetItems 返回遍历字典的迭代器，可在 for-range 中直接使用
func (own *UnsafeDict[K, V]) GetItems() iter.Seq2[K, V] {
	return func(yield func(key K, val V) bool) {
		for key, val := range *own {
			if !yield(key, val) {
				return
			}
		}
	}
}

// Filter 删除并返回符合过滤要求的键值对
func (own *UnsafeDict[K, V]) Filter(fn func(key K, val V) bool) map[K]V {
	items := make(map[K]V)
	for key, val := range *own {
		if fn(key, val) {
			items[key] = val
		}
	}

	for key := range items {
		delete(*own, key)
	}

	return items
}

// Transform 遍历并更新每个元素
func (own *UnsafeDict[K, V]) Transform(fn func(key K, val V) V) {
	for key, val := range *own {
		(*own)[key] = fn(key, val)
	}
}

// Clear 清空字典
func (own *UnsafeDict[K, V]) Clear() {
	clear(*own)
}
