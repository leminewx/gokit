package queue

import (
	"testing"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

func TestNewDelayQueueDefaults(t *testing.T) {
	own := NewDelayQueue(100 * time.Millisecond)

	if own.clock == nil {
		t.Fatal("默认应使用系统时钟")
	}
	if own.handleError == nil {
		t.Fatal("默认应设置 panic 处理函数")
	}
	if got := cap(own.taskChan); got != 1024 {
		t.Fatalf("默认通道容量 = %d, want 1024", got)
	}
}

func TestWithClock(t *testing.T) {
	source := timex.NewManual(time.Unix(1700000000, 0))
	own := NewDelayQueue(100*time.Millisecond, WithClock(source))

	if own.clock != source {
		t.Fatal("WithClock 未替换时间源")
	}

	// nil 时间源应被忽略，保留原值
	origin := own.clock
	WithClock(nil)(own)
	if own.clock != origin {
		t.Fatal("nil 时间源不应覆盖原值")
	}
}

func TestWithErrorHandler(t *testing.T) {
	var called bool
	handler := func(taskID int64, err error) { called = true }

	own := NewDelayQueue(100*time.Millisecond, WithErrorHandler(handler))

	own.handleError(1, nil)
	if !called {
		t.Fatal("WithErrorHandler 未生效")
	}

	// nil 处理函数应被忽略，保留默认处理函数
	WithErrorHandler(nil)(own)
	if own.handleError == nil {
		t.Fatal("nil 处理函数不应清空默认处理函数")
	}
}

func TestWithChanSize(t *testing.T) {
	own := NewDelayQueue(100*time.Millisecond, WithChanSize(8))

	if got := cap(own.taskChan); got != 8 {
		t.Fatalf("通道容量 = %d, want 8", got)
	}

	// 非正容量应被忽略
	WithChanSize(0)(own)
	if got := cap(own.taskChan); got != 8 {
		t.Fatalf("非正容量不应覆盖原值: %d", got)
	}
}
