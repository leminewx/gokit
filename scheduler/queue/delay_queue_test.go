package queue

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

// _TEST_TICK 测试用嘀嗒间隔
const _TEST_TICK = 10 * time.Millisecond

// newTestClock 新建从固定时刻开始的手动时钟
func newTestClock() *timex.Manual {
	return timex.NewManual(time.Unix(1700000000, 0))
}

// advanceUntil 按 step 推进假时钟，直到条件成立或真实时间超时
func advanceUntil(fake *timex.Manual, step time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(2 * time.Second)
	for {
		if cond() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		fake.Advance(step)
		time.Sleep(time.Millisecond)
	}
}

// TestDelayQueue_AddTaskAfterStop 验证停止后投递安全返回 0
func TestDelayQueue_AddTaskAfterStop(t *testing.T) {
	t.Parallel()

	q := NewDelayQueue(_TEST_TICK)
	q.Start()
	q.Stop()

	// 停止后投递应安全返回 0，而不是向无效通道发送
	if id := q.AddTask(time.Now(), func() {}, 0); id != 0 {
		t.Fatalf("expected zero id after stop, got %d", id)
	}
}

// TestDelayQueue_CancelTaskNilCallback 验证取消不传回调时不 panic，且任务不会执行
func TestDelayQueue_CancelTaskNilCallback(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	q := NewDelayQueue(_TEST_TICK, WithClock(fake))
	q.Start()
	defer q.Stop()

	var ran atomic.Bool
	id := q.AddTask(fake.GetNow().Add(100*time.Millisecond), func() { ran.Store(true) }, 0)

	// 不传回调时不能 panic；AddTask 已同步登记，可立即取消
	if !q.CancelTask(id, nil) {
		t.Fatal("first cancel should return true")
	}
	if q.CancelTask(id, nil) {
		t.Fatal("second cancel should return false")
	}

	fake.Advance(200 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if ran.Load() {
		t.Fatal("canceled task should not run")
	}
}

// TestDelayQueue_CancelTaskMarksBeforeCallback 验证取消回调执行时任务已被标记为取消
//   - 回归点：CancelTask 曾先执行 onCancel 再置 canceled，回调内读到“未取消”，
//     与 timewheel.CancelTask 的顺序相反
func TestDelayQueue_CancelTaskMarksBeforeCallback(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	q := NewDelayQueue(_TEST_TICK, WithClock(fake))
	q.Start()
	defer q.Stop()

	id := q.AddTask(fake.GetNow().Add(100*time.Millisecond), func() {}, 0)

	var jobTask *task
	q.taskMap.Range(func(_, value any) bool {
		jobTask = value.(*task)
		return false
	})
	if jobTask == nil {
		t.Fatal("未能从 taskMap 中取到任务")
	}

	var observed bool
	if !q.CancelTask(id, func() { observed = jobTask.canceled.Load() }) {
		t.Fatal("取消应成功")
	}
	if !observed {
		t.Fatal("取消回调执行时任务应已被标记为取消")
	}
}

// TestDelayQueue_PeriodicRepeats 验证周期任务按指定次数执行
func TestDelayQueue_PeriodicRepeats(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	q := NewDelayQueue(_TEST_TICK, WithClock(fake))
	q.Start()
	defer q.Stop()

	var count atomic.Int32
	q.AddTask(fake.GetNow(), func() { count.Add(1) }, 30*time.Millisecond, 3)

	if !advanceUntil(fake, _TEST_TICK, func() bool { return count.Load() >= 3 }) {
		t.Fatalf("expected 3 executions, got %d", count.Load())
	}

	// 再推进一段时间，剩余次数用尽后不应继续执行
	fake.Advance(200 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if got := count.Load(); got != 3 {
		t.Fatalf("expected 3 executions, got %d", got)
	}
}

// TestDelayQueue_PanicRecover 验证任务 panic 被错误处理器捕获且队列继续工作
func TestDelayQueue_PanicRecover(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	type captured struct {
		taskID int64
		err    error
	}

	events := make(chan captured, 1)
	q := NewDelayQueue(_TEST_TICK,
		WithClock(fake),
		WithErrorHandler(func(taskID int64, err error) {
			select {
			case events <- captured{taskID: taskID, err: err}:
			default:
			}
		}),
	)
	q.Start()
	defer q.Stop()

	var finished atomic.Bool
	q.AddTask(fake.GetNow(), func() { panic("test panic") }, 0)
	q.AddTask(fake.GetNow().Add(30*time.Millisecond), func() { finished.Store(true) }, 0)

	if !advanceUntil(fake, _TEST_TICK, finished.Load) {
		t.Fatal("panic 任务之后的任务未执行")
	}

	select {
	case event := <-events:
		if !errors.Is(event.err, ErrTaskPanic) {
			t.Fatalf("错误未包装 ErrTaskPanic: %v", event.err)
		}
		if event.taskID == 0 {
			t.Fatal("错误处理器未收到任务 ID")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("任务 panic 未被错误处理器捕获")
	}
}

// TestDelayQueue_ConcurrentAddAndStop 验证并发投递与停止不会出现 send on closed channel
func TestDelayQueue_ConcurrentAddAndStop(t *testing.T) {
	t.Parallel()

	q := NewDelayQueue(5*time.Millisecond, WithChanSize(8))
	q.Start()

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = q.AddTask(time.Now(), func() {}, 0)
		}()
	}

	q.Stop()
	wg.Wait()
}

// TestDelayQueue_StartAfterStopIgnored 验证停止后再次 Start 不会让队列"假运行"
func TestDelayQueue_StartAfterStopIgnored(t *testing.T) {
	t.Parallel()

	q := NewDelayQueue(_TEST_TICK)
	q.Start()
	q.Stop()

	// 重新启动应为无操作
	q.Start()

	if q.IsRunning() {
		t.Fatal("queue should stay stopped after Stop")
	}
	if id := q.AddTask(time.Now(), func() {}, 0); id != 0 {
		t.Fatalf("expected zero id after restart attempt, got %d", id)
	}
}
