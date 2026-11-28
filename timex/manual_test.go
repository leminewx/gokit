package timex

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestManualAdvance 验证手动时钟按间隔补发嘀嗒
func TestManualAdvance(t *testing.T) {
	t.Parallel()

	source := NewManualAt()
	ticker := source.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	source.Advance(25 * time.Millisecond)

	var ticks int
	for {
		select {
		case <-ticker.GetChannel():
			ticks++
			continue
		default:
		}

		break
	}

	if ticks != 2 {
		t.Fatalf("推进 25ms 应补发 2 个嘀嗒, 实际 %d", ticks)
	}

	if got, want := source.GetNow().UnixNano(), time.Unix(0, 25*int64(time.Millisecond)).UnixNano(); got != want {
		t.Fatalf("GetNow() = %d, want %d", got, want)
	}
}

// TestManualAdvanceTo 验证 AdvanceTo 只在目标时刻晚于当前时刻时推进
func TestManualAdvanceTo(t *testing.T) {
	t.Parallel()

	start := time.Unix(1700000000, 0)
	source := NewManual(start)

	source.AdvanceTo(start.Add(-time.Second))
	if !source.GetNow().Equal(start) {
		t.Fatalf("向过去推进不应改变当前时刻: %v", source.GetNow())
	}

	source.AdvanceTo(start.Add(time.Minute))
	if got := source.GetNow().Sub(start); got != time.Minute {
		t.Fatalf("推进差值 = %v, want 1m", got)
	}
}

// TestManualStop 验证停止后的嘀嗒器不再收到事件
func TestManualStop(t *testing.T) {
	t.Parallel()

	source := NewManualAt()
	ticker := source.NewTicker(time.Millisecond)
	ticker.Stop()
	ticker.Stop() // 重复停止安全

	source.Advance(10 * time.Millisecond)

	select {
	case <-ticker.GetChannel():
		t.Fatal("已停止的嘀嗒器不应再收到事件")
	default:
	}
}

// TestManualAdvanceCap 验证一次推进过长时间时嘀嗒数量有上限
func TestManualAdvanceCap(t *testing.T) {
	t.Parallel()

	source := NewManualAt()
	ticker := source.NewTicker(time.Nanosecond)
	defer ticker.Stop()

	source.Advance(time.Second)

	var ticks int
	for {
		select {
		case <-ticker.GetChannel():
			ticks++
			continue
		default:
		}

		break
	}

	if ticks > _MAX_TICKS_PER_ADVANCE {
		t.Fatalf("补发嘀嗒数 %d 超过上限 %d", ticks, _MAX_TICKS_PER_ADVANCE)
	}
}

// TestManualConcurrent 验证并发推进与新建嘀嗒器不会产生数据竞争
func TestManualConcurrent(t *testing.T) {
	t.Parallel()

	source := NewManualAt()

	var wg sync.WaitGroup
	var received atomic.Int64

	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 100 {
			source.Advance(time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()
		ticker := source.NewTicker(time.Millisecond)
		go func() {
			for range ticker.GetChannel() {
				received.Add(1)
			}
		}()

		time.Sleep(20 * time.Millisecond)
		ticker.Stop()
	}()

	wg.Wait()
	if source.GetNow().UnixNano() == 0 {
		t.Error("并发推进后时钟应已前进")
	}
}
