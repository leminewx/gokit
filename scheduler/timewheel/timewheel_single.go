package timewheel

import (
	"container/list"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

type (
	// scheduledTask 定义定时任务的数据结构（内部类型，由 AddTask 创建）
	scheduledTask struct {
		id        int64         // 任务唯一ID
		job       func()        // 任务函数
		executeAt int64         // 任务执行时刻（纳秒时间戳）
		canceled  atomic.Bool   // 标记任务已被取消（惰性删除）
		period    time.Duration // 重复执行间隔（0 表示不重复）
		repeats   int           // 重复执行次数（-1 表示一直重复；0 表示不再执行；>0 表示指定次数）
	}

	// SingleWheel 定义单级时间轮的数据结构
	SingleWheel struct {
		slots           []*list.List        // 槽数组
		tickDuration    time.Duration       // 每个槽的时间间隔
		currentPosition int                 // 当前指针位置
		stopChan        chan struct{}       // 停止通道
		taskChan        chan *scheduledTask // 任务队列
		taskMap         sync.Map            // 任务ID映射，用于取消任务
		taskIDGenerator atomic.Int64        // 任务ID生成器
		isRunning       atomic.Bool         // 运行状态原子标记
		stopped         atomic.Bool         // 是否已永久停止（停止后不可重启）
		stopOnce        sync.Once           // 保证停止信号只关闭一次

		clock       timex.Clock  // 时间源，便于测试注入
		handleError ErrorHandler // 任务 panic 处理器

		lifecycleLock sync.Mutex     // 串行化 Start / Stop
		loopWG        sync.WaitGroup // 事件循环 goroutine
		taskWG        sync.WaitGroup // 任务回调 goroutine
	}
)

// NewSingleWheel 新建一个单级时间轮
//   - 可选注入时间源（WithClock）与任务 panic 处理器（WithErrorHandler）
func NewSingleWheel(tickDuration time.Duration, slotNumber int, options ...Option) *SingleWheel {
	if tickDuration <= 0 {
		panic(ErrInvalidTickDuration)
	}

	if slotNumber <= 0 {
		panic(ErrInvalidSlotCount)
	}

	wheel := &SingleWheel{
		slots:        make([]*list.List, slotNumber),
		stopChan:     make(chan struct{}),
		taskChan:     make(chan *scheduledTask, 1024),
		tickDuration: tickDuration,
		clock:        timex.NewSystemClock(),
		handleError:  logErrorHandler,
	}

	// 初始化槽
	for i := 0; i < slotNumber; i++ {
		wheel.slots[i] = list.New()
	}

	for _, option := range options {
		option(wheel)
	}

	return wheel
}

// Start 启动时间轮
//   - 已启动时重复调用无副作用；已停止的时间轮不可再次启动
func (own *SingleWheel) Start() {
	own.lifecycleLock.Lock()
	defer own.lifecycleLock.Unlock()

	if own.stopped.Load() || own.isRunning.Load() {
		return
	}

	// 创建时间轮驱动计时器
	ticker := own.clock.NewTicker(own.tickDuration)
	own.isRunning.Store(true)

	// 异步起动时间轮
	own.loopWG.Go(func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.GetChannel():
				own.tick()
			case task := <-own.taskChan:
				own.addTaskToSlot(task)
			case <-own.stopChan:
				return
			}
		}
	})
}

// Stop 停止时间轮
//   - 可重复调用：并发调用时只有一次真正执行关闭逻辑
//   - 会等待正在执行的任务结束；停止后不可再次启动，AddTask 将返回 0
func (own *SingleWheel) Stop() {
	own.lifecycleLock.Lock()
	if !own.isRunning.Load() {
		own.lifecycleLock.Unlock()
		return
	}

	own.isRunning.Store(false)
	own.stopped.Store(true)
	own.stopOnce.Do(func() { close(own.stopChan) })
	own.lifecycleLock.Unlock()

	// 先等事件循环退出，再等回调结束；两个 WaitGroup 分开使用，避免 Wait 与 Add 并发
	own.loopWG.Wait()
	own.taskWG.Wait()
}

// IsRunning 返回时间轮是否在运行
func (own *SingleWheel) IsRunning() bool {
	return own.isRunning.Load()
}

/*
AddTask 添加定时性任务
  - at：首轮任务的执行时刻
  - job：任务函数（为 nil 时返回 0）
  - period：任务重复执行的时间间隔（<=0 表示只执行一次）
  - repeats：重复执行次数（-1 表示一直重复；缺省时只执行一次）
  - 返回任务 ID，未启动或已停止时返回 0
  - 注意：重复执行任务的间隔时间 period 需要大于时间轮的最小嘀嗒时间
*/
func (own *SingleWheel) AddTask(at time.Time, job func(), period time.Duration, repeats ...int) int64 {
	// 未启动或已停止时拒绝新任务，避免向无人接收的队列投递或向已关闭通道发送
	if !own.isRunning.Load() || job == nil {
		return 0
	}

	jobTask := &scheduledTask{
		id:        own.taskIDGenerator.Add(1),
		job:       job,
		executeAt: at.UnixNano(),
		repeats:   1,
	}

	if period > 0 {
		jobTask.period = period
	}

	// 显式指定重复次数时直接覆盖默认的 1 次，避免出现 "传 N 执行 N+1 次" 的偏差
	if jobTask.period > 0 && len(repeats) > 0 {
		if repeats[0] < 0 {
			jobTask.repeats = -1
		} else {
			jobTask.repeats = repeats[0]
		}
	}

	// 投递任务；与 Stop 并发时优先响应停止信号，不会因通道关闭而 panic
	select {
	case own.taskChan <- jobTask:
		// 投递与停止并发时以停止标志为准，避免返回永远不会被执行的任务 ID
		if own.stopped.Load() {
			return 0
		}

		return jobTask.id
	case <-own.stopChan:
		return 0
	}
}

/*
CancelTask 取消任务
  - taskID：任务 ID
  - onCancel：取消成功时的回调函数，可为 nil
  - 返回是否成功取消
*/
func (own *SingleWheel) CancelTask(taskID int64, onCancel func()) bool {
	if value, loaded := own.taskMap.LoadAndDelete(taskID); loaded {
		value.(*scheduledTask).canceled.Store(true)
		if onCancel != nil {
			onCancel()
		}

		return true
	}

	return false
}

// addTaskToSlot 内部添加任务方法
func (own *SingleWheel) addTaskToSlot(task *scheduledTask) {
	// task.repeats == 0 表示任务已无需执行
	if task.repeats == 0 {
		return
	}

	now := own.clock.GetNow().UnixNano()
	if task.executeAt <= now {
		// 已到期（或传入过去时刻）的任务直接执行，并同样处理重复调度
		own.runTask(task)
		return
	}

	slotNumber := len(own.slots)
	// 计算任务应该在多少个时间滴答后执行
	ticks := (task.executeAt - now) / int64(own.tickDuration)
	// 延迟不足一个滴答时至少放入下一个槽；否则会被放入刚扫过的当前槽而多等一整圈
	if ticks <= 0 {
		ticks = 1
	}
	// 计算任务应该放入的槽位置 (考虑当前指针位置)
	pos := (own.currentPosition + int(ticks)%slotNumber) % slotNumber
	// 将任务添加到对应的槽中
	own.slots[pos].PushBack(task)
	// 保存到任务映射
	own.taskMap.Store(task.id, task)
}

// tick 时间轮转动处理
func (own *SingleWheel) tick() {
	now := own.clock.GetNow().UnixNano()

	// 第一阶段：把当前槽的任务整体摘下到局部切片。
	// 不能在遍历链表的同时把任务回插到同一链表：重复任务重新调度后若仍落在当前槽
	// （周期为整圈的整数倍时 ticks%slotNumber == 0），节点会被追加到链表尾部，
	// 而遍历指针 e = e.Next() 会再次走到它，导致 for 循环永不终止、事件循环卡死。
	currentSlot := own.slots[own.currentPosition]
	var expired []*scheduledTask
	for e := currentSlot.Front(); e != nil; e = e.Next() {
		expired = append(expired, e.Value.(*scheduledTask))
	}
	currentSlot.Init()

	// 第二阶段：逐个处理已摘下的任务，此时回插落到任何槽（包括当前槽）都是安全的。
	for _, task := range expired {
		// 任务取消或重复执行次数结束
		if task.canceled.Load() || task.repeats == 0 {
			own.taskMap.Delete(task.id)
			continue
		}

		// 任务时间未到（延迟超过一轮槽位后回归），重新计算槽位而不是自投递
		if task.executeAt > now {
			own.addTaskToSlot(task)
			continue
		}

		// 任务时间到，执行回调并处理重复调度
		own.runTask(task)
	}

	// 移动指针
	own.currentPosition = (own.currentPosition + 1) % len(own.slots)
}

// runTask 执行一次任务并按需重新调度（仅由事件循环 goroutine 调用）
//   - 重复任务按原定周期推进而不是以执行时刻重新计时，避免累积漂移；
//   - 若已错过多个周期，则对齐到下一个未来执行点，避免补偿风暴。
func (own *SingleWheel) runTask(task *scheduledTask) {
	own.callback(task)

	if task.repeats > 0 {
		task.repeats--
	}

	if task.period <= 0 || task.repeats == 0 {
		own.taskMap.Delete(task.id)
		return
	}

	task.executeAt += int64(task.period)
	if task.executeAt <= own.clock.GetNow().UnixNano() {
		task.executeAt = own.clock.GetNow().Add(task.period).UnixNano()
	}

	own.addTaskToSlot(task)
}

// callback 执行任务
func (own *SingleWheel) callback(task *scheduledTask) {
	// taskWG 只记录回调，与事件循环的 loopWG 分开，避免 Stop 时 Wait 与 Add 并发
	own.taskWG.Go(func() {
		// 任务函数的 panic 不能穿透到事件循环，否则会终止整个进程
		defer func() {
			if r := recover(); r != nil {
				own.handleError(task.id, wrapError(ErrTaskPanic, "%v\n%s", r, debug.Stack()))
			}
		}()

		task.job()
	})
}
