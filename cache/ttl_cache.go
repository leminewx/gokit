package cache

import (
	"context"
	"sync"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

type (
	// TTLCache 带过期时间的缓存
	TTLCache[K comparable, V any] struct {
		lock       sync.RWMutex
		items      map[K]*ttlEntry[V]
		defaultTTL time.Duration
		source     timex.Clock
		onEvict    func(key K, value V)
	}

	ttlEntry[V any] struct {
		value   V
		expires time.Time
	}
)

// NewTTLCache 创建 TTL 缓存，defaultTTL=0 表示默认永不过期（需每次 Put 指定）
//   - 可选项：WithClock 注入时间源（过期判定与自动清理均使用该时间源），
//     WithOnEvict 指定元素被移除时的回调
func NewTTLCache[K comparable, V any](defaultTTL time.Duration, optionList ...Option[K, V]) *TTLCache[K, V] {
	settings := applyOptions(optionList)
	return &TTLCache[K, V]{
		items:      make(map[K]*ttlEntry[V]),
		defaultTTL: defaultTTL,
		source:     settings.source,
		onEvict:    settings.onEvict,
	}
}

// GetDefaultTTL 返回默认存活时间，0 表示默认永不过期
func (own *TTLCache[K, V]) GetDefaultTTL() time.Duration {
	return own.defaultTTL
}

// Put 写入键值，ttl<=0 时使用默认 TTL，默认也为 0 则永不过期
func (own *TTLCache[K, V]) Put(key K, value V, ttl time.Duration) {
	if ttl <= 0 {
		ttl = own.defaultTTL
	}

	var exp time.Time
	if ttl > 0 {
		exp = own.source.GetNow().Add(ttl)
	}

	own.lock.Lock()
	own.items[key] = &ttlEntry[V]{value: value, expires: exp}
	own.lock.Unlock()
}

// Get 取值，若已过期则返回 (zero, false) 并删除
func (own *TTLCache[K, V]) Get(key K) (V, bool) {
	value, _, ok := own.lookup(key)
	return value, ok
}

// GetWithTTL 取值并返回剩余有效期（永不过期项返回 0）
func (own *TTLCache[K, V]) GetWithTTL(key K) (V, time.Duration, bool) {
	return own.lookup(key)
}

// Peek 取值但不改变 LRU 语义以外的状态，过期项按不存在处理（与 Get 一致）
func (own *TTLCache[K, V]) Peek(key K) (V, bool) {
	return own.Get(key)
}

// Contain 判断键是否存在且未过期，会触发过期项的懒删除
func (own *TTLCache[K, V]) Contain(key K) bool {
	_, ok := own.Get(key)
	return ok
}

// lookup 读取键值并返回剩余有效期，过期项就地删除（懒清理）
func (own *TTLCache[K, V]) lookup(key K) (V, time.Duration, bool) {
	own.lock.RLock()
	el, ok := own.items[key]
	if !ok {
		own.lock.RUnlock()

		var zero V
		return zero, 0, false
	}

	now := own.source.GetNow()
	if !el.expires.IsZero() && now.After(el.expires) {
		own.lock.RUnlock()
		own.removeExpired(key, el)

		var zero V
		return zero, 0, false
	}

	remain := time.Duration(0)
	if !el.expires.IsZero() {
		remain = el.expires.Sub(now)
	}
	value := el.value
	own.lock.RUnlock()

	return value, remain, true
}

// removeExpired 删除过期元素，仅当键仍指向同一元素时才删除并回调
func (own *TTLCache[K, V]) removeExpired(key K, stale *ttlEntry[V]) {
	own.lock.Lock()
	current, ok := own.items[key]
	if !ok || current != stale {
		own.lock.Unlock()
		return
	}
	delete(own.items, key)
	own.lock.Unlock()

	own.notifyEvict(key, stale.value)
}

// Delete 删除键并返回被删除的值
func (own *TTLCache[K, V]) Delete(key K) (V, bool) {
	own.lock.Lock()
	el, ok := own.items[key]
	if !ok {
		own.lock.Unlock()

		var zero V
		return zero, false
	}
	delete(own.items, key)
	own.lock.Unlock()

	own.notifyEvict(key, el.value)
	return el.value, true
}

// GetKeys 返回当前所有键（含可能已过期的，过期清理为懒执行）
func (own *TTLCache[K, V]) GetKeys() []K {
	own.lock.RLock()
	defer own.lock.RUnlock()

	keys := make([]K, 0, len(own.items))
	for key := range own.items {
		keys = append(keys, key)
	}

	return keys
}

// GetLength 返回当前元素个数（含可能已过期的）
func (own *TTLCache[K, V]) GetLength() int {
	own.lock.RLock()
	defer own.lock.RUnlock()

	return len(own.items)
}

// Clear 清空缓存并返回清除的元素个数，每个元素都会触发淘汰回调
func (own *TTLCache[K, V]) Clear() int {
	own.lock.Lock()
	entries := own.items
	own.items = make(map[K]*ttlEntry[V])
	own.lock.Unlock()

	for key, el := range entries {
		own.notifyEvict(key, el.value)
	}

	return len(entries)
}

// Purge 清理所有已过期的元素，返回清理个数
func (own *TTLCache[K, V]) Purge() int {
	now := own.source.GetNow()

	own.lock.Lock()
	expired := make(map[K]*ttlEntry[V])
	for key, el := range own.items {
		if !el.expires.IsZero() && now.After(el.expires) {
			expired[key] = el
			delete(own.items, key)
		}
	}
	own.lock.Unlock()

	for key, el := range expired {
		own.notifyEvict(key, el.value)
	}

	return len(expired)
}

// StartAutoPurge 启动后台定期清理协程，每隔 interval 清理一次过期元素
//   - ctx 为 nil 时按 context.Background() 处理
//   - interval <= 0 时按 1 分钟处理
//   - 清理周期由构造时注入的时钟驱动，测试可用 timex.Manual 推进
//   - 返回的停止函数可重复调用，与 context 取消均可优雅关闭，无协程泄漏
func (own *TTLCache[K, V]) StartAutoPurge(ctx context.Context, interval time.Duration) (stop func()) {
	if ctx == nil {
		ctx = context.Background()
	}

	if interval <= 0 {
		interval = time.Minute
	}

	done := make(chan struct{})
	// 同步创建嘀嗒器，保证调用方在 StartAutoPurge 返回后即可推进手动时钟
	ticker := own.source.NewTicker(interval)
	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-ticker.GetChannel():
				own.Purge()
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
		})
	}
}

// notifyEvict 在锁外触发淘汰回调
func (own *TTLCache[K, V]) notifyEvict(key K, value V) {
	if own.onEvict != nil {
		own.onEvict(key, value)
	}
}
