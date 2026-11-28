package dict

import (
	"iter"
	"sync"
)

var _ Dict[int, int] = &SafeDict[int, int]{}

// SafeDict 定义基于 sync.RWMutex 的并发安全的字典类型
type SafeDict[K comparable, V any] struct {
	dict *UnsafeDict[K, V]
	lock sync.RWMutex
}

// NewSafeDict 新建一个基于 sync.RWMutex 的并发安全的字典
func NewSafeDict[K comparable, V any]() *SafeDict[K, V] {
	return &SafeDict[K, V]{dict: NewUnsafeDict[K, V]()}
}

// Set 设置键值对
func (own *SafeDict[K, V]) Set(key K, val V) {
	own.lock.Lock()
	own.dict.Set(key, val)
	own.lock.Unlock()
}

// Update 更新字典
func (own *SafeDict[K, V]) Update(items map[K]V) {
	own.lock.Lock()
	own.dict.Update(items)
	own.lock.Unlock()
}

// Delete 删除键值对
func (own *SafeDict[K, V]) Delete(keys ...K) {
	own.lock.Lock()
	own.dict.Delete(keys...)
	own.lock.Unlock()
}

// HasAll 判断所有指定键值对是否存在
func (own *SafeDict[K, V]) HasAll(keys ...K) bool {
	own.lock.RLock()
	ok := own.dict.HasAll(keys...)
	own.lock.RUnlock()
	return ok
}

// HasAny 判断任一指定元素是存在
func (own *SafeDict[K, V]) HasAny(keys ...K) bool {
	own.lock.RLock()
	ok := own.dict.HasAny(keys...)
	own.lock.RUnlock()
	return ok
}

// Get 查询键值对
func (own *SafeDict[K, V]) Get(key K) (V, bool) {
	own.lock.RLock()
	val, ok := own.dict.Get(key)
	own.lock.RUnlock()
	return val, ok
}

// GetLength 获取字典长度
func (own *SafeDict[K, V]) GetLength() int {
	own.lock.RLock()
	length := own.dict.GetLength()
	own.lock.RUnlock()
	return length
}

// GetKeys 获取字典中所有的键
func (own *SafeDict[K, V]) GetKeys() []K {
	own.lock.RLock()
	keys := own.dict.GetKeys()
	own.lock.RUnlock()
	return keys
}

// GetValues 获取字典中所有的值
func (own *SafeDict[K, V]) GetValues() []V {
	own.lock.RLock()
	values := own.dict.GetValues()
	own.lock.RUnlock()
	return values
}

// GetMap 获取Map
func (own *SafeDict[K, V]) GetMap() map[K]V {
	own.lock.RLock()
	data := own.dict.GetMap()
	own.lock.RUnlock()
	return data
}

// Range 遍历字典
//   - 先对键值对做快照再遍历，避免在回调执行期间持有读锁：
//     回调内调用 Set/Delete 会因 RWMutex 不可重入而永久阻塞
func (own *SafeDict[K, V]) Range(fn func(key K, val V)) {
	for key, val := range own.GetMap() {
		fn(key, val)
	}
}

// GetItems 返回遍历字典的迭代器，可在 for-range 中直接使用
//   - 遍历前先对键值对做快照，避免在 yield 期间持有读锁
func (own *SafeDict[K, V]) GetItems() iter.Seq2[K, V] {
	return func(yield func(key K, val V) bool) {
		for key, val := range own.GetMap() {
			if !yield(key, val) {
				return
			}
		}
	}
}

// Filter 删除并返回符合过滤要求的键值对
//   - 会修改字典内容，必须使用写锁；内层 Filter 会直接 delete 底层 map
func (own *SafeDict[K, V]) Filter(fn func(key K, val V) bool) map[K]V {
	own.lock.Lock()
	items := own.dict.Filter(fn)
	own.lock.Unlock()
	return items
}

// Transform 遍历并更新每个元素
func (own *SafeDict[K, V]) Transform(fn func(key K, val V) V) {
	own.lock.Lock()
	own.dict.Transform(fn)
	own.lock.Unlock()
}

// Clear 清空字典
func (own *SafeDict[K, V]) Clear() {
	own.lock.Lock()
	own.dict.Clear()
	own.lock.Unlock()
}
