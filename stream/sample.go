package stream

import "time"

// SampleByStep 根据步长采样（从第 1 条数据开始采样）
//   - size: 采样步长
func (own *stream[T]) SampleByStep(size int64) Stream[T] {
	if size < 2 {
		return own
	}

	ctx := own.ctx
	output := make(chan Item[T], getCacheSize())
	go func() {
		defer close(output)

		var cnt int64
		for {
			item, ok := recv(ctx, own.source)
			if !ok {
				return
			}

			if cnt == 0 {
				if !send(ctx, output, item) {
					return
				}
			}

			cnt = (cnt + 1) % size
		}
	}()

	return own.wrap(output)
}

// SampleByTime 根据时间采样（从第 1 条数据开始采样）
//   - interval: 采样间隔
//   - 每个间隔输出最近一次到达的元素；流结束时输出尚未输出的最后一条
func (own *stream[T]) SampleByTime(interval time.Duration) Stream[T] {
	if interval <= 0 {
		return own
	}

	ctx := own.ctx
	output := make(chan Item[T], getCacheSize())
	go func() {
		defer close(output)

		ticker := own.clock.NewTicker(interval)
		defer ticker.Stop()

		var hasItem bool
		var lastItem Item[T]
		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-own.source:
				if !ok {
					// 流结束，发送最后缓存的元素（如果有）
					if hasItem {
						send(ctx, output, lastItem)
					}

					return
				}

				hasItem = true
				lastItem = item
			case <-ticker.GetChannel():
				if !hasItem {
					continue
				}

				if !send(ctx, output, lastItem) {
					return
				}
				hasItem = false // 每个间隔只发一次
			}
		}
	}()

	return own.wrap(output)
}

// FanOut 将当前流散出到多个子流（不修改原流）
//   - num: 子流数量
//   - 子流不能有任意一阻塞，否则会导致整体阻塞
func (own *stream[T]) FanOut(num int) []Stream[T] {
	if num < 1 {
		return []Stream[T]{}
	}

	ctx := own.ctx
	outputs := make([]chan Item[T], num)
	streams := make([]Stream[T], num)
	for i := range num {
		outputs[i] = make(chan Item[T], getCacheSize())
		streams[i] = &stream[T]{source: outputs[i], ctx: ctx, clock: own.clock}
	}

	go func() {
		defer func() {
			for _, output := range outputs {
				close(output)
			}
		}()

		for {
			item, ok := recv(ctx, own.source)
			if !ok {
				return
			}

			// 先快照再分发：子流接收到元素后可能原地修改它，
			// 投递之后再读取会与下游形成数据竞争
			value := item.GetValue()
			index := item.GetIndex()
			group := item.GetGroup()

			for i, output := range outputs {
				// 每个子流使用独立的元素副本，避免下游 Transform / Group 修改共享元素时互相影响
				target := item
				if i > 0 {
					clone := NewItem(value)
					clone.SetIndex(index)
					clone.SetGroup(group)
					target = clone
				}

				if !send(ctx, output, target) {
					return
				}
			}
		}
	}()

	return streams
}
