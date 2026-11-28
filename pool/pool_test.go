package pool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_BasicSubmit(t *testing.T) {
	p := New(WithWorkers(4))
	var counter int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		_ = p.Submit(func(_ context.Context) error {
			atomic.AddInt64(&counter, 1)
			wg.Done()
			return nil
		})
	}
	wg.Wait()
	if counter != 100 {
		t.Fatalf("expected 100, got %d", counter)
	}
	p.Release()
}

func TestPool_ConcurrencySafety(t *testing.T) {
	p := New(WithWorkers(8))
	var m sync.Map
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		idx := i
		_ = p.Submit(func(_ context.Context) error {
			m.Store(idx, idx)
			wg.Done()
			return nil
		})
	}
	wg.Wait()
	p.Release()
	count := 0
	m.Range(func(_, _ any) bool { count++; return true })
	if count != 1000 {
		t.Fatalf("expected 1000 entries, got %d", count)
	}
}

func TestPool_SubmitAfterClose(t *testing.T) {
	p := New(WithWorkers(2))
	p.Release()
	if err := p.Submit(func(_ context.Context) error { return nil }); err == nil {
		t.Fatal("expected error when submitting to closed pool")
	}
}

func TestPool_TrySubmit(t *testing.T) {
	p := New(WithWorkers(1), WithQueueSize(2))

	// 占用唯一的工作 goroutine，使其进入运行态（hold.Wait 阻塞）
	workerStarted := make(chan struct{})
	var hold sync.WaitGroup
	hold.Add(1)
	_ = p.Submit(func(_ context.Context) error {
		close(workerStarted)
		hold.Wait()
		return nil
	})
	<-workerStarted // 确保 worker 正在执行 hold 任务，任务已占用（不在缓冲）

	// 缓冲容量为 2，可成功排队两个任务
	if !p.TrySubmit(func(_ context.Context) error { return nil }) {
		t.Fatal("first TrySubmit (queued) should succeed")
	}
	if !p.TrySubmit(func(_ context.Context) error { return nil }) {
		t.Fatal("second TrySubmit (queued) should succeed")
	}
	// 缓冲已满 → 失败
	if p.TrySubmit(func(_ context.Context) error { return nil }) {
		t.Fatal("third TrySubmit should fail (queue full)")
	}

	hold.Done()
	p.Release()
}

func TestPool_PanicRecovery(t *testing.T) {
	var recovered atomic.Bool
	p := New(WithWorkers(2), WithPanicHandler(func(_ any, _ []byte) {
		recovered.Store(true)
	}))
	var done atomic.Bool
	_ = p.Submit(func(_ context.Context) error {
		defer func() { done.Store(true) }()
		panic("boom")
	})
	// 等待 panic 处理
	for i := 0; i < 100; i++ {
		if recovered.Load() && done.Load() {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !recovered.Load() {
		t.Fatal("panic handler was not called")
	}
	p.Release()
}

func TestPool_CloseReleases(t *testing.T) {
	p := New(WithWorkers(2))
	// 提交一个会正常完成的任务
	done := make(chan struct{}, 1)
	_ = p.Submit(func(_ context.Context) error {
		done <- struct{}{}
		return nil
	})
	<-done
	p.Close()
	// 关闭后不应泄漏 goroutine（无 panic、无死锁）
}

func BenchmarkPool_Submit(b *testing.B) {
	p := New(WithWorkers(4))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Submit(func(_ context.Context) error { return nil })
	}
	p.Release()
}

func TestPool_ConcurrentSubmitAndRelease(t *testing.T) {
	p := New(WithWorkers(4), WithQueueSize(4))

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 与 Release 并发提交，不能出现 send on closed channel
			_ = p.Submit(func(context.Context) error { return nil })
		}()
	}

	p.Release()
	wg.Wait()

	if err := p.Submit(func(context.Context) error { return nil }); err == nil {
		t.Fatal("expected error when submitting to a released pool")
	}
	if got := p.GetWorkers(); got != 4 {
		t.Fatalf("expected 4 workers, got %d", got)
	}
}

// TestPool_ReleaseUnblocksSubmit 验证关闭时：
//   - 阻塞在满队列上的 Submit 能及时返回；
//   - 任务上下文被取消，监听 ctx 的任务能正常退出，Release 不会无限阻塞
func TestPool_ReleaseUnblocksSubmit(t *testing.T) {
	started := make(chan struct{})
	p := New(WithWorkers(1), WithQueueSize(0))

	// 占住唯一的 worker，并在上下文取消后退出
	if err := p.Submit(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	<-started

	// 队列无缓冲，该提交会阻塞等待 worker
	done := make(chan error, 1)
	go func() {
		done <- p.Submit(func(context.Context) error { return nil })
	}()

	p.Release()

	// Release 返回后阻塞的 Submit 已被唤醒，但结果写入 channel 仍需一次调度，
	// 因此不能用 default 立即判定，而应等待一个上限时间
	select {
	case err := <-done:
		// 任务被接走时返回 nil，被取消时返回 ErrClosing/ErrClosed，两者均属正常
		if err != nil && !errors.Is(err, ErrClosing) && !errors.Is(err, ErrClosed) {
			t.Fatalf("unexpected error from blocked submit: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("blocked Submit should return during Release")
	}

	if err := p.Submit(func(context.Context) error { return nil }); !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed after release, got %v", err)
	}
}
