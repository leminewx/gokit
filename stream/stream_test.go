package stream

import (
	"context"

	"gitee.com/leminewx/gokit/timex"
)

// testSource 构建一个直接对接算子的可控数据源
//   - 返回的通道为无缓冲通道，发送操作与算子消费严格同步，便于确定性测试
func testSource[T any](ctx context.Context, source timex.Clock) (chan Item[T], Stream[T]) {
	items := make(chan Item[T])

	return items, &stream[T]{source: items, ctx: ctx, clock: source}
}
