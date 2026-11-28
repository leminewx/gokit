package stream

import (
	"sync"
	"time"
)

// Concat 聚合
//   - streams: 待聚合数据流，按当前流、参数流依次拼接
func (own *stream[T]) Concat(streams ...Stream[T]) Stream[T] {
	if len(streams) == 0 {
		return own
	}

	ctx := own.ctx
	output := make(chan Item[T], getCacheSize())
	go func() {
		defer close(output)

		var wg sync.WaitGroup
		for _, source := range append([]Stream[T]{own}, streams...) {
			if source == nil {
				continue
			}

			wg.Go(func() {
				for {
					item, ok := recv(ctx, source.GetChannel())
					if !ok {
						return
					}

					if !send(ctx, output, item) {
						return
					}
				}
			})
		}
		wg.Wait()
	}()

	return own.wrap(output)
}

// TakeFirst 只接收数据流前 num 个元素
func (own *stream[T]) TakeFirst(num int) Stream[T] {
	if num < 1 {
		return newEmptyStream[T]()
	}

	ctx := own.ctx
	output := make(chan Item[T], getCacheSize())
	go func() {
		defer func() {
			close(output)
			// 达到上限后继续排空上游，避免上游 goroutine 阻塞在投递上
			go drain(ctx, own.source)
		}()

		for range num {
			item, ok := recv(ctx, own.source)
			if !ok {
				return
			}

			if !send(ctx, output, item) {
				return
			}
		}
	}()

	return own.wrap(output)
}

// TakeLast 只接收数据流后 num 个元素
func (own *stream[T]) TakeLast(num int) Stream[T] {
	if num < 1 {
		return newEmptyStream[T]()
	}

	ctx := own.ctx
	output := make(chan Item[T], getCacheSize())
	go func() {
		defer close(output)

		total := 0
		items := make([]Item[T], num)
		for {
			item, ok := recv(ctx, own.source)
			if !ok {
				break
			}

			items[total%num] = item
			total++
		}

		if total == 0 {
			return
		}

		// 环形缓冲区不满，从 0 开始输出
		if total < num {
			for i := range total {
				if !send(ctx, output, items[i]) {
					return
				}
			}

			return
		}

		// 缓冲区已满，从 (total % num) 开始顺序输出 num 个
		start := total % num
		for i := range num {
			if !send(ctx, output, items[(start+i)%num]) {
				return
			}
		}
	}()

	return own.wrap(output)
}

// Deduplicate 去重
//   - deduplicate: 去重函数
//   - window: 设置一个合理的去重时间窗口，防止内存无限增长（推荐使用）
func (own *stream[T]) Deduplicate(deduplicate func(item Item[T]) any, window ...time.Duration) Stream[T] {
	if deduplicate == nil {
		return own
	}

	ctx := own.ctx

	// 未使用时间窗口：记录全部已出现的 key
	if len(window) == 0 || window[0] < 1 {
		output := make(chan Item[T], getCacheSize())
		go func() {
			defer close(output)

			keys := make(map[any]struct{})
			for {
				item, ok := recv(ctx, own.source)
				if !ok {
					return
				}

				key := deduplicate(item)
				if _, exists := keys[key]; exists {
					continue
				}
				keys[key] = struct{}{}

				if !send(ctx, output, item) {
					return
				}
			}
		}()

		return own.wrap(output)
	}

	// 使用时间窗口：按最近出现时间淘汰 key
	period := window[0]
	interval := period / 2
	if interval <= 0 {
		interval = period
	}

	output := make(chan Item[T], getCacheSize())
	go func() {
		cache := make(map[any]time.Time)
		var cacheLock sync.Mutex
		done := make(chan struct{})

		ticker := own.clock.NewTicker(interval) // 定时清理过期 key
		defer func() {
			ticker.Stop()
			close(done)
			close(output)
		}()

		go func() {
			for {
				select {
				case now := <-ticker.GetChannel():
					cacheLock.Lock()
					for key, lastSeen := range cache {
						if now.Sub(lastSeen) > period {
							delete(cache, key)
						}
					}
					cacheLock.Unlock()
				case <-done:
					return
				}
			}
		}()

		for {
			item, ok := recv(ctx, own.source)
			if !ok {
				return
			}

			key := deduplicate(item)
			now := own.clock.GetNow()

			cacheLock.Lock()
			lastSeen, exists := cache[key]
			if exists && now.Sub(lastSeen) <= period {
				cacheLock.Unlock()
				continue // 窗口内重复，跳过
			}

			cache[key] = now // 更新或插入
			cacheLock.Unlock()

			if !send(ctx, output, item) {
				return
			}
		}
	}()

	return own.wrap(output)
}
