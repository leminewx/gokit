package queue

import (
	"container/heap"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

// DelayQueue 定义延时队列的数据结构
type DelayQueue struct {
	heap         *taskHeap
	heapLock     sync.RWMutex
	tickDuration time.Duration // 延时队列的嘀嗒间隔时间
	taskMap      sync.Map      // 任务id映射，用于取消任务
	taskChan     chan *task    // 任务队列
	stopChan     chan struct{} // 停止通道
	idGenerator  atomic.Int64  // 任务id生成器
	isRunning    atomic.Bool   // 运行状态原子标记
	stopped      atomic.Bool   // 是否已永久停止（停止后不可重启）

	lifecycleLock sync.Mutex     // 串行化 Start / Stop，避免并发启动与停止交错
	stopOnce      sync.Once      // 保证停止信号只关闭一次
	callbackWG    sync.WaitGroup // 用于等待所有回调完成
	workerWG      sync.WaitGroup // 用于等待工作goroutine完成

	clock       timex.Clock  // 时间源，便于测试注入
	handleError ErrorHandler // 任务 panic 处理器
}

// NewDelayQueue 新建一个延时队列
//   - 可选注入时间源（WithClock）、panic 处理器（WithErrorHandler）与通道容量（WithChanSize）
func NewDelayQueue(tickDuration time.Duration, options ...Option) *DelayQueue {
	if tickDuration <= 0 {
		panic(ErrInvalidTickDuration)
	}

	heapInstance := make(taskHeap, 0)
	queue := &DelayQueue{
		heap:         &heapInstance,
		tickDuration: tickDuration,
		stopChan:     make(chan struct{}),
		taskChan:     make(chan *task, 1024),
		clock:        timex.NewSystemClock(),
		handleError:  logErrorHandler,
	}

	for _, option := range options {
		option(queue)
	}

	heap.Init(queue.heap)
	return queue
}

// Start 启动延时队列
//   - 已启动时重复调用无副作用；已停止的队列不可再次启动
func (own *DelayQueue) Start() {
	own.lifecycleLock.Lock()
	defer own.lifecycleLock.Unlock()

	if own.stopped.Load() || own.isRunning.Load() {
		return
	}

	// 创建延时队列驱动计时器
	ticker := own.clock.NewTicker(own.tickDuration)
	own.isRunning.Store(true)
	own.workerWG.Add(1)

	// 异步运行延时队列
	go func() {
		defer own.workerWG.Done()
		defer ticker.Stop()

		for {
			select {
			case task := <-own.taskChan:
				own.addTask(task)
			case <-ticker.GetChannel():
				own.tick()
			case <-own.stopChan:
				return
			}
		}
	}()
}

// Stop 停止延时队列
//   - 可重复调用；会等待工作循环与正在执行的回调结束
//   - 停止后不可再次启动，AddTask 将返回 0
func (own *DelayQueue) Stop() {
	own.lifecycleLock.Lock()
	if !own.isRunning.Load() {
		own.lifecycleLock.Unlock()
		return
	}

	own.isRunning.Store(false)
	own.stopped.Store(true)
	own.stopOnce.Do(func() { close(own.stopChan) })
	own.lifecycleLock.Unlock()

	own.workerWG.Wait()

	// 等待所有回调任务完成
	own.callbackWG.Wait()
}

// AddTask 添加定时(循环)任务
//
//   - at：首轮任务的执行时刻
//   - period：任务重复执行的时间间隔（<=0 表示只执行一次）
//   - remaining：重复执行次数（-1 表示一直重复；缺省时只执行一次）
//   - 返回任务 ID，未启动或已停止时返回 0
//
// 注意：重复执行任务的间隔时间 period 需要大于延时队列的最小嘀嗒时间
func (own *DelayQueue) AddTask(at time.Time, job func(), period time.Duration, remaining ...int) int64 {
	// 检查延时队列是否已停止
	if !own.isRunning.Load() {
		return 0
	}

	if job == nil {
		return 0
	}

	jobTask := &task{
		id:        own.idGenerator.Add(1),
		job:       job,
		executeAt: at.UnixNano(),
		period:    period,
		remaining: 1, // 默认执行一次
	}

	if period > 0 && len(remaining) > 0 {
		jobTask.remaining = remaining[0]
		if jobTask.remaining < 0 {
			jobTask.remaining = -1 // 无限重复
		}
	}

	// 先同步注册到任务映射，使 CancelTask 在任务被工作协程取出之前即可生效
	own.taskMap.Store(jobTask.id, jobTask)

	// 添加任务到任务队列；与 Stop 并发时优先响应停止信号，避免向已关闭通道发送
	select {
	case own.taskChan <- jobTask:
		// 投递与停止并发时以停止标志为准，避免返回永远不会被执行的任务 ID
		if own.stopped.Load() {
			own.taskMap.Delete(jobTask.id)
			return 0
		}

		return jobTask.id
	case <-own.stopChan:
		own.taskMap.Delete(jobTask.id)
		return 0
	}
}

// CancelTask 取消任务
//   - taskID：任务 ID
//   - onCancel：取消成功时的回调函数，可为 nil
//   - 返回是否成功取消
//   - 先置取消标记再执行回调：回调内读取“是否已取消”时应已为 true，
//     与 timewheel.CancelTask 的顺序保持一致
func (own *DelayQueue) CancelTask(taskID int64, onCancel func()) bool {
	if value, loaded := own.taskMap.LoadAndDelete(taskID); loaded {
		jobTask := value.(*task)
		jobTask.canceled.Store(true)
		if onCancel != nil {
			onCancel()
		}

		return true
	}

	return false
}

// addTask 添加任务到延时队列
func (own *DelayQueue) addTask(task *task) {
	// 检查任务是否应该执行
	if task.remaining == 0 || task.canceled.Load() {
		own.taskMap.Delete(task.id)
		return
	}

	now := own.clock.GetNow().UnixNano()

	// 定时时间已到或已过，立即执行
	if task.executeAt <= now {
		own.executeTask(task)
		return
	}

	// 加入最小堆
	own.pushTask(task)
}

// pushTask 将任务放回最小堆（内部持堆锁，避免与 tick 的堆操作并发）
func (own *DelayQueue) pushTask(task *task) {
	own.heapLock.Lock()
	heap.Push(own.heap, task)
	own.heapLock.Unlock()

	// 保存到任务映射
	own.taskMap.Store(task.id, task)
}

// tick 周期性处理延时队列
//   - 仅在持锁期间完成堆的弹出，用户回调在锁外执行，避免长时间阻塞队列与停机
func (own *DelayQueue) tick() {
	now := own.clock.GetNow().UnixNano()

	// 收集所有到期任务
	expired := make([]*task, 0, 8)
	own.heapLock.Lock()
	for own.heap.Len() > 0 {
		// 检查堆顶任务是否到期
		task := (*own.heap)[0]
		if task.executeAt > now {
			break
		}

		// 弹出堆顶任务
		heap.Pop(own.heap)
		expired = append(expired, task)
	}
	own.heapLock.Unlock()

	// 在锁外逐个执行
	for _, task := range expired {
		if task.canceled.Load() || task.remaining == 0 {
			own.taskMap.Delete(task.id)
			continue
		}

		own.executeTask(task)
	}
}

// executeTask 执行任务并处理重复逻辑
func (own *DelayQueue) executeTask(task *task) {
	// 执行任务
	own.safeCallback(task)

	// 更新重复任务
	if task.period > 0 && task.remaining != 0 {
		// 计算下一次执行时间
		task.executeAt = own.clock.GetNow().Add(task.period).UnixNano()

		// 更新剩余次数
		if task.remaining > 0 {
			task.remaining--
		}

		// 如果还有剩余执行次数，重新加入队列
		if task.remaining != 0 && !task.canceled.Load() {
			own.pushTask(task)
		} else {
			own.taskMap.Delete(task.id)
		}
	} else {
		own.taskMap.Delete(task.id)
	}
}

// safeCallback 安全执行回调
func (own *DelayQueue) safeCallback(jobTask *task) {
	own.callbackWG.Add(1)
	go func() {
		defer own.callbackWG.Done()
		defer func() {
			if r := recover(); r != nil {
				own.handleError(jobTask.id, wrapError(ErrTaskPanic, "%v\n%s", r, debug.Stack()))
			}
		}()

		jobTask.job()
	}()
}

// GetSize 返回队列中任务数量
func (own *DelayQueue) GetSize() int {
	own.heapLock.RLock()
	defer own.heapLock.RUnlock()
	return own.heap.Len()
}

// IsRunning 返回队列是否在运行
func (own *DelayQueue) IsRunning() bool {
	return own.isRunning.Load()
}
