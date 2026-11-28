package timewheel

import (
	"testing"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

func TestNewSingleWheelDefaults(t *testing.T) {
	own := NewSingleWheel(100*time.Millisecond, 60)

	if own.clock == nil {
		t.Fatal("默认应使用系统时钟")
	}
	if own.handleError == nil {
		t.Fatal("默认应设置 panic 处理函数")
	}
}

func TestWithClock(t *testing.T) {
	source := timex.NewManual(time.Unix(1700000000, 0))
	own := NewSingleWheel(100*time.Millisecond, 60, WithClock(source))

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

	own := NewSingleWheel(100*time.Millisecond, 60, WithErrorHandler(handler))

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
