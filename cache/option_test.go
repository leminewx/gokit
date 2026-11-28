package cache

import (
	"testing"

	"gitee.com/leminewx/gokit/timex"
)

func TestApplyOptionsDefaults(t *testing.T) {
	t.Parallel()

	own := applyOptions[string, int](nil)
	if own.source == nil {
		t.Fatal("applyOptions 未填充默认时间源")
	}
	if own.onEvict != nil {
		t.Fatal("applyOptions 默认不应设置淘汰回调")
	}
}

func TestWithClock(t *testing.T) {
	t.Parallel()

	source := timex.NewManualAt()
	own := applyOptions[string, int]([]Option[string, int]{WithClock[string, int](source)})
	if own.source != source {
		t.Errorf("WithClock 未注入指定时间源")
	}

	own = applyOptions[string, int]([]Option[string, int]{WithClock[string, int](nil)})
	if own.source == nil {
		t.Errorf("WithClock(nil) 不应清空默认时间源")
	}
}

func TestWithOnEvict(t *testing.T) {
	t.Parallel()

	var gotKey string
	var gotValue int
	own := applyOptions[string, int]([]Option[string, int]{
		WithOnEvict[string, int](func(key string, value int) {
			gotKey, gotValue = key, value
		}),
	})

	if own.onEvict == nil {
		t.Fatal("WithOnEvict 未注入淘汰回调")
	}
	own.onEvict("k", 7)
	if gotKey != "k" || gotValue != 7 {
		t.Errorf("淘汰回调收到 (%q, %d), want (\"k\", 7)", gotKey, gotValue)
	}

	own = applyOptions[string, int]([]Option[string, int]{WithOnEvict[string, int](nil)})
	if own.onEvict != nil {
		t.Errorf("WithOnEvict(nil) 不应设置回调")
	}
}
