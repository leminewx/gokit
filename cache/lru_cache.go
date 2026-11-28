package cache

import (
	"container/list"
	"sync"
)

type (
	// LRUCache 线程安全的 LRU 缓存
	LRUCache[K comparable, V any] struct {
		lock     sync.Mutex
		capacity int
		ll       *list.List
		items    map[K]*list.Element
		onEvict  func(key K, value V)
	}

	lruEntry[K comparable, V any] struct {
		key   K
		value V
	}
)

// NewLRUCache 创建容量为 capacity 的 LRU 缓存（capacity <= 0 表示不限制）
//   - 可选项：WithOnEvict 指定淘汰回调，WithClock 仅为接口一致性保留
func NewLRUCache[K comparable, V any](capacity int, optionList ...Option[K, V]) *LRUCache[K, V] {
	settings := applyOptions(optionList)
	return &LRUCache[K, V]{
		capacity: capacity,
		ll:       list.New(),
		items:    make(map[K]*list.Element),
		onEvict:  settings.onEvict,
	}
}

// Get 取值并将元素移至最近使用位置
func (own *LRUCache[K, V]) Get(key K) (V, bool) {
	own.lock.Lock()
	el, ok := own.items[key]
	if !ok {
		own.lock.Unlock()

		var zero V
		return zero, false
	}

	own.ll.MoveToFront(el)
	value := el.Value.(*lruEntry[K, V]).value
	own.lock.Unlock()

	return value, true
}

// Peek 取值但不改变元素的使用顺序
func (own *LRUCache[K, V]) Peek(key K) (V, bool) {
	own.lock.Lock()
	defer own.lock.Unlock()

	el, ok := own.items[key]
	if !ok {
		var zero V
		return zero, false
	}

	return el.Value.(*lruEntry[K, V]).value, true
}

// Contain 判断键是否存在，不改变元素的使用顺序
func (own *LRUCache[K, V]) Contain(key K) bool {
	own.lock.Lock()
	defer own.lock.Unlock()

	_, ok := own.items[key]
	return ok
}

// Put 写入键值，超出容量时淘汰最久未使用元素
//   - 返回值为 true 表示本次写入触发了一次容量淘汰
func (own *LRUCache[K, V]) Put(key K, value V) (evicted bool) {
	own.lock.Lock()
	if el, ok := own.items[key]; ok {
		own.ll.MoveToFront(el)
		el.Value.(*lruEntry[K, V]).value = value
		own.lock.Unlock()
		return false
	}

	el := own.ll.PushFront(&lruEntry[K, V]{key: key, value: value})
	own.items[key] = el

	var (
		oldKey   K
		oldValue V
	)
	if own.capacity > 0 && own.ll.Len() > own.capacity {
		oldKey, oldValue, evicted = own.removeOldest()
	}
	own.lock.Unlock()

	if evicted {
		own.notifyEvict(oldKey, oldValue)
	}

	return evicted
}

// removeOldest 移除最久未使用元素并返回其键值（调用方需持有锁）
func (own *LRUCache[K, V]) removeOldest() (key K, value V, ok bool) {
	el := own.ll.Back()
	if el == nil {
		return key, value, false
	}

	entry := el.Value.(*lruEntry[K, V])
	own.ll.Remove(el)
	delete(own.items, entry.key)

	return entry.key, entry.value, true
}

// Delete 删除键并返回被删除的值
func (own *LRUCache[K, V]) Delete(key K) (V, bool) {
	own.lock.Lock()
	el, ok := own.items[key]
	if !ok {
		own.lock.Unlock()

		var zero V
		return zero, false
	}

	entry := el.Value.(*lruEntry[K, V])
	own.ll.Remove(el)
	delete(own.items, key)
	own.lock.Unlock()

	own.notifyEvict(entry.key, entry.value)
	return entry.value, true
}

// GetKeys 返回当前所有键，最近使用在前
func (own *LRUCache[K, V]) GetKeys() []K {
	own.lock.Lock()
	defer own.lock.Unlock()

	keys := make([]K, 0, len(own.items))
	for el := own.ll.Front(); el != nil; el = el.Next() {
		keys = append(keys, el.Value.(*lruEntry[K, V]).key)
	}

	return keys
}

// GetLength 返回当前元素个数
func (own *LRUCache[K, V]) GetLength() int {
	own.lock.Lock()
	defer own.lock.Unlock()

	return own.ll.Len()
}

// GetCapacity 返回容量上限，<=0 表示不限制
func (own *LRUCache[K, V]) GetCapacity() int {
	return own.capacity
}

// Clear 清空缓存并返回清除的元素个数，每个元素都会触发淘汰回调
func (own *LRUCache[K, V]) Clear() int {
	own.lock.Lock()
	entries := make([]lruEntry[K, V], 0, len(own.items))
	for el := own.ll.Front(); el != nil; el = el.Next() {
		entries = append(entries, *el.Value.(*lruEntry[K, V]))
	}
	own.ll.Init()
	own.items = make(map[K]*list.Element)
	own.lock.Unlock()

	for _, entry := range entries {
		own.notifyEvict(entry.key, entry.value)
	}

	return len(entries)
}

// notifyEvict 在锁外触发淘汰回调
func (own *LRUCache[K, V]) notifyEvict(key K, value V) {
	if own.onEvict != nil {
		own.onEvict(key, value)
	}
}
