package stream

import (
	"context"
	"time"
)

// ZipLeft 按 s1（左侧）的顺序进行对齐配对
//
// 对于 s1 的每个元素，按 s2 的顺序寻找第一个满足 match 条件的元素（从 s2 当前未消费的位置开始），找到后，通过 mapper 输出 R，然后继续下一个 s1 元素。
// 如果超时时间内未找到，根据 onTimeout 决定行为
//   - s1, s2: 数据流，s2 向 s1 对齐
//   - match: 对齐匹配函数
//   - mapper: s1 和 s2 对齐之后，调用 mapper 生成 R
//   - timeout: 元素对齐超时时间，<= 0 时表示不等待
//   - onTimeout: 元素对齐超时时的处理（若返回 true，则消费，反之则抛弃）
func ZipLeft[T, U, R any](
	s1 Stream[T], s2 Stream[U],
	match func(a Item[T], b Item[U]) bool,
	mapper func(a Item[T], b Item[U]) R,
	timeout time.Duration, onTimeout func(a Item[T]) (R, bool),
) Stream[R] {
	if s1 == nil || s2 == nil || match == nil || mapper == nil {
		return newEmptyStream[R]()
	}

	parent := pipelineContext(s1)
	source := pipelineClock(s1)
	output := make(chan Item[R], getCacheSize())
	own := &stream[R]{source: output, ctx: parent, clock: source}
	go func() {
		// 派生可取消的上下文：主循环结束时一并终止读取 s2 的协程，避免 goroutine 泄漏
		ctx, cancel := context.WithCancel(parent)
		defer cancel()
		defer close(output)

		var (
			index   uint64
			pending []Item[U] // 缓存 s2 中未匹配的元素（按顺序）
		)

		pendingChan := make(chan Item[U], getCacheSize())

		// 后台 goroutine 持续读取 s2 并放入 pending 队列
		go func() {
			defer close(pendingChan)

			for {
				item, ok := recv(ctx, s2.GetChannel())
				if !ok {
					return
				}

				if !send(ctx, pendingChan, item) {
					return
				}
			}
		}()

		// findMatch 从 pending 中寻找第一个匹配的元素（并移除它）
		findMatch := func(a Item[T]) (Item[U], bool) {
			for i, b := range pending {
				if match(a, b) {
					pending = append(pending[:i], pending[i+1:]...)
					return b, true
				}
			}

			return nil, false
		}

		// waitMatch 继续从 s2 读取，直到匹配成功、s2 结束、超时或流水线取消
		waitMatch := func(a Item[T]) (Item[U], bool) {
			timeoutChan, stop := after(source, timeout)
			defer stop()

			for {
				select {
				case <-ctx.Done():
					return nil, false
				case bItem, ok := <-pendingChan:
					if !ok {
						return nil, false
					}

					pending = append(pending, bItem)
					if b, matched := findMatch(a); matched {
						return b, true
					}
				case <-timeoutChan:
					return nil, false
				}
			}
		}

		// emit 输出对齐结果
		emit := func(value R) bool {
			item := NewItem(value)
			item.SetIndex(index)
			index++

			return send(ctx, output, item)
		}

		for {
			a, ok := recv(ctx, s1.GetChannel())
			if !ok {
				return
			}

			b, matched := findMatch(a)
			if !matched {
				b, matched = waitMatch(a)
			}
			if ctx.Err() != nil {
				return
			}

			if matched {
				if !emit(mapper(a, b)) {
					return
				}

				continue
			}

			// 超时未匹配，调用 onTimeout 决定是否输出默认值
			if onTimeout == nil {
				continue
			}

			if value, ok := onTimeout(a); ok {
				if !emit(value) {
					return
				}
			}
		}
	}()

	return own
}

// ZipLeftByKey 按 s1（左侧）的顺序进行等值键对齐配对
//
// 对于 s1 的每个元素，提取 key1，然后从 s2 的缓存中查找相同 key2 的第一个元素（按 s2 到达顺序），找到后，应用 mapper 输出 R，并从 s2 缓存中移除该元素。
// 如果超时时间内未找到，根据 onTimeout 决定行为。使用 map 索引，匹配效率 O(1)，适用于等值连接场景
//   - s1, s2: 数据流，s2 向 s1 对齐
//   - keyFunc1, keyFunc2: 用于分别生成 s1 和 s2 对齐的 key，适用基于某个键的等值比较（如：ID、按时间戳等）
//   - mapper: s1 和 s2 对齐之后，调用 mapper 生成 R
//   - timeout: 元素对齐超时时间，<= 0 时表示不等待
//   - onTimeout: 元素对齐超时时的处理（若返回 true，则消费，反之则抛弃）
func ZipLeftByKey[T, U, R any, K comparable](
	s1 Stream[T], s2 Stream[U],
	keyFunc1 func(a Item[T]) K, keyFunc2 func(b Item[U]) K,
	mapper func(a Item[T], b Item[U]) R,
	timeout time.Duration, onTimeout func(a Item[T]) (R, bool),
) Stream[R] {
	if s1 == nil || s2 == nil || keyFunc1 == nil || keyFunc2 == nil || mapper == nil {
		return newEmptyStream[R]()
	}

	parent := pipelineContext(s1)
	source := pipelineClock(s1)
	output := make(chan Item[R], getCacheSize())
	own := &stream[R]{source: output, ctx: parent, clock: source}
	go func() {
		// 派生可取消的上下文：主循环结束时一并终止读取 s2 的协程，避免 goroutine 泄漏
		ctx, cancel := context.WithCancel(parent)
		defer cancel()
		defer close(output)

		var (
			index uint64
			cache = make(map[K][]Item[U]) // s2 缓存：key -> 队列（按到达顺序存储同一个 key 的多个元素）
		)

		pendingChan := make(chan Item[U], getCacheSize())
		go func() {
			defer close(pendingChan)

			for {
				item, ok := recv(ctx, s2.GetChannel())
				if !ok {
					return
				}

				if !send(ctx, pendingChan, item) {
					return
				}
			}
		}()

		// tryMatch 从缓存中取出第一个匹配的 item
		tryMatch := func(k K) (Item[U], bool) {
			queue, ok := cache[k]
			if !ok || len(queue) == 0 {
				return nil, false
			}

			first := queue[0]
			if len(queue) == 1 {
				delete(cache, k)
			} else {
				cache[k] = queue[1:]
			}

			return first, true
		}

		// waitMatch 继续从 s2 读取，直到匹配成功、s2 结束、超时或流水线取消
		waitMatch := func(k K) (Item[U], bool) {
			timeoutChan, stop := after(source, timeout)
			defer stop()

			for {
				select {
				case <-ctx.Done():
					return nil, false
				case bItem, ok := <-pendingChan:
					if !ok {
						return nil, false
					}

					// 将新元素加入缓存（按 key 分组），并尝试匹配当前 key
					k2 := keyFunc2(bItem)
					cache[k2] = append(cache[k2], bItem)
					if b, matched := tryMatch(k); matched {
						return b, true
					}
				case <-timeoutChan:
					return nil, false
				}
			}
		}

		// emit 输出对齐结果
		emit := func(value R) bool {
			item := NewItem(value)
			item.SetIndex(index)
			index++

			return send(ctx, output, item)
		}

		for {
			a, ok := recv(ctx, s1.GetChannel())
			if !ok {
				return
			}

			k := keyFunc1(a)
			b, matched := tryMatch(k)
			if !matched {
				b, matched = waitMatch(k)
			}
			if ctx.Err() != nil {
				return
			}

			if matched {
				if !emit(mapper(a, b)) {
					return
				}

				continue
			}

			if onTimeout == nil {
				continue
			}

			if value, ok := onTimeout(a); ok {
				if !emit(value) {
					return
				}
			}
		}
	}()

	return own
}
