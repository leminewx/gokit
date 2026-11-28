package stream

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestContextCancelWait 验证取消信号能让 Wait 立即返回
func TestContextCancelWait(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := make(chan int)
	s := NewStreamByContext(ctx, source).
		Group(func(item Item[int]) string { return "all" }).
		Filter(func(item Item[int]) bool { return true }).
		Peek(func(item Item[int]) {})

	done := make(chan struct{})
	go func() {
		s.Wait()
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("取消后 Wait 未及时返回")
	}
}

// TestContextCancelConsume 验证同步消费在被取消后停止
func TestContextCancelConsume(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := make(chan int)
	s := NewStreamByContext(ctx, source)

	go func() {
		for index := 0; ; index++ {
			select {
			case source <- index:
			case <-ctx.Done():
				return
			}
		}
	}()

	var consumed int32
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		s.Consume(func(item Item[int]) {
			// 消费 5 条后触发取消，后续消费必须立即停止
			if atomic.AddInt32(&consumed, 1) == 5 {
				cancel()
			}
		})
	}()

	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("取消后 Consume 未及时返回")
	}

	if got := atomic.LoadInt32(&consumed); got < 5 {
		t.Fatalf("取消前应至少消费 5 条, 实际 %d", got)
	}
}

// TestContextCancelViaWithContext 验证中间算子通过 WithContext 接管取消信号
func TestContextCancelViaWithContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	values := make([]int, 100)
	for i := range values {
		values[i] = i
	}

	finished := make(chan struct{})
	go func() {
		defer close(finished)
		// 源头使用 Background，仅让下游算子感知取消
		NewStreamOf(values...).
			WithContext(ctx).
			Filter(func(item Item[int]) bool {
				if item.GetValue() == 10 {
					cancel()
				}
				return true
			}).
			Wait()
	}()

	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("WithContext 接管后未及时退出")
	}
}

// TestContextCancelStopsFileSource 验证文件数据源在取消后关闭 reader
func TestContextCancelStopsFileSource(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	reader := newCountingReader(1000)

	s := NewStreamFromLineFile(ctx, reader, func(line []byte, err error) (string, bool) {
		if err != nil {
			return "", false
		}

		return string(line), true
	})

	// 读一条后立即取消，reader 应被关闭
	if _, ok := s.Read(); !ok {
		t.Fatal("预期读取到第一行")
	}
	cancel()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-reader.closed:
			return
		case <-deadline:
			t.Fatal("取消后文件未被关闭")
		case <-time.After(5 * time.Millisecond):
		}
	}
}
