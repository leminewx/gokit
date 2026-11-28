# 基础能力 API

本文覆盖与业务领域无关的基础包：文本、时间、文件、版本、缓存、颜色、日志、NMEA、
协程池、进度条、限流、计速与数据流。导入前缀统一为 `gitee.com/leminewx/gokit/<包名>`，
这些包各自独立成目录，不存在 `toolkit/` 之类的公共父目录。

## `stringx` 文本工具

| 符号 | 说明 |
|------|------|
| `ConvertToSnakeCase(s string) string` | 驼峰转蛇形：`HelloWorld` → `hello_world`、`HTTPServer` → `http_server`、`UserID` → `user_id`；已含下划线或空格的字符串保持原样 |
| `HumanizeBytes(bytes int64) string` | 字节数转人类可读格式：以 1024 为进制、IEC 单位（`B` / `KiB` / `MiB` / `GiB` / `TiB` / `PiB`），保留两位小数，小于 1024 时输出 `%d B` |
| `StripANSIString(s string) string` / `StripANSIBytes(data []byte) []byte` | 剪除 ANSI 转义序列；不含 `ESC` 时直接返回原值 |
| `JoinInts[T gokit.Int](items []T, sep string) string` / `JoinUints[T gokit.Uint](items []T, sep string) string` | 整型切片按分隔符拼接 |
| `ParseNumbers[T gokit.Int \| gokit.Uint](items []string) ([]T, error)` | 十进制字符串切片解析为 `T`；按 `T` 的位宽做溢出校验，任一元素失败返回 `ErrInvalidNumber` 并保留 `strconv` 原始错误链 |
| `GetBuilder() *strings.Builder` / `PutBuilder(builder *strings.Builder)` | 复用 `strings.Builder`；`PutBuilder(nil)` 忽略，容量超过 64 KiB 的实例直接丢弃 |

```go
text := stringx.HumanizeBytes(1536)                 // "1.50 KiB"
nums, err := stringx.ParseNumbers[uint16](fields)   // 溢出即报错
```

## `timex` 可注入时间源

| 符号 | 说明 |
|------|------|
| `Clock` 接口 | `GetNow() time.Time`、`NewTicker(interval) Ticker` |
| `Ticker` 接口 | `GetChannel() <-chan time.Time`、`Stop()` |
| `NewSystemClock() Clock` | 生产默认时间源，基于 `time` 包 |
| `NewManual(start time.Time) *Manual` | 从 `start` 开始的手动时钟，时间不会自行流逝 |
| `NewManualAt() *Manual` | 从 `time.Unix(0, 0).UTC()` 开始的手动时钟 |
| `(*Manual) Advance(d time.Duration)` / `AdvanceTo(target time.Time)` | 推进时间并补发期间到期的嘀嗒；单个嘀嗒器单次最多补发 4096 个，超出时直接跳到未来 |
| `(*Manual) NewTicker(interval)` | `interval <= 0` 按 `1ns` 处理，避免除零 |

`Manual` 并发安全，可在被测 goroutine 与测试 goroutine 之间共享；`Ticker.Stop()` 会把
自身从时钟中移除，不会泄漏。

```go
source := timex.NewManualAt()
c := cache.NewTTLCache[string, int](time.Minute, cache.WithClock[string, int](source))
c.Put("k", 1, 0)
source.Advance(2 * time.Minute) // 过期判定无需 sleep
```

## `filex` 原子文件操作与路径定位

| 函数 | 说明 |
|------|------|
| `WriteFileAtomic(path string, data []byte, perm os.FileMode) error` | 同目录临时文件 → `Chmod` → `Sync` → `Rename`；`perm == 0` 时按 `0600`；失败不破坏已有文件并清理临时文件 |
| `CopyFileAtomic(src, dst string) error` | 原子复制，权限继承源文件 |
| `CopyFileAtomicWithMode(src, dst string, perm os.FileMode) error` | 原子复制并显式指定权限 |
| `GetModuleRoot() (string, error)` | 从工作目录向上定位 `go.mod`；工作目录不在模块内时退回到包源码位置重试，找不到返回 `ErrModuleRoot` |
| `ResolvePath(root, relative string) (string, error)` | 把相对路径限制在 `root` 内：忽略前导 `/`，空路径返回 `ErrInvalidPath`，`..` 或指向根目录本身返回 `ErrOutsideRoot`；只做路径计算，不检查存在性与符号链接 |

非 Windows 平台会尽力 fsync 目录项；临时文件统一命名为 `.<目标文件名>.tmp-*`。
哨兵错误：`ErrInvalidPath`、`ErrCreateTemp`、`ErrOpen`、`ErrCopy`、`ErrWrite`、
`ErrChmod`、`ErrSync`、`ErrRename`、`ErrOutsideRoot`、`ErrModuleRoot`。

## `version` 版本号与区间表达式

| 函数 | 说明 |
|------|------|
| `Compare(left, right string) int` | 缺失段视为 0（`v6.4 == v6.4.0`），数字段按数值比较，发布后缀不参与；返回 `-1/0/1` |
| `IsEqual(left, right string) bool` | `Compare == 0` |
| `Normalize(raw string) string` | 去空白、转小写、去 `v` 前缀 |
| `NormalizeActual(raw string) string` | 先抽取版本核心再归一（`app_V7.0.0.0_20260401` → `7.0.0.0`） |
| `Extract(raw string) string` | 抽取版本核心并保留 `v` 前缀，未命中返回空串 |
| `SplitSegments(ver string) []string` / `IsDigits(s string) bool` / `ParseInt(s string) (int, bool)` | 段拆分与解析辅助 |
| `IsUnlimited(expr string) bool` | 是否为「不限」（空串 / `*` / `any` / `all`） |
| `Validate(expr string) error` | 校验每个条件均以 `=`、`≥`、`≤`、`>`、`<` 开头 |
| `NormalizeExpr(expr string) string` | 比较符统一为 `≥/≤`，单行用空格、多区间换行 |
| `Match(expr, actual string) (bool, error)` | 行内 AND、行间 OR；`actual` 为空返回 `ErrEmptyVersion`，不静默放行 |
| `Explain(expr string) string` | 输出规范化描述（如 `< [v3.0, v3.999999]`） |

表达式支持 `x` 通配（按操作符方向取上/下界）、Unicode `≥/≤` 与全角比较符。
哨兵错误：`ErrEmptyVersion`、`ErrInvalidVersionExpr`、`ErrInvalidVersion`。

## `cache` 缓存

LRU 与 TTL 两个泛型实现，全部方法线程安全（LRU 用 `sync.Mutex`，TTL 用 `sync.RWMutex`），
构造选项共用。

| 选项 | 说明 |
|------|------|
| `WithClock[K,V](source timex.Clock) Option[K,V]` | 注入时间源，仅 TTL 生效；`nil` 忽略 |
| `WithOnEvict[K,V](onEvict func(key K, value V)) Option[K,V]` | 元素移出缓存时回调；回调在锁外执行，可安全操作缓存本身；`nil` 忽略 |

### LRUCache

```go
c := cache.NewLRUCache[string, int](128) // capacity <= 0 表示不限制容量
if evicted := c.Put("a", 1); evicted {
    // 本次写入淘汰了最久未使用项
}
v, ok := c.Get("a")  // 命中并刷新热度
v, ok = c.Peek("a")  // 不刷新热度
ok = c.Contain("a")  // 只判断存在
```

| 方法 | 说明 |
|------|------|
| `Put(key K, value V) (evicted bool)` | 写入并刷新热度，超容量时淘汰最久未使用项 |
| `Get(key K) (V, bool)` / `Peek(key K) (V, bool)` / `Contain(key K) bool` | 读取并刷新 / 读取不刷新 / 仅判存 |
| `Delete(key K) (V, bool)` | 删除并返回原值 |
| `GetKeys() []K` / `GetLength() int` / `GetCapacity() int` | 键快照（最近使用在前）/ 元素个数 / 容量上限 |
| `Clear() int` | 清空并返回清除数量 |

### TTLCache

```go
c := cache.NewTTLCache[string, int](time.Minute) // defaultTTL = 0 表示默认永不过期
c.Put("token", 42, 0)                            // ttl <= 0 用默认 TTL
v, ttl, ok := c.GetWithTTL("token")              // ttl 为剩余有效期，永不过期为 0
stop := c.StartAutoPurge(ctx, time.Minute)       // ctx 为 nil 时用 Background
defer stop()
```

| 方法 | 说明 |
|------|------|
| `GetDefaultTTL() time.Duration` | 默认有效期，0 表示永不过期 |
| `Put(key K, value V, ttl time.Duration)` | `ttl <= 0` 回退到默认 TTL；覆盖旧值不触发回调 |
| `Get(key K) (V, bool)` / `GetWithTTL(key K) (V, time.Duration, bool)` | 命中过期键时删除并返回零值 + `false` |
| `Peek(key K) (V, bool)` / `Contain(key K) bool` | 与 `Get` 一致地清理过期项 |
| `Delete(key K) (V, bool)` / `Clear() int` / `Purge() int` | 删除单个 / 清空 / 清理全部过期项 |
| `GetKeys() []K` / `GetLength() int` | 可能包含尚未懒清理的过期项 |
| `StartAutoPurge(ctx, interval) (stop func())` | 后台清理协程；`interval <= 0` 按 1 分钟；`stop()` 幂等，取消 `ctx` 亦可退出 |

过期清理是懒执行的：需要严格控制内存时配合 `Purge` / `StartAutoPurge` 使用。

## `color` ANSI 着色

```go
color.New(color.FG_GREEN, color.STYLE_BOLD).Println("成功")
color.New(color.NewRGB(255, 128, 0)).Printf("错误码: %d\n", 500)
color.SetNoColor(true) // 写文件、非 TTY、CI 场景
```

| 符号 | 说明 |
|------|------|
| `New(codes ...Code) style` | 组合样式与颜色，返回支持 `Print/Printf/Println/Sprint/Sprintf/Sprintln/Fprint*` 的对象 |
| `STYLE_BOLD` … `STYLE_STRIKETHROUGH` | 文本样式（加粗、淡色、斜体、下划线、闪烁、快速闪烁、反转、隐藏、删除线） |
| `FG_BLACK` … `FG_WHITE` / `FG_BRIGHT_BLACK` … `FG_BRIGHT_WHITE` | 前景色（基础 8 色 / 亮色 8 色） |
| `BG_BLACK` … `BG_WHITE` / `BG_BRIGHT_BLACK` … `BG_BRIGHT_WHITE` | 背景色 |
| `NewRGB(r, g, b uint8) Code` / `NewBgRGB(r, g, b uint8) Code` | 24 位真彩色 |
| `SetNoColor(disabled bool)` / `IsNoColor() bool` | 全局开关；`NO_COLOR` 非空或 `TERM=dumb` 时包初始化自动关闭 |
| `SetOutput(w io.Writer)` | 修改默认输出流（默认 `os.Stdout`），`nil` 忽略 |

## `logger` 文本日志 Handler

基于 `log/slog` 的 `PrettyHandler`，实现 `slog.Handler`，可并发写入。

```go
h := logger.NewPrettyHandler(os.Stdout, &logger.Options{
    Level:     slog.LevelDebug,
    WithColor: true,
})
log := slog.New(h)
log.Info("服务启动", "host", "127.0.0.1", "port", 8080)
// INFO  - 2026-08-28 22:31:44.416 - 服务启动 | host=127.0.0.1 port=8080
```

| 符号 | 说明 |
|------|------|
| `NewPrettyHandler(w io.Writer, opts *Options) slog.Handler` | `w == nil` 用 `os.Stdout`；`opts == nil` 用默认值；`opts` 会被复制，不修改调用方入参 |
| `NewLogger(handler slog.Handler) *Logger` | 包装为带源码定位能力的 `*Logger`（`Debug/Info/Warn/Error/Log/With/WithGroup/GetHandler`） |
| `Options.Level slog.Leveler` | 最低输出级别，默认 `slog.LevelInfo` |
| `Options.TimeFormat string` | 时间格式，默认 `2006-01-02 15:04:05.000` |
| `Options.WithColor bool` | 级别着色（DEBUG 青、INFO 蓝、WARN 黄、ERROR 红） |
| `Options.WithSource bool` | 追加 `file:line` |
| `Options.AttrSplit` / `Options.AttrConnect` | 属性分隔符（默认空格）与键值连接符（默认 `=`） |
| `Options.ReplaceAttr func([]string, slog.Attr) slog.Attr` | 标准替换钩子，可用于脱敏 |

`WithColor` 仅建议用于终端；写文件时关闭，或同步调用 `color.SetNoColor(true)`。

## `nmea` NMEA 0183 解析

```go
gga, err := nmea.ParseGGA("$GNGGA,082315.00,2232.12345,N,11354.54321,E,1,12,0.98,45.6,M,-2.1,M,,*5C")
if err == nil {
    fmt.Println(gga.LatitudeDir, gga.LongitudeDir, gga.Satellites)
    fmt.Println(gga.String()) // 按定长位数重新拼接，可被再次解析
}

rmc, err := nmea.ParseRMC(sentence)
lla := rmc.GetLLAPoint() // 默认 WGS84
```

| 符号 | 说明 |
|------|------|
| `ParseGGA(sentence string) (*GGA, error)` | 解析 GGA；校验 talker、语句类型与校验和 |
| `ParseRMC(sentence string) (*RMC, error)` | 解析 RMC；磁偏角西偏为负 |
| `(*GGA) String()` / `(*RMC) String()` | 按 NMEA 0183 定长位数输出，保证 round-trip |
| `(*GGA) GetLLAPoint(dat ...datum.Datum)` / `(*RMC) GetLLAPoint(...)` | 转经纬高，缺省 `datum.WGS84` |
| `ConvertRawToDecimal(raw float64) float64` | 度分格式（`ddmm.mmmm` / `dddmm.mmmm`）转十进制度 |

枚举：`Type`（`TYPE_GGA` / `TYPE_RMC`）、`System`（`SYSTEM_GPS` / `SYSTEM_GALILEO` /
`SYSTEM_GLONASS` / `SYSTEM_MULTI_GNSS`）、`Quality`（`QUALITY_INVALID` … `QUALITY_ESTIMATED`）、
`Mode`（`MODE_AUTONOMOUS` … `MODE_SIMULATOR`）。字段含义见结构体注释。

哨兵错误：`ErrNotGGASentence`、`ErrGGACountMismatch`、`ErrNotRMCSentence`、
`ErrRMCCountMismatch`、`ErrMissingChecksum`。

## `pool` 协程池

```go
p := pool.New(
    pool.WithWorkers(8),      // 缺省 4
    pool.WithQueueSize(100),  // 缺省 0（无缓冲）
    pool.WithPanicHandler(func(recv any, stack []byte) { log.Printf("panic: %v", recv) }),
)
defer p.Release()

_ = p.Submit(func(ctx context.Context) error { return nil })
ok := p.TrySubmit(task) // 队列满时立即返回 false
```

| 符号 | 说明 |
|------|------|
| `New(opts ...Option) *Pool` | 创建并立即启动 worker；`workers <= 0` 取 4 |
| `Submit(Task) error` / `SubmitFn(func() error) error` | 阻塞提交；队列满时等待空位，池关闭返回 `ErrClosed`，关闭中返回 `ErrClosing` |
| `TrySubmit(Task) bool` | 非阻塞提交，队列满或已关闭返回 `false` |
| `Release()` / `Close()` | 优雅关闭：先取消上下文唤醒阻塞的 `Submit`，再关闭任务队列并等待全部 worker 退出 |
| `GetWorkers() int` | worker 数量 |
| `Task func(ctx context.Context) error` | 任务签名 |
| `WithWorkers` / `WithQueueSize` / `WithPanicHandler` | 配置项 |

任务 panic 会被统一恢复，不会拖垮池；`Release` 已提交的任务仍会执行，依赖 `Task(ctx)`
的清理逻辑应幂等。

## `progressbar` 终端进度条

```go
bar := progressbar.New(totalBytes) // total <= 0 时按 0 处理并直接结束
bar.SetType(progressbar.TYPE_BYTES)
bar.SetPrefix("下载：")
bar.SetStyle(progressbar.StyleSquarePoint)

bar.Add(int64(n))
bar.Update(current) // 仅接受递增
if bar.IsFinished() {
    // 自动收尾
}
bar.Finish() // 异常中断时手动收尾，重复调用不会重复输出
```

| 符号 | 说明 |
|------|------|
| `New(total int64) *Progressbar` | 默认样式 `StyleBarPoint`（`█·`）、默认类型 `TYPE_BYTES`、默认写 `os.Stdout` |
| `Add(increment int64)` | 累加；`increment <= 0` 被忽略 |
| `Update(current int64)` | 直接设置当前值；回退被忽略 |
| `GetCount() int64` / `GetTotal() int64` / `GetProgress() float64` | 当前值 / 总量 / 进度（0.0~1.0，`total <= 0` 返回 0） |
| `IsFinished() bool` / `Finish()` / `Reset() error` | 是否完成 / 手动收尾 / 重置状态（不重置总量与类型） |
| `SetPrefix` / `SetStyle` / `SetType` / `SetSpinners` / `SetWriter` / `SetFinish` | 配置项，均无返回值，逐条调用即可 |

内置样式（`Style [2]rune`，`[0]` 已完成、`[1]` 未完成）：`StyleSquarePoint`（`■·`）、
`StyleSquareSpace`（`■␣`）、`StyleBarPoint`（`█·`）、`StyleBarSpace`（`█␣`）、
`StyleRhombus`（`▰▱`）。字节类型按 1024 进制展示 `B/KiB/MiB/GiB`。
渲染含 ANSI 序列（清行、隐藏/显示光标），仅适用于 ANSI 终端。

## `ratelimiter` 令牌桶限流

```go
l := ratelimiter.New(100, 10) // 每秒 100 个令牌，桶容量 10
if l.Allow() {
    // 放行
}
if err := l.WaitN(ctx, 3); err == nil {
    // 等到 3 个令牌
}
```

| 方法 | 说明 |
|------|------|
| `New(rate, capacity float64) *Limiter` | `rate >= 0`（0 表示不补充）、`capacity > 0`；非法参数 panic `ErrInvalidConfig` |
| `Allow() bool` / `AllowN(n int) bool` | 非阻塞获取；`n < 1` 返回 `false` |
| `Wait(ctx) error` / `WaitN(ctx, n int) error` | 阻塞等待；`ctx` 为 `nil` 时用 `context.Background()`；`n < 1`、`n > capacity` 或速率为 0 且令牌不足时返回 `ErrUnattainable`，不会永久阻塞 |
| `SetRate(rate, capacity float64)` | 动态调整；参数约束与 `New` 一致，容量下调后超出部分立即丢弃 |

## `speedometer` 滑动窗口计速

```go
sp := speedometer.New(3*time.Second) // 默认 7 个桶，建议 5~10
sp.Add(len(chunk))
fmt.Println(sp.GetSpeed()) // 事件数/秒
```

| 符号 | 说明 |
|------|------|
| `New(windowSize time.Duration, options ...Option) Speedometer` | `windowSize <= 0` 时取 2 秒；桶数缺省 7 |
| `Add(int)` / `GetSpeed() float64` | 增加计数（负数表示减少）/ 当前速率 |
| `WithBucketNumber(number int)` / `WithClock(source timex.Clock)` | 桶数量 / 时间源 |

## `stream` 泛型数据流

```go
stream.NewStreamOf(1, 2, 3, 4).
    Filter(func(it stream.Item[int]) bool { return it.GetValue()%2 == 0 }).
    Transform(func(it stream.Item[int]) int { return it.GetValue() * 10 }).
    Consume(func(it stream.Item[int]) {
        fmt.Println(it.GetValue())
    })
```

| 构造 | 说明 |
|------|------|
| `NewStreamOf(values ...T)` / `NewStreamOfContext(ctx, values ...T)` | 从切片创建（可选绑定取消信号） |
| `NewStreamBy(source <-chan T)` / `NewStreamByContext(ctx, source)` | 从通道创建；通道由调用方关闭 |
| `NewStreamFromLineFile[T](ctx, reader io.ReadCloser, format func(line []byte, err error) (T, bool))` | 按行读取，结束或取消后自动关闭 reader |
| `NewStreamFromJsonFile[T](ctx, reader, format func(data T, err error) (T, bool), newer ...func() T)` | 读取 JSON 数组（`[]object`），自动关闭 reader |
| `SetStreamCacheSize(size int)` | 调整内部通道缓存，仅对新创建的通道生效 |

| 方法 | 说明 |
|------|------|
| `WithContext(ctx)` / `WithClock(source)` | 替换下游算子的取消信号 / 时间源 |
| `Transform(fn, callback...)` | 元素转换，零值继续向下游流动 |
| `Filter(fn) bool` | 保留返回 `true` 的元素 |
| `Group(func(Item[T]) string)` | 为元素设置分组，供后续按组消费 |
| `Deduplicate(keyFn, window...)` | 去重，`window` 用于定期重置记录、防止内存增长 |
| `Concat(...Stream[T])` | 聚合多条流 |
| `Peek(fn, callback...)` | 窥视但不改变数据 |
| `TakeFirst(n)` / `TakeLast(n)` | 只接收前 / 后 N 个元素 |
| `SampleByStep(n)` / `SampleByTime(d)` | 按步长 / 时间采样 |
| `FanOut(num) []Stream[T]` | 散出为多条子流；任一子流阻塞会拖住整体，子流持有独立元素实例 |
| `WriteToJSON(w io.Writer)` | 旁路写 JSON 数组，失败不影响主流 |
| `Consume(fn, async...)` / `Read() (Item[T], bool)` / `GetChannel() <-chan Item[T]` / `Wait()` | 消费、手动读取与等待结束 |

| 辅助函数 | 说明 |
|----------|------|
| `Map[T,R](s Stream[T], mapper func(Item[T]) R) Stream[R]` | 一对多映射 |
| `Batch[T](s Stream[T], num int, timeout ...time.Duration) Stream[[]T]` | 按数量或超时切批 |
| `ZipLeft(s1, s2, match, mapper, timeout, onTimeout)` | 左连接对齐，超时走 `onTimeout` |
| `ZipLeftByKey(s1, s2, keyFunc1, keyFunc2, mapper, timeout, onTimeout)` | 按键等值对齐，`O(1)` 匹配 |

`Item[T]` 提供 `GetValue/SetValue`、`GetGroup/SetGroup`、`GetIndex/SetIndex`。
每条流必须及时消费，否则上游会被反压阻塞；Go 接口方法不支持泛型参数，因此接口层
不提供 `Map[U]`，需要换元素类型时使用包级 `Map` / `Transform`。