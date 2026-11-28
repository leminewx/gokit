package stream

import (
	"context"

	"gitee.com/leminewx/gokit/timex"
)

// *********************************************************************************************************************
// 标准流：所有中间算子均记录为独立的 goroutine，算子之间通过 channel 串联，逐元素向下游推送
// *********************************************************************************************************************

// stream 定义数据流的内置实现
type stream[T any] struct {
	source <-chan Item[T]  // 上游数据
	ctx    context.Context // 取消信号，派生流自动继承
	clock  timex.Clock     // 时间源，供时间相关算子使用
}

// 编译期断言：内置流实现 Stream 接口
var _ Stream[int] = (*stream[int])(nil)

// WithContext 替换当前流派生出的下游算子所使用的取消信号
func (own *stream[T]) WithContext(ctx context.Context) Stream[T] {
	if ctx == nil {
		ctx = context.Background()
	}

	clone := *own
	clone.ctx = ctx

	return &clone
}

// WithClock 替换当前流派生出的下游算子所使用的时间源
func (own *stream[T]) WithClock(source timex.Clock) Stream[T] {
	if source == nil {
		return own
	}

	clone := *own
	clone.clock = source

	return &clone
}

// NewStreamOf 通过指定数据创建一个新的数据流
//   - 适合小规模数据
func NewStreamOf[T any](values ...T) Stream[T] {
	return NewStreamOfContext(context.Background(), values...)
}

// NewStreamOfContext 通过指定数据创建一个绑定取消信号的新数据流
func NewStreamOfContext[T any](ctx context.Context, values ...T) Stream[T] {
	if ctx == nil {
		ctx = context.Background()
	}

	output := make(chan Item[T], getCacheSize())
	own := &stream[T]{source: output, ctx: ctx, clock: timex.NewSystemClock()}
	go func() {
		defer close(output)

		var index uint64
		for _, value := range values {
			item := NewItem(value)
			item.SetIndex(index)
			index++

			if !send(ctx, output, item) {
				return
			}
		}
	}()

	return own
}

// NewStreamBy 通过指定通道创建一个新的数据流
//   - 指定通道由调用方负责关闭，关闭后数据流随之结束
func NewStreamBy[T any](source <-chan T) Stream[T] {
	return NewStreamByContext(context.Background(), source)
}

// NewStreamByContext 通过指定通道创建一个绑定取消信号的新数据流
func NewStreamByContext[T any](ctx context.Context, source <-chan T) Stream[T] {
	if ctx == nil {
		ctx = context.Background()
	}

	output := make(chan Item[T], getCacheSize())
	own := &stream[T]{source: output, ctx: ctx, clock: timex.NewSystemClock()}
	go func() {
		defer close(output)

		var index uint64
		for {
			select {
			case <-ctx.Done():
				return
			case value, ok := <-source:
				if !ok {
					return
				}

				item := NewItem(value)
				item.SetIndex(index)
				index++

				if !send(ctx, output, item) {
					return
				}
			}
		}
	}()

	return own
}

// GetChannel 获取数据流管道
func (own *stream[T]) GetChannel() <-chan Item[T] {
	return own.source
}

// Read 读取单个元素
//   - 流结束或取消时返回 false
func (own *stream[T]) Read() (Item[T], bool) {
	return recv(own.ctx, own.source)
}

// Wait 阻塞直到流结束或取消
func (own *stream[T]) Wait() {
	drain(own.ctx, own.source)
}

// Consume 消费
//   - consume: 消费函数
//   - async: 是否异步执行
func (own *stream[T]) Consume(consume func(item Item[T]), async ...bool) {
	if consume == nil {
		return
	}

	ctx := own.ctx
	consumeAll := func() {
		for {
			item, ok := recv(ctx, own.source)
			if !ok {
				return
			}

			consume(item)
		}
	}

	if len(async) > 0 && async[0] {
		go consumeAll()
		return
	}

	consumeAll()
}

// wrap 用新的输出通道包装当前流，继承取消信号与时间源
func (own *stream[T]) wrap(output chan Item[T]) *stream[T] {
	return &stream[T]{source: output, ctx: own.ctx, clock: own.clock}
}

// newEmptyStream 新建一个已结束的空流
func newEmptyStream[T any]() Stream[T] {
	empty := make(chan Item[T])
	close(empty)

	return &stream[T]{source: empty, ctx: context.Background(), clock: timex.NewSystemClock()}
}
