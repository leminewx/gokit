// Package cache 提供两个线程安全、零依赖的通用缓存实现：
//   - 基于容量淘汰的 LRU 缓存与基于时间过期的 TTL 缓存。
//   - 两者均为泛型实现，键类型需满足 comparable，值类型任意。
//
// 设计目标：
//   - 不依赖任何第三方库，仅使用 Go 标准库与 gokit/timex；
//   - 全部方法线程安全：LRUCache 使用 sync.Mutex，TTLCache 使用 sync.RWMutex；
//   - 语义清晰、内存可控，并在 TTL 场景下避免协程泄漏；
//   - 淘汰/过期事件可通过 WithOnEvict 回调感知，时间源可通过 WithClock 注入
//     （WithClock 仅对 TTL 缓存生效，LRU 不依赖时间）。
//
// # LRU 缓存（LRUCache）
//
// 最近最少使用（Least Recently Used）策略：当写入导致元素数量超过容量时，
// 自动淘汰"最久未被访问"的元素（Get/Put 命中都会刷新为最近使用）。
//
//	c := cache.NewLRUCache[string, int](128)
//	c.Put("hits", 1)
//	if v, ok := c.Get("hits"); ok {
//	    // v == 1
//	}
//
// 主要方法：
//   - NewLRUCache[K, V](capacity, options...)：capacity<=0 表示不限制容量；
//   - Get：取值并刷新为最近使用；Peek / Contain：取值或判存但不改变顺序；
//   - Put：写入，返回是否触发了容量淘汰；Delete：删除并返回旧值；
//   - GetKeys：按键的最近使用顺序导出快照（最近使用在前）；
//   - GetLength / GetCapacity：当前元素个数与容量上限；
//   - Clear：清空并返回清除个数。
//
// # TTL 缓存（TTLCache）
//
// 基于过期时间的缓存：每个键可携带独立的存活时间（TTL）。读取已过期键时
// 自动返回零值并就地删除（懒清理）。
//
//	c := cache.NewTTLCache[string, int](time.Minute) // 默认 1 分钟
//	c.Put("token", 42, 0)                            // 使用默认 TTL
//	c.Put("code", 7, 30*time.Second)                 // 单次指定 30 秒
//	if v, ok := c.Get("token"); ok { /* v == 42 */ }
//
// 主要方法：
//   - NewTTLCache[K, V](defaultTTL, options...)：defaultTTL=0 表示默认永不过期；
//   - Put(key, value, ttl)：ttl<=0 时回退到默认 TTL，默认仍为 0 则永不过期；
//   - Get / GetWithTTL：取值；GetWithTTL 额外返回剩余有效期（永不过期为 0）；
//   - Contain：判断键存在且未过期（会触发过期项懒删除）；
//   - Delete / Clear：删除单个键或清空，均返回被移除的值或个数；
//   - GetKeys / GetLength：导出键快照与统计元素个数（含可能已过期的）；
//   - GetDefaultTTL：查询默认存活时间；
//   - Purge：主动清理所有已过期元素，返回清理个数；
//   - StartAutoPurge(ctx, interval)：启动后台定期清理协程，清理周期由注入
//     的时钟驱动；返回的 stop() 可重复调用，与 context 取消均可优雅关闭。
//
// 示例（注入手动时钟，使过期行为可确定性测试）：
//
//	source := timex.NewManual(time.Unix(0, 0).UTC())
//	c := cache.NewTTLCache[string, int](time.Minute, cache.WithClock[string, int](source))
//	c.Put("k", 1, 0)
//	source.Advance(2 * time.Minute)
//	_, ok := c.Get("k") // ok == false
//
// 注意事项：
//   - 懒清理：Get 命中过期键会触发删除，但 GetLength 在 Purge 之前可能
//     仍包含过期项，这是有意为之的轻量设计；
//   - 如需严格控制内存，建议配合 StartAutoPurge 或定期调用 Purge；
//   - 淘汰回调在锁外执行，可在回调内安全地操作缓存本身。
//
// 本包除 gokit/timex（可注入时间源）外不依赖 gokit 的其它模块，可独立使用。
package cache
