# 架构与约定

## 定位

gokit 是一个**工具库集合**，不是框架：没有全局状态、没有强制生命周期、没有注册表。
调用方按需导入子包，每个子包可以独立使用，也可以互相组合。

## 目录结构

全部包都直接挂在模块根路径下，没有中间层：

```
gokit（根包：泛型数值约束 + 通用工具）
├── stringx/       stdlib 增强：文本（命名转换、ANSI 剪除、数值切片、Builder 池）
├── timex/         stdlib 增强：可注入时间源（系统时钟 / 手动时钟）
├── filex/         stdlib 增强：原子文件落盘
├── version/       版本号比较与区间表达式
├── logger/        log/slog 彩色文本 Handler
├── color/         ANSI 颜色与样式
├── cache/         LRU / TTL 缓存
├── pool/          Goroutine 池
├── ratelimiter/   令牌桶限流
├── progressbar/   终端进度条
├── speedometer/   环形桶速率计
├── stream/        泛型流式处理
├── nmea/          NMEA 0183 报文
├── configs/       配置读取：INI / JSON
├── cryptos/       密码学：RSA 密钥与加解密签名、bcrypt 口令散列
├── maths/         纯计算：统计、百分位、数值分析（求根/积分/插值/ODE/微分/滤波/拟合/优化/频谱/采样）、地理、投影、几何、向量、机器学习
├── nets/          通信：TCP、编解码、Mesh、SSH（含指标采集）、SMTP、串口
├── scheduler/     主动调度：延迟队列、单级时间轮
├── structures/    纯内存结构：字典、集合、位集、布隆、Trie、FSM
└── harness/       开发辅助：assetpath（数据目录定位，库）/ mesh_demo（示例，main）
```

依赖方向总体自下而上：`stringx` / `timex` / `filex` / `version` / `harness/assetpath`
不依赖任何内部包；`structures` / `maths` 不依赖 `nets`；`nets/mesh` 复用 `structures/tree`（主题匹配）；
`scheduler`、`cache`、`speedometer` 依赖 `timex`；`logger` 依赖 `color` 与 `stringx`；
`progressbar` 依赖 `speedometer` 与 `stringx`。

## 包归属与命名

### 目录归属

- **根路径扁平化**：所有库包直接位于模块根下；只为「同一领域的多个实现」保留一层命名空间（`configs/`、`cryptos/`、`maths/`、`nets/`、`scheduler/`、`structures/`、`harness/`）。组件类包（缓存、池、限流、进度条、日志、颜色、NMEA、流处理）与工具类包同等对待，直接挂在根下，避免「工具 vs 组件」的归属争议。
- **单主题工具包不归组**：`logger` / `color` / `progressbar` / `speedometer` / `pool` / `ratelimiter`各自是独立领域（日志、着色、进度展示、速率统计、并发池、限流），互不依赖、也很少同时被同一个调用方导入，因此不为它们建 `toolkit/`、`tools/` 之类的通用桶：目录名只说明「这些是工具」而不说明「是什么领域」，反而让调用方需要记忆包被分到哪个桶。它们的性质由包名规则表达（见下文包名约定）。
- **只在领域真实存在时分层**：若某一天出现真正同领域、且成组演进的能力（如终端输出类），应新建领域目录（如 `console/`）而不是通用桶。
- **`structures`**：纯内存、无 IO、无后台协程的数据结构（含 FSM）。
- **`maths`**：纯计算与统计（含 `percentile`、向量、地理与机器学习）。
- **`scheduler`**：会启动事件循环、依赖时间源的主动调度器。
- **`harness`**：不参与对外 API 的开发辅助；`assetpath` 是可导入的库包，`mesh_demo` 仅是 `main`。
- 不新增 `common/`、`utils/`、`core/` 等语义不明的目录。

### 包名约定

| 规则 | 适用包 | 说明 |
|------|--------|------|
| 必须是实体对象名词 | 全部包 | 包名是一个具体对象（`cache`、`pool`、`scheduler`、`queue`、`timewheel`），不是动作或过程；**禁止动名词**（`scheduling` → `scheduler`）、禁止抽象容器名（`util`、`common`、`helper`） |
| 标准库增强包以 `x` 结尾 | `stringx`、`timex`、`filex`、`smtpx`、`sshx`、`rsax` | 语义上「对标准库/官方协议的增强封装」，`x` 表示扩展而非替代 |
| 同一领域的多个实现以 `s` 结尾 | `maths`、`nets`、`cryptos`、`configs`、`structures` | 这些目录下确实存在多个不同子包，故保留复数命名 |
| 单一职责包用单数名词 | `cache`、`color`、`logger`、`pool`、`nmea`、`progressbar`、`speedometer`、`stream`、`ratelimiter`、`version` | 包内只有一个主题，且不提供「多种并行实现」的对外语义 |
| 集合/夹具名词保持原形 | `harness`、`sshx/command` | `harness`（测试夹具）是领域名而非容器，加 `s` 反而失真 |

- 目录名与包名保持一致；若出现例外，导入方以包名为准。
- 不使用 `-` / `_` 等非标识符字符；不以 `go`、`std` 等可能与标准库混淆的前缀命名。
- 调度域采用「实体名」而非「域名词」：目录为 `scheduler/`，其下 `queue`（延迟队列）、`timewheel`（时间轮）都是可实例化的对象，导入路径为 `gitee.com/leminewx/gokit/scheduler/queue`。

## 设计原则

1. **标准库优先**：能用标准库解决的不再引入第三方依赖。当前仅 `serial`（串口）、`sshx`（SSH 协议）与 `progressbar`（终端探测）依赖 `golang.org/x/sys`、`golang.org/x/crypto`。
2. **泛型约束集中在根包**：`gokit.Int` / `Uint` / `Float` / `Complex` / `Number` 作为全库通用数值约束，避免各包重复声明；容器与算法以类型参数实现。
3. **接口 + 多实现**：字典、集合、布隆过滤器、流、编解码器、百分位均先定义接口，再提供多种实现，由调用方按并发强度与内存预算选择。
4. **零值可用优先**：构造函数集中校验参数（如 `pool.New` 保留默认 worker 数、`progressbar.New` 对 `total <= 0` 兜底），避免运行期出现 NaN 或除零。
5. **可关闭的后台协程**：所有启动后台协程的组件都提供停止手段（`pool.Release`、`TTLCache.StartAutoPurge` 返回的 `stop`、`timewheel.Stop`、`queue.Stop`、`mesh.Client.Close`、`progressbar.Finish`），不允许泄漏 goroutine。
6. **失败可观测**：网络与外部资源操作返回 `error`；需要上下文的提供 `...Context` 变体（`tcp.ExecuteContext`、`sshx.ExecuteContext`、`sshx.ExecuteStreamContext`）；需要静默失败的提供选项（`Command.StderrFilter` / `ExitCodeFilter`）。
7. **时间可注入**：依赖「当前时间 / 定时器」的组件通过 `timex.Clock` 取时（`cache.WithClock`），生产默认 `timex.NewSystemClock()`，测试注入 `timex.Manual` 后用 `Advance` 驱动，不依赖 `time.Sleep`。
8. **统一错误模型**：见下文「错误约定」。
9. **选项式构造**：`NewXxx(args..., opts...)` + `With<属性>` 选项，零值选项忽略非法输入，构造函数内部补齐默认值；新增可变参数不破坏旧调用。

## 代码规范

### 参考基线

本库以以下公开规范为基线：

- **Effective Go**（`go.dev/doc/effective_go`）
- **Go Code Review Comments**（`go.dev/wiki/CodeReviewComments`）
- **Google Go Style Guide**（`google.github.io/styleguide/go`）

在上述基线之上，本库有两条**强制偏离**，以团队规范优先：

1. **常量使用 `UPPER_SNAKE_CASE`**（Go 基线建议 MixedCaps）；
2. **私有常量/包级私有标识符以 `_` 开头**，用于一眼区分导出与私有。

带来的影响与应对：`golint` / `revive` 的 `var-naming` 与 `staticcheck` 的 `ST1003`（“不要用下划线命名”）
会与本规则冲突。当前门禁（`staticcheck ./...`、`make lint`）默认未触发 `ST1003`；若后续启用
`stylecheck` / `revive`，须在 `.golangci.yml` 中对该规则做排除，**不得为迎合 linter 改动命名**。

### 文件组织

- **一文件一主题**：`xxx.go` 与 `xxx_test.go` 一一对应；写在别处的用例必须抽回同名测试文件。
- **`errors.go`**：会返回可判别错误的包各一个，集中定义 `_ERROR_PREFIX` 与 `Err*` 哨兵。
- **`option.go`**：`With<属性>` 选项集中放置；**`doc.go`**：只放包注释（亦可写在主文件顶部）。
- **平台/构建约束用文件名后缀**：`*_windows.go`、`*_unix.go`、`termios_linux.go`；
  集成测试文件加 `//go:build integration`。
- 文件名全小写，必要时用下划线分词（`leaky_relu_activator.go`、`net_env_test.go`），
  不强行把导出的驼峰名映射到文件名。
- 示例程序放 `harness/` 下的独立 `main` 包，不放进库包目录。

### 类型命名

- 导出类型用名词：`Pool`、`Limiter`、`Progressbar`、`Command`、`Snapshot`；未导出类型小写开头。
- 单方法接口用 `-er`：`Percentiler`、`Activator`、`Projector`、`Codec`。
- 泛型参数：元素 `T`、键 `K`、值 `V`、结果 `R`。

### 函数与方法

- 导出函数/方法首词必须是**动词原形**：`Load`、`Save`、`Get`、`Set`、`Add`、`Delete`、`Execute`、`Calculate`、`ConvertTo`、`Normalize`、`Extract`、`Resolve`。
- 禁止第三人称单数（`Matches`、`NeedsRehash`、`Contains`）与名词开头（`Any`、`Head`、`Tail`）：布尔判定用 `Match` / `IsRehashNeeded` / `IsXxx` / `HasXxx`，取前 N 个元素用 `TakeFirst` / `TakeLast`。
- 判定类统一 `Is*` / `Has*`（`IsClosed`、`IsEmpty`、`HasAny`、`HasAll`）；取值类统一 `Get*`（`GetLength`、`GetStats`、`CPUTimes.GetTotal`）；转换类 `ConvertTo*`；计算类 `Calculate*`。
- 构造函数：`New(...)` / `NewXxx(...)`；需要注入依赖或可选项时用 `opts ...Option`。
- 可能阻塞或耗时的调用提供 `...Context` 变体（`ExecuteContext`、`ExecuteStreamContext`）。
- **例外只有两类**，其余全部适用上述规则：
  1. **实现标准库/第三方接口时，方法名与接口定义完全一致且不得改动**（改了就不再实现该接口）：
     `error` 的 `Error`、
     `fmt.Stringer` 的 `String`、
     `io.Reader`/`io.Writer`/`io.Closer` 的 `Read`/`Write`/`Close`、
     `net.Conn` 的 `LocalAddr`/`RemoteAddr`/`SetDeadline`/`SetReadDeadline`/`SetWriteDeadline`、
     `net.Addr` 的 `Network`、
     `net.Error` 的 `Timeout`/`Temporary`、
     `sort.Interface` / `heap.Interface` 的 `Len`/`Less`/`Swap`/`Push`/`Pop`、
     `slog.Handler` 的 `Enabled`/`Handle`/`WithAttrs`/`WithGroup`、
     `slog.Leveler` 的 `Level`。
  2. **日志级别方法**：`Debug` / `Info` / `Warn` / `Error`，保持与 `log` / `slog` 生态一致。
- 不属于上两类的名字，即使看上去像惯例，也必须改造成规则内的形态。本库自己的接口同样不享有豁免：时间源的 `Now` → `GetNow`、嘀嗒器的 `Chan` → `GetChannel`、TCP 客户端的地址访问器`LocalAddr`/`RemoteAddr` → `GetLocalAddr`/`GetRemoteAddr`（该类型并未实现 `net.Conn`）、位集的 `Len`/`Count` → `GetLength`/`GetCount`、百分位与进度条的 `Count` → `GetCount`。
- `String` / `Read` / `Write` / `Close` / `Swap` / `Handle` / `Must<Xxx>` 本身就是动词形态，符合规则，不是「例外」；真正的例外只有被接口钉死的方法名（`Error`、`Len`、`Less`、`Enabled`、`LocalAddr` 等）与日志级别。
- 实现接口的类型在类型定义旁补编译期断言 `var _ Xxx = (*T)(nil)`（`net.Conn`、`net.Addr`、`net.Error`、`heap.Interface`、`slog.Handler`、本包的 `Clock`/`Ticker` 均已就位）：既防漏实现，又标明这组方法名来自接口而非自拟。
- 未导出函数/方法不强制首词为动词（如 `loopGossip`、`firstLine`），但同一包内应保持一致。

### 常量

- **导出常量**：`UPPER_SNAKE_CASE`，例：`STYLE_BOLD`、`FG_BLACK`、`TYPE_BYTES`、`MESSAGE_TYPE_PUB`、`QUALITY_INVALID`、`FLOW_CONTROL_NONE`、`DEFAULT_COST`。
- **私有常量**：`_` + `UPPER_SNAKE_CASE`，例：`_ERROR_PREFIX`、`_DEFAULT_PERM`、`_TEST_TICK`、`_MAX_POOLED_BUILDER_CAP`、`_DEFAULT_MAX_MESSAGE_SIZE`。
- **词边界拆分规则**（`CamelCase` → `UPPER_SNAKE`）：
  - 按大写字母边界拆分：`MessageTypePub` → `MESSAGE_TYPE_PUB`；
  - 连续缩写作为一个词：`HTMLHeadTemplate` → `HTML_HEAD_TEMPLATE`、`NameReLU` → `NAME_RELU`、`QualityRTKFixed` → `QUALITY_RTK_FIXED`、`SystemMultiGNSS` → `SYSTEM_MULTI_GNSS`；
  - 官方专名整体保留：`WGS84`、`NAD83`、`CGCS2000`、`GRS80`、`IAG1975`；名称+年份拆开：`CRSXian1980` → `CRS_XIAN_1980`、`CRSBeijing1954` → `CRS_BEIJING_1954`；
  - 编号与缩写用下划线分隔：`EPSG3857` → `EPSG_3857`；数字粘连在前一个词：`demoSeedAddr1` → `_DEMO_SEED_ADDR1`。
- **iota 枚举**：同一 `const` 块内共享前缀与词序（`STYLE_*`、`FG_*`、`BG_*`、`MESSAGE_TYPE_*`、`QUALITY_*`、`MODE_*`、`FLOW_CONTROL_*`）。
- **错误哨兵**：`Err*` 是包级变量而非常量，沿用 MixedCaps（`ErrInvalidCIDR`、`ErrShapeNotMatch`），与标准库和 `errors.Is` 习惯一致。
- **唯一固定常量**：每个 `errors.go` 的 `_ERROR_PREFIX`，不可改名、不可导出。

### 变量

- 导出变量用 `CamelCase`（`color.NoColor`）。
- 包级私有变量用 `camelCase`（`noColor`、`versionCore`、`writerMu`），**不加** `_` 前缀——`_` 前缀只用于私有常量，便于区分「常量 vs 变量」。
- 局部变量与参数用 `camelCase`，短作用域可用单字母（`i`、`k`、`v`）。

### 接收者

- 方法接收者统一用 `own`；禁止 `this`、`self`、`me` 等。

### 注释与文档

- 每个包必须有包注释：`// Package xxx ...`，紧随 `package` 子句之前，说明职责、边界与关键约束。
- 导出符号的注释以符号名开头，说明「做什么、何时用、有何副作用」，默认值、单位、返回值必须写准。
- 注释必须与实现一致：默认值、单位（IEC 的 `KiB/MiB`）、是否可链式调用、是否并发安全。
- 禁止出现内部项目名、内网地址、内部层次术语（算法层 / 服务层 / 业务侧）；库是独立公共组件，描述只依赖公开语义。
- 不在注释、日志、错误文案中出现口令、密钥、令牌等敏感值。
- 不留无主 `TODO`；确需记录时写 `// TODO(<包名>): 说明`。

### 测试

- 表驱动 + `t.Run` 子用例；公共准备逻辑用 `t.Helper()`；资源清理用 `t.Cleanup`。
- 可并行用例声明 `t.Parallel()`；禁止依赖执行顺序。
- 默认用例不依赖网络、设备、外部账号；此类用例加 `//go:build integration`，参数从环境变量读取，缺失时 `t.Skip`（`GOKIT_TCP_*`、`GOKIT_SERIAL_PORT`、`GOKIT_SSH_*`、`SMTPX_*`、`GOKIT_NET_INTERFACE`）。
- 断言只用标准库 `testing`，不引入第三方断言库；时间相关用例注入 `timex.Manual` 而非 `time.Sleep`。

### 格式化与门禁

- 全仓 `gofmt`，行尾统一 LF；`make fmt-check` 作为硬门禁。
- 质量门禁：`make check`（格式 + `go vet` + `go build` + `go test`），另有 `make race` / `make lint`。
- 多包命令统一 `-p 1`，避免并发编译对低配机器与 CI 造成资源抖动。

## 错误约定

每个会返回可判别错误的包维护一个 `errors.go`：

```go
const _ERROR_PREFIX = "gokit.nets.sshx: "

var (
	// ErrInvalidCIDR 表示 CIDR 文本非法
	ErrInvalidCIDR = errors.New(_ERROR_PREFIX + "invalid CIDR")
)

// wrapError 在保留哨兵错误的前提下附加上下文，便于 errors.Is 判定与日志定位
func wrapError(sentinel error, format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
```

- **前缀**：`gokit.<包路径点分>`，与导入路径后缀一致（如 `gokit.nets.sshx.command: `）。
- **文案**：全小写、无句号、无多余空白，只用三种句式：
  - `failed to <动词> <对象>`（操作失败，如 `failed to parse output`）
  - `<主体> is|are <状态>` / `<主体> <形容词>`（状态非法，如 `key is nil`、`invalid CIDR`）
  - `<主体> not found`（查找为空，如 `peer not found`）
  不允许 `X is found` / `no data is collected` 之类的被动语态混用，统一为 `no valid IP address found`、
  `no data collected`。
- **包装**：一律 `wrapError(sentinel, "...: %w", err)`，禁止 `fmt.Errorf("%w: %w", wrapError(...), err)`；调用方用 `errors.Is` / `errors.As` 判定，上层不需要解析文案。
- **缩写大小写**：正文中协议名保持大写（`TLS`、`SSH`、`SMTP`、`GGA`、`RMC`、`CIDR`），句首仍小写（`failed to connect to SSH server`）。

## 并发安全矩阵

| 包 / 类型 | 并发安全 | 说明 |
|-----------|----------|------|
| `cache.LRUCache` | 是 | `sync.Mutex`；淘汰回调在锁外执行 |
| `cache.TTLCache` | 是 | `sync.RWMutex`，过期清理为懒执行；回调在锁外执行 |
| `pool.Pool` | 是 | channel 任务队列 + 原子关闭标记 |
| `ratelimiter.Limiter` | 是 | 内部互斥保护令牌桶 |
| `progressbar.Progressbar` | 是 | `sync.RWMutex`，渲染 100ms 节流 |
| `speedometer` | 是 | 环形桶 + `sync.Mutex` |
| `stream` | 否（需单一消费者） | 每条流应及时消费；`FanOut` 子流不能阻塞 |
| `structures/dict` | 按实现 | `UnsafeDict` 否；`SyncDict` / `SafeDict` / `ShardSafeDict` 是 |
| `structures/set` | 按实现 | `UnsafeSet` 否；`SyncSet` / `SafeSet` 是 |
| `structures/bloom` | 可选 | `Config.Concurrent = true` 时启用读锁；位数组本身为原子 |
| `structures/bitset` | 否 | 由调用方串行化 |
| `structures/fsm` | 否 | 同一状态机实例需串行 `Trigger` |
| `scheduler/queue.DelayQueue` | 是 | `heapLock` + `sync.Map` |
| `scheduler/timewheel.SingleWheel` | 是 | `sync.Map` + 原子状态，任务惰性删除 |
| `nets/tcp.Client` | 是 | 读写加锁，`Execute` 串行化请求/响应；`ExecuteContext` 可取消 |
| `nets/smtpx.Client` | 是 | 无内部可变状态，可并发发送（每次调用独立建连） |
| `nets/sshx.Client` | 部分 | `Execute` 可并发；`SetReconnect` 不要并发调用 |
| `nets/mesh.Client` | 是 | 成员表、消息总线、pending 请求均加锁 |

## 错误与超时约定

- 构造函数返回 `(*T, error)`，参数非法时立即失败；运行期错误通过方法返回。
- 超时统一使用 `time.Duration` 字段或选项，默认值集中在各包的默认配置中（Mesh 见 `NewDefaultConfig`，SSH / TCP / SMTP 见各自的 `WithTimeout` / `With*Timeout`）。
- 请求类调用（`mesh.Client.Get` / `Query`、`tcp.ExecuteContext`）在超时或取消后返回错误，不阻塞调用方；`ExecuteContext` 在 `ctx` 取消时返回 `ctx.Err()`。
- 日志：库内不打印业务日志，调试输出仅在 `Debug` 开关打开时生效（`mesh.WithDebug`、`sshx.WithDebug`）；需要结构化日志时注入 `mesh.Logger` 或使用 `logger` 包。

## 目录与资产约定

- **测试数据唯一目录**：`harness/testdata`。用例通过 `assetpath.Resolve` 或相对路径访问；用例自己生成的密钥等文件必须在 `t.Cleanup` 中删除。
- **示例程序**：放在 `harness/` 下的独立 `main` 包，不放进 `examples/`，避免被当作库包引用。
- **文档**：库级说明放 `docs/`，包级说明写在包注释（`go doc` 可见）。
- **构建产物**：`*.exe` 等二进制不提交（见 `.gitignore`）。

## 构建与质量门禁

- 统一入口：`make check`（格式检查 + `go vet` + `go build` + `go test`）；另有 `make race`、`make cover`、`make lint`。
- 格式：全仓 `gofmt`，行尾统一 LF；`make fmt-check` 在 CI 中作为硬门禁。
- 静态检查：`.golangci.yml`（golangci-lint v2，`standard` 集合 + `bodyclose`、`errorlint`、`nilerr`、`unconvert` 等）。
- CI：按 `build`、`race`、`lint` 三个 job 组织，直接调用 `make build` / `make race` / `make lint`；Go 版本从 `go.mod` 读取，多包测试使用 `-p 1`。

## 兼容性

- 版本以 Git 标签与提交信息为准，当前为 `v1.2.0`（与根包常量 `gokit.VERSION` 一致）；遵循语义化版本：不兼容变更升 MAJOR，新增功能升 MINOR，修复升 PATCH，纯文档/测试/构建变更不单独发版。
- `gokit.VERSION` 与 Git 标签保持一致，均为三段号；依赖方按 `@v1.2.0` 或 `@latest` 拉取，升级前请阅读 `RELEASE.md`。
- 最低 Go 版本跟随 `go.mod`（当前 `1.27.0`）；使用新标准库 API 时同步更新该声明。