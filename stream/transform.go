package stream

// Map 将指定数据流处理后，输出到一个新的数据流
//   - s: 源数据流
//   - mapper: 元素转换函数
func Map[T, R any](s Stream[T], mapper func(item Item[T]) R) Stream[R] {
	if s == nil || mapper == nil {
		return newEmptyStream[R]()
	}

	ctx := pipelineContext(s)
	output := make(chan Item[R], getCacheSize())
	own := &stream[R]{source: output, ctx: ctx, clock: pipelineClock(s)}
	go func() {
		defer close(output)

		var index uint64
		for {
			item, ok := recv(ctx, s.GetChannel())
			if !ok {
				return
			}

			newItem := NewItem(mapper(item))
			newItem.SetIndex(index)
			index++

			if !send(ctx, output, newItem) {
				return
			}
		}
	}()

	return own
}

// Transform 处理
//   - transform: 处理函数
//   - 零值也会继续向下游流动
func (own *stream[T]) Transform(transform func(item Item[T]) T, callback ...func()) Stream[T] {
	if transform == nil {
		return own
	}

	ctx := own.ctx
	output := make(chan Item[T], getCacheSize())
	go func() {
		defer func() {
			close(output)
			runCallback(callback)
		}()

		for {
			item, ok := recv(ctx, own.source)
			if !ok {
				return
			}

			item.SetValue(transform(item))
			if !send(ctx, output, item) {
				return
			}
		}
	}()

	return own.wrap(output)
}

// Group 分组
//   - group: 分组函数，为每个元素设置分组，通过 item.GetGroup() 查询，为后续按组消费提供支撑
func (own *stream[T]) Group(group func(item Item[T]) string) Stream[T] {
	if group == nil {
		return own
	}

	ctx := own.ctx
	output := make(chan Item[T], getCacheSize())
	go func() {
		defer close(output)

		for {
			item, ok := recv(ctx, own.source)
			if !ok {
				return
			}

			item.SetGroup(group(item))
			if !send(ctx, output, item) {
				return
			}
		}
	}()

	return own.wrap(output)
}

// Filter 过滤
//   - filter: 过滤函数，满足过滤条件时返回 true（保留）；否则返回 false（丢弃）
func (own *stream[T]) Filter(filter func(item Item[T]) bool) Stream[T] {
	if filter == nil {
		return own
	}

	ctx := own.ctx
	output := make(chan Item[T], getCacheSize())
	go func() {
		defer close(output)

		for {
			item, ok := recv(ctx, own.source)
			if !ok {
				return
			}

			if !filter(item) {
				continue
			}

			if !send(ctx, output, item) {
				return
			}
		}
	}()

	return own.wrap(output)
}

// Peek 窥视
//   - peek: 窥视函数
func (own *stream[T]) Peek(peek func(item Item[T]), callback ...func()) Stream[T] {
	if peek == nil {
		return own
	}

	ctx := own.ctx
	output := make(chan Item[T], getCacheSize())
	go func() {
		defer func() {
			close(output)
			runCallback(callback)
		}()

		for {
			item, ok := recv(ctx, own.source)
			if !ok {
				return
			}

			peek(item)
			if !send(ctx, output, item) {
				return
			}
		}
	}()

	return own.wrap(output)
}
