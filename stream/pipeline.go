package stream

import (
	"context"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

// send 向下游投递元素
//   - 流水线被取消时返回 false，调用方应结束当前 goroutine
func send[T any](ctx context.Context, out chan Item[T], item Item[T]) bool {
	select {
	case out <- item:
		return true
	case <-ctx.Done():
		return false
	}
}

// recv 从上游接收元素
//   - 上游关闭或流水线被取消时返回 false，调用方应结束当前 goroutine
func recv[T any](ctx context.Context, in <-chan Item[T]) (Item[T], bool) {
	select {
	case item, ok := <-in:
		return item, ok
	case <-ctx.Done():
		return nil, false
	}
}

// drain 排空上游
//   - 用于 Head 等提前终止的算子，避免上游 goroutine 阻塞在投递上
func drain[T any](ctx context.Context, in <-chan Item[T]) {
	for {
		if _, ok := recv(ctx, in); !ok {
			return
		}
	}
}

// after 返回一个在 timeout 之后触发一次的通道，以及停止计时的函数
//   - timeout <= 0 时立即触发，避免系统时钟不接受非正间隔
func after(source timex.Clock, timeout time.Duration) (<-chan time.Time, func()) {
	if timeout <= 0 {
		fired := make(chan time.Time, 1)
		fired <- time.Now()

		return fired, func() {}
	}

	ticker := source.NewTicker(timeout)

	return ticker.GetChannel(), ticker.Stop
}

// runCallback 执行算子的收尾回调
func runCallback(callback []func()) {
	if len(callback) > 0 && callback[0] != nil {
		callback[0]()
	}
}

// pipelineContext 提取流的取消信号，非内置实现回退到 Background
func pipelineContext[T any](s Stream[T]) context.Context {
	if internal, ok := s.(*stream[T]); ok && internal.ctx != nil {
		return internal.ctx
	}

	return context.Background()
}

// pipelineClock 提取流的时间源，非内置实现回退到系统时钟
func pipelineClock[T any](s Stream[T]) timex.Clock {
	if internal, ok := s.(*stream[T]); ok && internal.clock != nil {
		return internal.clock
	}

	return timex.NewSystemClock()
}
