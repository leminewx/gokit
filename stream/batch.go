package stream

import "time"

// Batch 批处理
//   - s: 数据流
//   - num: 批处理的数量
//   - timeout: 超时自动批处理；按固定周期检查，实际最大延迟为 timeout
func Batch[T any](s Stream[T], num int, timeout ...time.Duration) Stream[[]T] {
	if s == nil || num < 1 {
		return newEmptyStream[[]T]()
	}

	var period time.Duration
	if len(timeout) > 0 && timeout[0] > 0 {
		period = timeout[0]
	}

	ctx := pipelineContext(s)
	output := make(chan Item[[]T], getCacheSize())
	own := &stream[[]T]{source: output, ctx: ctx, clock: pipelineClock(s)}
	go func() {
		defer close(output)

		var index uint64
		var batch []T
		sendBatch := func() bool {
			if len(batch) == 0 {
				return true
			}

			// 发送后立即置空，下一次 append 会重新分配底层数组，不存在数据竞争
			item := NewItem(batch)
			item.SetIndex(index)
			index++
			batch = nil

			return send(ctx, output, item)
		}

		var flushChan <-chan time.Time
		if period > 0 {
			ticker := own.clock.NewTicker(period)
			defer ticker.Stop()

			flushChan = ticker.GetChannel()
		}

		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-s.GetChannel():
				if !ok {
					sendBatch()
					return
				}

				batch = append(batch, item.GetValue())
				if len(batch) < num {
					continue
				}

				if !sendBatch() {
					return
				}
			case <-flushChan:
				if !sendBatch() {
					return
				}
			}
		}
	}()

	return own
}
