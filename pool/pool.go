// Package pool 提供简单、安全、高并发且无第三方依赖的 goroutine 池
//
// 设计目标：
//   - 安全：统一的 panic 恢复，不会因单个任务 panic 而崩溃池
//   - 无依赖：仅使用标准库
//   - 高并发：固定数量的工作 goroutine + 无锁任务队列（channel）
//   - 优雅关闭：Close/Release 等待所有任务完成，不会泄漏 goroutine
package pool

import (
	"context"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

type (
	// Pool 代表一个 goroutine 池
	Pool struct {
		workers      int
		tasks        chan Task
		ctx          context.Context
		cancel       context.CancelFunc
		wg           sync.WaitGroup
		submits      sync.WaitGroup // 记录正在投递的任务，关闭前必须等它们退出
		closeOnce    sync.Once
		submitLock   sync.RWMutex // 保证提交与关闭互斥，避免向已关闭通道发送
		closed       atomic.Bool
		panicHandler PanicHandler
	}

	// Task 表示一个可提交给池执行的任务
	Task func(ctx context.Context) error

	// PanicHandler 在任务发生 panic 时被调用，可用于上报日志
	PanicHandler func(recv any, stack []byte)
)

// New 创建一个新的 goroutine 池并立即启动工作 goroutine
func New(opts ...Option) *Pool {
	cfg := &config{workers: 0, queueSize: 0}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.workers <= 0 {
		cfg.workers = 4
	}

	ctx, cancel := context.WithCancel(context.Background())
	own := &Pool{
		workers:      cfg.workers,
		tasks:        make(chan Task, cfg.queueSize),
		ctx:          ctx,
		cancel:       cancel,
		panicHandler: cfg.panicHandler,
	}

	own.wg.Add(cfg.workers)
	for i := 0; i < cfg.workers; i++ {
		go own.loop()
	}

	return own
}

// Submit 提交一个任务到池中执行
// 当池已关闭时返回错误；当队列满时阻塞，直到有空位或池进入关闭流程
func (own *Pool) Submit(task Task) error {
	// 登记投递中的调用，Release 只有在它们全部退出后才会关闭通道
	own.submitLock.RLock()
	if own.closed.Load() {
		own.submitLock.RUnlock()
		return ErrClosed
	}
	own.submits.Add(1)
	own.submitLock.RUnlock()

	defer own.submits.Done()

	select {
	case own.tasks <- task:
		return nil
	case <-own.ctx.Done():
		return ErrClosing
	}
}

// SubmitFn 提交一个无 context 的简单函数任务
func (own *Pool) SubmitFn(fn func() error) error {
	return own.Submit(func(_ context.Context) error { return fn() })
}

// TrySubmit 非阻塞地提交任务，若队列满立即返回 false
func (own *Pool) TrySubmit(task Task) bool {
	own.submitLock.RLock()
	defer own.submitLock.RUnlock()

	if own.closed.Load() {
		return false
	}

	select {
	case own.tasks <- task:
		return true
	default:
		return false
	}
}

// GetWorkers 返回当前工作 goroutine 数量
func (own *Pool) GetWorkers() int { return own.workers }

// Release 优雅关闭池：停止接收新任务，等待所有已提交任务退出
//   - 会先取消上下文，使监听 ctx 的任务能尽快返回，避免关闭过程无限阻塞
//   - 已提交到队列的任务仍会被 worker 执行
func (own *Pool) Release() {
	own.closeOnce.Do(func() {
		// 先置关闭标记，阻止新的 Submit 进入
		own.submitLock.Lock()
		own.closed.Store(true)
		own.submitLock.Unlock()

		// 唤醒可能因队列满而阻塞的 Submit，使其返回 ErrClosing
		own.cancel()
		own.submits.Wait()

		// 确认无投递中的调用后再关闭通道，彻底消除 send on closed channel
		own.submitLock.Lock()
		close(own.tasks)
		own.submitLock.Unlock()
	})

	own.wg.Wait()
}

// Close 同 Release，语义保持一致
func (own *Pool) Close() { own.Release() }

func (own *Pool) loop() {
	defer own.wg.Done()
	for task := range own.tasks {
		own.run(task)
	}
}

func (own *Pool) run(task Task) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			if own.panicHandler != nil {
				own.panicHandler(r, stack)
			}
		}
	}()
	_ = task(own.ctx)
}
