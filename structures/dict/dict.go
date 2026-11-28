package dict

import "iter"

// Dict 定义字典的接口类型
type Dict[K comparable, V any] interface {
	// Set 设置键值对
	Set(key K, val V)
	// Update 更新字典
	Update(map[K]V)
	// Delete 删除键值对
	Delete(...K)
	// HasAll 判断所有指定键值对是否存在
	HasAll(...K) bool
	// HasAny 判断任一指定元素是存在
	HasAny(...K) bool
	// Get 询查键值对
	Get(key K) (V, bool)
	// GetKeys 获取字典中所有的键
	GetKeys() []K
	// GetValues 获取字典中所有的值
	GetValues() []V
	// GetLength 获取字典长度
	GetLength() int
	// GetMap 获取Map
	GetMap() map[K]V

	// Transform 遍历并更新所有元素
	Transform(func(key K, val V) V)
	// Filter 遍历字典，删除并获取键值对
	Filter(func(key K, val V) bool) map[K]V
	// Range 遍历所有元素
	Range(func(key K, val V))
	// GetItems 返回遍历字典的迭代器，可在 for-range 中直接使用
	GetItems() iter.Seq2[K, V]

	// Clear 清空字典
	Clear()
}
