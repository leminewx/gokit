package speedometer

import (
	"sync"
	"testing"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

const (
	_TEST_WINDOW  = time.Second
	_TEST_BUCKETS = 5
)

// testStart 测试时钟的起点，避免使用零值时间引发边界歧义
var testStart = time.Unix(1700000000, 0).UTC()

// newTestSpeedometer 新建一个由手动时钟驱动的计速器
func newTestSpeedometer() (*defaultSpeedometer, *timex.Manual) {
	fake := timex.NewManual(testStart)
	own := New(_TEST_WINDOW, WithClock(fake), WithBucketNumber(_TEST_BUCKETS))

	return own.(*defaultSpeedometer), fake
}

// TestSpeedometer_CountingWindow 验证窗口内的事件累计与速率换算
func TestSpeedometer_CountingWindow(t *testing.T) {
	t.Parallel()

	own, fake := newTestSpeedometer()

	if own.actualWindow != _TEST_WINDOW {
		t.Fatalf("实际窗口长度错误: 期望 %v, 实际 %v", _TEST_WINDOW, own.actualWindow)
	}
	if own.bucketSize != _TEST_WINDOW/_TEST_BUCKETS {
		t.Fatalf("桶长度错误: 期望 %v, 实际 %v", _TEST_WINDOW/_TEST_BUCKETS, own.bucketSize)
	}

	for range 10 {
		own.Add(10)
	}
	if speed := own.GetSpeed(); speed != 100 {
		t.Fatalf("窗口内速率错误: 期望 100, 实际 %v", speed)
	}

	// 窗口未过期前速率保持不变
	fake.Advance(500 * time.Millisecond)
	if speed := own.GetSpeed(); speed != 100 {
		t.Fatalf("未过期时速率不应变化: 期望 100, 实际 %v", speed)
	}

	// 追加事件后速率随之上升
	own.Add(50)
	if speed := own.GetSpeed(); speed != 150 {
		t.Fatalf("追加事件后速率错误: 期望 150, 实际 %v", speed)
	}
}

// TestSpeedometer_PartialExpiry 验证分桶淘汰：只有过期桶的事件被扣除
func TestSpeedometer_PartialExpiry(t *testing.T) {
	t.Parallel()

	own, fake := newTestSpeedometer()

	// 第一个桶：t0 时刻写入 100
	own.Add(100)

	// 第二个桶：t0+200ms 时刻写入 50
	fake.Advance(own.bucketSize)
	own.Add(50)

	if speed := own.GetSpeed(); speed != 150 {
		t.Fatalf("两桶累计错误: 期望 150, 实际 %v", speed)
	}

	// 窗口未占满前，两个桶的事件都在统计范围内
	for step := 2; step <= 4; step++ {
		fake.Advance(own.bucketSize)
		if speed := own.GetSpeed(); speed != 150 {
			t.Fatalf("第 %d 个桶时速率错误: 期望 150, 实际 %v", step, speed)
		}
	}

	// 推进到 t0+1s，第一个桶（t0 写入的 100）过期，只剩第二个桶的 50
	fake.Advance(own.bucketSize)
	if speed := own.GetSpeed(); speed != 50 {
		t.Fatalf("部分淘汰后速率错误: 期望 50, 实际 %v", speed)
	}

	// 再推进一个桶，第二个桶也已过期
	fake.Advance(own.bucketSize)
	if speed := own.GetSpeed(); speed != 0 {
		t.Fatalf("全部淘汰后速率错误: 期望 0, 实际 %v", speed)
	}
}

// TestSpeedometer_StaleBuckets 验证跨越多桶的更新会一次性淘汰所有过期数据
//   - 移动步数 >= 桶数量时直接清空窗口，接口快速路径与分桶淘汰结果一致
func TestSpeedometer_StaleBuckets(t *testing.T) {
	t.Parallel()

	own, fake := newTestSpeedometer()

	own.Add(250)
	if speed := own.GetSpeed(); speed != 250 {
		t.Fatalf("初始速率错误: 期望 250, 实际 %v", speed)
	}

	// 推进超过一个完整窗口，无论走清空分支还是逐桶淘汰，速率都应为 0
	fake.Advance(2 * _TEST_WINDOW)
	if speed := own.GetSpeed(); speed != 0 {
		t.Fatalf("窗口整体过期后速率错误: 期望 0, 实际 %v", speed)
	}
	if own.currentTotal != 0 {
		t.Fatalf("窗口整体过期后计数未清零: %d", own.currentTotal)
	}

	// 清空后重新计数，不应残留旧桶数据
	own.Add(30)
	if speed := own.GetSpeed(); speed != 30 {
		t.Fatalf("清空后重新计数错误: 期望 30, 实际 %v", speed)
	}
}

// TestSpeedometer_TimeRollback 验证时间回拨时安全降级为清空窗口
func TestSpeedometer_TimeRollback(t *testing.T) {
	t.Parallel()

	fake := newStubClock(testStart)
	own := New(_TEST_WINDOW, WithClock(fake), WithBucketNumber(_TEST_BUCKETS)).(*defaultSpeedometer)

	own.Add(80)
	if speed := own.GetSpeed(); speed != 80 {
		t.Fatalf("初始速率错误: 期望 80, 实际 %v", speed)
	}

	// 时钟回拨到构造之前
	fake.Set(testStart.Add(-time.Second))
	if speed := own.GetSpeed(); speed != 0 {
		t.Fatalf("时间回拨后速率错误: 期望 0, 实际 %v", speed)
	}
	if own.lastTime != testStart.Add(-time.Second) {
		t.Fatalf("时间回拨后基准时间未重置: %v", own.lastTime)
	}
}

// TestSpeedometer_Defaults 验证缺省值与非法选项的兜底行为
func TestSpeedometer_Defaults(t *testing.T) {
	t.Parallel()

	for _, window := range []time.Duration{0, -time.Second} {
		own := New(window, WithClock(nil), WithBucketNumber(0)).(*defaultSpeedometer)

		if own.windowSize != 2*time.Second {
			t.Fatalf("缺省窗口错误: window=%v, 实际 %v", window, own.windowSize)
		}
		if own.bucketNumber != _DEFAULT_BUCKET_NUMBER {
			t.Fatalf("缺省桶数量错误: 期望 %d, 实际 %d", _DEFAULT_BUCKET_NUMBER, own.bucketNumber)
		}
		if own.bucketSize <= 0 || own.actualWindow <= 0 {
			t.Fatalf("桶长度或实际窗口非法: bucketSize=%v, actualWindow=%v", own.bucketSize, own.actualWindow)
		}
		if own.clock == nil {
			t.Fatal("缺省时钟不应为 nil")
		}
	}

	// 窗口小于桶数量时桶长度被兜底为 1ns，实际窗口随之放大
	own := New(time.Nanosecond, WithBucketNumber(_TEST_BUCKETS)).(*defaultSpeedometer)
	if own.bucketSize != 1 {
		t.Fatalf("极小窗口的桶长度兜底错误: 期望 1ns, 实际 %v", own.bucketSize)
	}
	if own.actualWindow != time.Duration(_TEST_BUCKETS) {
		t.Fatalf("极小窗口的实际窗口错误: 期望 %v, 实际 %v", time.Duration(_TEST_BUCKETS), own.actualWindow)
	}
}

// TestSpeedometer_Concurrent 验证并发读写同一计速器不产生数据竞争
func TestSpeedometer_Concurrent(t *testing.T) {
	t.Parallel()

	own := New(_TEST_WINDOW, WithBucketNumber(_TEST_BUCKETS)).(*defaultSpeedometer)

	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			for range 100 {
				own.Add(1)
				_ = own.GetSpeed()
			}
		})
	}
	group.Wait()

	if speed := own.GetSpeed(); speed <= 0 {
		t.Fatalf("并发写入后速率应大于 0, 实际 %v", speed)
	}
}

// stubClock 仅提供当前时刻的可控时钟，用于模拟时间回拨
type stubClock struct {
	lock sync.Mutex
	now  time.Time
}

func newStubClock(start time.Time) *stubClock {
	return &stubClock{now: start}
}

func (own *stubClock) GetNow() time.Time {
	own.lock.Lock()
	defer own.lock.Unlock()

	return own.now
}

func (own *stubClock) NewTicker(time.Duration) timex.Ticker {
	return stubTicker{}
}

func (own *stubClock) Set(now time.Time) {
	own.lock.Lock()
	own.now = now
	own.lock.Unlock()
}

// stubTicker 永不触发的嘀嗒器
type stubTicker struct{}

func (stubTicker) GetChannel() <-chan time.Time { return nil }

func (stubTicker) Stop() {}
