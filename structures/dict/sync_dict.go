package dict

import (
	"iter"
	"sync"
)

var _ Dict[int, int] = (*SyncDict[int, int])(nil)

// SyncDict 定义基于 sync.Map 的并发安全的字典类型
type SyncDict[K comparable, V any] struct {
	dict sync.Map
}

// NewSyncDict 新建一个基于 sync.Map 的并发安全的字典
func NewSyncDict[K comparable, V any]() *SyncDict[K, V] {
	return &SyncDict[K, V]{}
}

// Set 设置键值对
func (own *SyncDict[K, V]) Set(key K, val V) {
	own.dict.Store(key, val)
}

// Update 更新字典
func (own *SyncDict[K, V]) Update(items map[K]V) {
	for k, v := range items {
		own.dict.Store(k, v)
	}
}

// Delete 删除键值对
func (own *SyncDict[K, V]) Delete(keys ...K) {
	for _, k := range keys {
		own.dict.Delete(k)
	}
}

// HasAll 判断所有指定键值对是否存在
func (own *SyncDict[K, V]) HasAll(keys ...K) bool {
	for _, k := range keys {
		if _, ok := own.dict.Load(k); !ok {
			return false
		}
	}

	return true
}

// HasAny 判断任一指定元素是存在
func (own *SyncDict[K, V]) HasAny(keys ...K) bool {
	for _, k := range keys {
		if _, ok := own.dict.Load(k); ok {
			return true
		}
	}

	return false
}

// Get 查询键值对
func (own *SyncDict[K, V]) Get(key K) (V, bool) {
	if val, ok := own.dict.Load(key); ok {
		return val.(V), true
	}

	var val V
	return val, false
}

// GetLength 获取字典长度
func (own *SyncDict[K, V]) GetLength() (length int) {
	own.dict.Range(func(key, value any) bool {
		length++
		return true
	})

	return
}

// GetKeys 获取字典中所有的键
func (own *SyncDict[K, V]) GetKeys() []K {
	keys := make([]K, 0)
	own.dict.Range(func(key, value any) bool {
		keys = append(keys, key.(K))
		return true
	})

	return keys
}

// GetValues 获取字典中所有的值
func (own *SyncDict[K, V]) GetValues() []V {
	values := make([]V, 0)
	own.dict.Range(func(key, value any) bool {
		values = append(values, value.(V))
		return true
	})

	return values
}

// GetMap 获取Map
func (own *SyncDict[K, V]) GetMap() map[K]V {
	dict := make(map[K]V)
	own.dict.Range(func(key, value any) bool {
		dict[key.(K)] = value.(V)
		return true
	})

	return dict
}

// Range 遍历字典
func (own *SyncDict[K, V]) Range(fn func(key K, val V)) {
	own.dict.Range(func(key, value any) bool {
		fn(key.(K), value.(V))
		return true
	})
}

// GetItems 返回遍历字典的迭代器，可在 for-range 中直接使用
//   - sync.Map 的 Range 不持有全局锁，遍历结果弱一致
func (own *SyncDict[K, V]) GetItems() iter.Seq2[K, V] {
	return func(yield func(key K, val V) bool) {
		own.dict.Range(func(key, value any) bool {
			return yield(key.(K), value.(V))
		})
	}
}

// Filter 删除并返回符合过滤要求的键值对
func (own *SyncDict[K, V]) Filter(fn func(key K, val V) bool) map[K]V {
	dict := make(map[K]V)
	own.dict.Range(func(key, value any) bool {
		if k, v := key.(K), value.(V); fn(k, v) {
			dict[k] = v
		}
		return true
	})

	for key := range dict {
		own.dict.Delete(key)
	}

	return dict
}

// Transform 遍历并更新每个元素
func (own *SyncDict[K, V]) Transform(fn func(key K, val V) V) {
	tmpDict := make(map[K]V)
	own.dict.Range(func(key, value any) bool {
		k := key.(K)
		tmpDict[k] = fn(k, value.(V))
		return true
	})

	for k, v := range tmpDict {
		own.dict.Store(k, v)
	}
}

// Clear 清空字典
func (own *SyncDict[K, V]) Clear() {
	own.dict.Clear()
}
