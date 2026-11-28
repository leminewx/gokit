# scheduler 调度 API

导入前缀：`gitee.com/leminewx/gokit/scheduler/...`

两个包都是**主动调度器**：`Start` 后启动事件循环，`Stop` 会等待正在执行的回调结束；
停止后不可再次启动，`AddTask` 返回 0。二者都依赖 `timex.Clock`，测试可注入 `timex.NewManual()`。

共同选项：

| 选项 | 说明 |
|------|------|
| `WithClock(source timex.Clock)` | 替换时间源，缺省 `timex.NewSystemClock()`；传入 `nil` 时忽略 |
| `WithErrorHandler(handler ErrorHandler)` | 任务回调 panic 的处理函数，缺省写入标准日志 |

`ErrorHandler` 签名为 `func(taskID int64, err error)`，`err` 已包装 `ErrTaskPanic` 并附带 panic 值与调用栈。

## scheduler/timewheel

单级时间轮，目录名与包名均为 `timewheel`。

```go
import "gitee.com/leminewx/gokit/scheduler/timewheel"

wheel := timewheel.NewSingleWheel(100*time.Millisecond, 60)
wheel.Start()
defer wheel.Stop()

id := wheel.AddTask(time.Now().Add(5*time.Second), func() { doWork() }, 0)
// 周期任务：period > 0，repeats 指定次数（不传表示只执行一次，-1 表示无限）
id = wheel.AddTask(time.Now().Add(time.Second), job, time.Second, 3)
wheel.CancelTask(id, func() { fmt.Println("已取消") })
```

| 符号 | 说明 |
|------|------|
| `NewSingleWheel(tickDuration time.Duration, slotNumber int, options ...Option) *SingleWheel` | 刻度与槽位数；参数非法时 panic `ErrInvalidTickDuration` / `ErrInvalidSlotCount` |
| `(*SingleWheel) AddTask(at time.Time, job func(), period time.Duration, repeats ...int) int64` | 首次执行时刻、任务、周期、重复次数；返回任务 ID |
| `(*SingleWheel) CancelTask(taskID int64, onCancel func()) bool` | 取消任务，`onCancel` 在取消成功后调用 |
| `(*SingleWheel) Start()` / `Stop()` | 启动 / 停止；停止后再调用 `Start` 为无操作 |
| `(*SingleWheel) IsRunning() bool` | 当前是否处于运行状态 |
| `ErrTaskPanic` | 任务回调 panic 的包装错误 |

任务采用惰性删除，取消后不会立即从槽位移除；重复任务的 `period` 应大于最小滴答间隔。
首次执行时刻已过期的重复任务不会被丢弃：首轮立即执行，之后按 `period` 继续重复，
执行完成后自动从任务表移除。

## scheduler/queue

基于最小堆的延迟队列。

```go
import "gitee.com/leminewx/gokit/scheduler/queue"

q := queue.NewDelayQueue(100*time.Millisecond, queue.WithChanSize(128))
q.Start()
defer q.Stop()

id := q.AddTask(time.Now().Add(time.Second), func() {}, 0)
q.CancelTask(id, nil)
fmt.Println(q.GetSize(), q.IsRunning())
```

| 符号 | 说明 |
|------|------|
| `NewDelayQueue(tickDuration time.Duration, options ...Option) *DelayQueue` | 刻度与选项；`tickDuration <= 0` 时 panic `ErrInvalidTickDuration` |
| `(*DelayQueue) AddTask(at time.Time, job func(), period time.Duration, remaining ...int) int64` | 添加任务，返回任务 ID；`job == nil` 或未启动时返回 0 |
| `(*DelayQueue) CancelTask(taskID int64, onCancel func()) bool` | 取消任务 |
| `(*DelayQueue) GetSize() int` / `IsRunning() bool` | 队列长度 / 运行状态 |
| `(*DelayQueue) Start()` / `Stop()` | 启动 / 停止；停止后再调用 `Start` 为无操作 |
| `WithChanSize(size int)` | 内部任务通道容量，缺省 1024 |
| `ErrTaskPanic` | 任务回调 panic 的包装错误 |

`DelayQueue` 内部使用 `heapLock`、`sync.Map` 与原子状态，可并发调用。
`Stop` 会等待正在执行与已入队的回调结束；停止后 `AddTask` 返回 0，
与 `Stop` 并发提交的任务若恰在关闭后入队，`AddTask` 同样返回 0。