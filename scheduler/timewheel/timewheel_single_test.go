package timewheel

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

// newTestClock 新建从固定时刻开始的手动时钟，避免测试依赖墙钟
func newTestClock() *timex.Manual {
	return timex.NewManual(time.Unix(1700000000, 0))
}

// waitUntil 轮询等待条件成立
func waitUntil(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for {
		if cond() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Millisecond)
	}
}

// advanceUntil 按 step 推进假时钟，直到条件成立或真实时间超时
//   - 真实等待仅用于让时间轮事件循环消费嘀嗒与新增任务
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

// TestSingleWheel_OneShot 验证一次性任务在预定时刻后执行
func TestSingleWheel_OneShot(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	wheel := NewSingleWheel(_TEST_TICK, 50, WithClock(fake))
	wheel.Start()
	defer wheel.Stop()

	var executed atomic.Bool
	wheel.AddTask(fake.GetNow().Add(50*time.Millisecond), func() { executed.Store(true) }, 0)

	// 推进到预定时刻之前，任务不应执行
	fake.Advance(40 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if executed.Load() {
		t.Fatal("任务在预定时刻之前被执行")
	}

	if !advanceUntil(fake, _TEST_TICK, executed.Load) {
		t.Fatal("一次性任务未在预定时刻后执行")
	}
}

// TestSingleWheel_Periodic 验证周期任务按固定周期重复执行指定次数
func TestSingleWheel_Periodic(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	wheel := NewSingleWheel(_TEST_TICK, 100, WithClock(fake))
	wheel.Start()
	defer wheel.Stop()

	var count atomic.Int32
	wheel.AddTask(fake.GetNow().Add(20*time.Millisecond), func() { count.Add(1) }, 30*time.Millisecond, 4)

	if !advanceUntil(fake, _TEST_TICK, func() bool { return count.Load() >= 4 }) {
		t.Fatalf("周期任务执行次数 = %d, want 4", count.Load())
	}

	// 再推进一段时间，重复次数用尽后不应继续执行
	fake.Advance(200 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if got := count.Load(); got != 4 {
		t.Fatalf("周期任务执行次数 = %d, want 4", got)
	}
}

// TestSingleWheel_TaskBeyondOneRound 验证延迟超过一整圈的任务在正确时刻执行
//   - 一圈 = 10ms * 5 = 50ms，任务延迟 120ms 需要跨越多圈后命中
func TestSingleWheel_TaskBeyondOneRound(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	wheel := NewSingleWheel(_TEST_TICK, 5, WithClock(fake))
	wheel.Start()
	defer wheel.Stop()

	var executed atomic.Bool
	wheel.AddTask(fake.GetNow().Add(120*time.Millisecond), func() { executed.Store(true) }, 0)

	fake.Advance(60 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if executed.Load() {
		t.Fatal("任务在预定时刻之前被执行")
	}

	if !advanceUntil(fake, 60*time.Millisecond, executed.Load) {
		t.Fatal("任务未在预定时刻后执行")
	}
}

// TestSingleWheel_NilJob 验证 nil 任务被拒绝且不占用任务 ID
func TestSingleWheel_NilJob(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	wheel := NewSingleWheel(_TEST_TICK, 10, WithClock(fake))
	wheel.Start()
	defer wheel.Stop()

	if id := wheel.AddTask(fake.GetNow(), nil, 0); id != 0 {
		t.Fatalf("nil 任务返回 ID = %d, want 0", id)
	}
}

// TestSingleWheel_CancelTaskWithoutCallback 验证不传取消回调时也能取消任务
func TestSingleWheel_CancelTaskWithoutCallback(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	wheel := NewSingleWheel(_TEST_TICK, 10, WithClock(fake))
	wheel.Start()
	defer wheel.Stop()

	var executed atomic.Bool
	id := wheel.AddTask(fake.GetNow().Add(100*time.Millisecond), func() { executed.Store(true) }, 0)

	// 任务可能仍在投递通道中，轮询直到取消成功
	if !waitUntil(2*time.Second, func() bool { return wheel.CancelTask(id, nil) }) {
		t.Fatal("任务未被成功取消")
	}

	fake.Advance(200 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if executed.Load() {
		t.Fatal("已取消的任务不应执行")
	}
}

// TestSingleWheel_PanicRecover 验证任务 panic 被错误处理器捕获且不终止时间轮
func TestSingleWheel_PanicRecover(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	type captured struct {
		taskID int64
		err    error
	}

	events := make(chan captured, 1)
	wheel := NewSingleWheel(_TEST_TICK, 10,
		WithClock(fake),
		WithErrorHandler(func(taskID int64, err error) {
			select {
			case events <- captured{taskID: taskID, err: err}:
			default:
			}
		}),
	)
	wheel.Start()
	defer wheel.Stop()

	var finished atomic.Bool
	wheel.AddTask(fake.GetNow(), func() { panic("test panic") }, 0)
	wheel.AddTask(fake.GetNow().Add(30*time.Millisecond), func() { finished.Store(true) }, 0)

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

// TestSingleWheel_AddTaskOutsideRunning 验证未启动与已停止时拒绝新任务
func TestSingleWheel_AddTaskOutsideRunning(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	tw := NewSingleWheel(_TEST_TICK, 100, WithClock(fake))

	// 未启动时不接受任务
	if id := tw.AddTask(fake.GetNow(), func() {}, 0); id != 0 {
		t.Fatalf("expected zero id before start, got %d", id)
	}

	tw.Start()
	tw.Stop()

	// 停止后不接受任务，且不能 panic
	if id := tw.AddTask(fake.GetNow(), func() {}, 0); id != 0 {
		t.Fatalf("expected zero id after stop, got %d", id)
	}
}

// TestSingleWheel_RepeatsExact 验证 repeats=N 精确执行 N 次而不是 N+1 次
func TestSingleWheel_RepeatsExact(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	tw := NewSingleWheel(_TEST_TICK, 100, WithClock(fake))
	tw.Start()
	defer tw.Stop()

	var count atomic.Int32
	tw.AddTask(fake.GetNow().Add(20*time.Millisecond), func() { count.Add(1) }, 30*time.Millisecond, 3)

	if !advanceUntil(fake, _TEST_TICK, func() bool { return count.Load() >= 3 }) {
		t.Fatalf("expected 3 executions, got %d", count.Load())
	}

	fake.Advance(200 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if got := count.Load(); got != 3 {
		t.Fatalf("expected 3 executions, got %d", got)
	}
}

// TestSingleWheel_CancelTask 验证取消成功并触发取消回调，且任务不再执行
func TestSingleWheel_CancelTask(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	tw := NewSingleWheel(_TEST_TICK, 100, WithClock(fake))
	tw.Start()
	defer tw.Stop()

	var ran, canceled atomic.Bool
	id := tw.AddTask(fake.GetNow().Add(100*time.Millisecond), func() { ran.Store(true) }, 0)

	// 任务可能仍在投递通道中，轮询直到登记到 taskMap 并取消成功
	if !waitUntil(2*time.Second, func() bool {
		return tw.CancelTask(id, func() { canceled.Store(true) })
	}) {
		t.Fatal("expected CancelTask to return true")
	}
	if !canceled.Load() {
		t.Fatal("expected cancel callback to be invoked")
	}

	fake.Advance(300 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if ran.Load() {
		t.Fatal("canceled task should not run")
	}
}

// TestSingleWheel_ConcurrentStop 验证并发停止不会 panic
func TestSingleWheel_ConcurrentStop(t *testing.T) {
	t.Parallel()

	tw := NewSingleWheel(_TEST_TICK, 100)
	tw.Start()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tw.Stop()
		}()
	}

	wg.Wait()
}

// TestSingleWheel_RepeatTaskWithPastStart 验证首轮时刻已过的重复任务仍按周期重复执行
//   - 回归点：逾期任务曾只执行一次且永久留在 taskMap 中
func TestSingleWheel_RepeatTaskWithPastStart(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	tw := NewSingleWheel(_TEST_TICK, 100, WithClock(fake))
	tw.Start()
	defer tw.Stop()

	var count atomic.Int32
	id := tw.AddTask(fake.GetNow().Add(-time.Second), func() { count.Add(1) }, 20*time.Millisecond, 3)
	if id == 0 {
		t.Fatal("expected non-zero task id")
	}

	if !advanceUntil(fake, _TEST_TICK, func() bool { return count.Load() >= 3 }) {
		t.Fatalf("expected 3 executions, got %d", count.Load())
	}

	fake.Advance(200 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if got := count.Load(); got != 3 {
		t.Fatalf("expected exactly 3 executions, got %d", got)
	}

	if _, loaded := tw.taskMap.Load(id); loaded {
		t.Fatal("finished task should be removed from taskMap")
	}
}

// TestSingleWheel_FullCircleRepeatTasksDoNotHangTick 验证同槽内多个"整圈周期"任务不会让 tick 死循环
//   - 回归点：tick 曾边遍历当前槽边把重新调度的任务回插到同一链表，
//     周期为整圈整数倍时回插位置与当前槽重合，e = e.Next() 会反复走到新插入的节点，
//     导致 tick 永不返回（事件循环停摆、Stop 阻塞、回调被反复触发）
func TestSingleWheel_FullCircleRepeatTasksDoNotHangTick(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	const slotNumber = 4
	period := slotNumber * _TEST_TICK // 恰好一整圈
	tw := NewSingleWheel(_TEST_TICK, slotNumber, WithClock(fake))

	var count atomic.Int32
	for range 2 {
		task := &scheduledTask{
			id:        tw.taskIDGenerator.Add(1),
			job:       func() { count.Add(1) },
			executeAt: fake.GetNow().Add(period).UnixNano(),
			period:    period,
			repeats:   -1,
		}
		tw.slots[0].PushBack(task)
		tw.taskMap.Store(task.id, task)
	}

	// 两个任务同时到期：执行后各自重新调度回当前槽 0
	fake.Advance(period)

	done := make(chan struct{})
	go func() {
		defer close(done)
		tw.tick()
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("tick() 未返回：回插到正在遍历的槽导致死循环")
	}

	if !waitUntil(2*time.Second, func() bool { return count.Load() == 2 }) {
		t.Fatalf("到期任务执行次数 = %d, want 2", count.Load())
	}

	// 回插后应各自落回当前槽，且指针前移一格
	if got := tw.slots[0].Len(); got != 2 {
		t.Fatalf("槽 0 任务数 = %d, want 2", got)
	}
	if got := tw.currentPosition; got != 1 {
		t.Fatalf("currentPosition = %d, want 1", got)
	}

	// 继续转动剩余槽位，任务尚未到期，不应被重复执行
	for i := 0; i < slotNumber-1; i++ {
		tw.tick()
	}
	time.Sleep(5 * time.Millisecond)
	if got := count.Load(); got != 2 {
		t.Fatalf("未到期任务被提前执行，执行次数 = %d, want 2", got)
	}
}

// TestSingleWheel_StartAfterStopIgnored 验证停止后再次 Start 不会让时间轮"假运行"
func TestSingleWheel_StartAfterStopIgnored(t *testing.T) {
	t.Parallel()

	fake := newTestClock()
	tw := NewSingleWheel(_TEST_TICK, 10, WithClock(fake))
	tw.Start()
	tw.Stop()

	// 重新启动应为无操作
	tw.Start()

	if tw.IsRunning() {
		t.Fatal("wheel should stay stopped after Stop")
	}
	if id := tw.AddTask(fake.GetNow(), func() {}, 0); id != 0 {
		t.Fatalf("expected zero id after restart attempt, got %d", id)
	}
}
