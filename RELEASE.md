# 变更记录

版本号遵循语义化版本 `MAJOR.MINOR.PATCH`，以 Git 标签与提交信息为准，与根包常量
`gokit.VERSION` 保持一致（当前 `v1.2.0`）。

> 发布标签：Go Modules 只接受三段语义化版本标签，推送时打 `v1.2.0`；依赖方按
> `go get gitee.com/leminewx/gokit@v1.2.0` 或 `@latest` 拉取。

## v1.2.0（泛型百分位入口支持 time.Duration）

- `maths/percentile` 新增输入约束 `Numeric`（`gokit.Number` 的全部数值类型与`time.Duration`），`TypedPercentiler` 及 `NewTypedExact` / `NewTypedTDigest` /`NewTypedNearestRank` 由 `[T gokit.Number]` 放宽为 `[T Numeric]`，可直接添加`time.Duration` 数据点；`Merge` 仍只接受同算法、同数值类型的实例。
- 精度边界：`time.Duration` 以纳秒计数，float64 可精确表示约 104 天，超出部分在 `Add` 时按最近偶数舍入；需要完整保留原始精度时应选用最近秩口径。
- 兼容性：约束放宽，既有以 `gokit.Number` 类型参数实例化的调用与 `Percentiler` 接口均不受影响。

## v1.1.1（百分位压测改用 b.Loop）

- `maths/percentile` 的基准测试改用 `b.Loop()`，由测试框架负责计时与防止循环体被优化，不改变库行为。

## v1.1.0（泛型百分位入口）

- `maths/percentile` 新增泛型输入包装 `TypedPercentiler[T gokit.Number]`：`NewTypedExact` /`NewTypedTDigest` / `NewTypedNearestRank` 分别对应三种口径，`Add(T)` 后内部统一按 float64计算，`CalculatePercentile` 返回 float64 分位值；`Merge` 只接受同算法、同数值类型的实例，`GetPercentiler` 可取回底层实例与既有 API 互操作。
- 精度边界：计算域为 float64，可精确表示的整数上限为 2^53；超出部分在 `Add` 时按最近偶数舍入，需要完整保留整数精度时应选用最近秩口径。
- 兼容性：`Percentiler` 接口与 `NewExact` / `NewTDigest` / `NewNearestRank` 保持不变，升级无需修改现有调用点。

## v1.0.0（正式首发）

首个公开发行版本，共 58 个包（含 `harness/` 下的测试辅助与演示命令）：单主题工具包直接位于模块根目录，多实现领域按
`maths/`、`nets/`、`configs/`、`cryptos/`、`structures/`、`scheduler/` 子域组织，导入路径统一为
`gitee.com/leminewx/gokit/<包名>`。`nets/sshx`、`nets/serial`、`cryptos/password` 与 `progressbar`
通过 `golang.org/x/crypto`、`golang.org/x/sys` 实现 SSH、串口与终端尺寸探测，其余包只依赖标准库；
除 `color` 依据 `NO_COLOR` / `TERM=dumb` 初始化默认着色开关外，包级状态均为惰性构建，每个包可独立引入。

### 根包与基础工具

- `gokit`：泛型数值约束 `Int` / `Uint` / `Float` / `Complex` / `Number`，`ListenQuitSignal`、
  `GetValueWithDefault`、`CreateEmptyFile`（传播 `Close` 错误）、`IsDir`，以及版本常量 `VERSION`。
- `stringx`：`ConvertToSnakeCase`、`HumanizeBytes`（1024 进制 IEC 单位）、`StripANSIString` /
  `StripANSIBytes`、`JoinInts` / `JoinUints`、`ParseNumbers[T]`（按目标位宽做溢出校验）、
  `GetBuilder` / `PutBuilder`（容量异常膨胀的实例直接丢弃）。
- `timex`：`Clock` / `Ticker` 接口，`NewSystemClock()` 与测试用 `Manual`（`Advance` / `AdvanceTo`），
  依赖时间的组件可按需注入。
- `filex`：`WriteFileAtomic` / `CopyFileAtomic`——同目录临时文件 → `Chmod` → `Sync` → `Close`
  → `Rename` → 目录 fsync，失败路径清理临时文件；复制时继承源文件权限。
- `version`：`Compare` / `Normalize` 版本号比较（缺失段视为 0、数字段按长度比较）；
  `Validate` / `Match` / `NormalizeExpr` / `Explain` 区间表达式（`>=1.2.0 <2.0.0`、`1.2.x` 通配、
  行内 AND / 行间 OR、全角与 `≥` `≤` 归一）。
- `harness/assetpath`：`GetDataDir` / `Resolve` / `ResolveOr` 从源码位置与工作目录向上定位
  `harness/testdata`，越界路径返回 `ErrOutsideRoot`。
- `logger`：基于 `log/slog` 的 `PrettyHandler`，彩色输出、属性分组与 `ReplaceAttr`。
- `color`：ANSI 颜色与文本样式，支持 24 位真彩与 `SetNoColor`。

### 通用组件

- `cache`：`LRUCache`（容量淘汰）与 `TTLCache`（过期 + 自动清理），支持 `WithClock` / `WithOnEvict`；
  `Peek` / `Contain` / `GetKeys` / `GetCapacity` / `GetDefaultTTL`；自动清理停止幂等。
- `pool`：固定 worker 协程池，panic 恢复与优雅关闭；`Release` 按「置关闭标记 → 取消 ctx →
  等待在途 `Submit` → 关闭任务通道 → 等待 worker」执行，不丢任务、不阻塞提交方。
- `ratelimiter`：令牌桶 `Allow` / `AllowN` / `Wait` / `WaitN` / `SetRate`；非法配置返回
  `ErrInvalidConfig`，不可满足的请求返回 `ErrUnattainable`，不永久阻塞。
- `progressbar`：自适应终端宽度，字节与计数两种模式；样式由 `Style [2]rune` 描述，默认 `StyleBarPoint`。
- `speedometer`：环形桶速率计。
- `stream`：泛型流式处理（过滤、变换、去重、采样、分流、JSON、Zip）；`FanOut` 为各子流构造
  独立元素，`ZipLeft` / `ZipLeftByKey` 在主流结束后取消读取协程。
- `nmea`：GGA / RMC 解析与定长格式化，磁偏角按「西偏为负」解析，`Parse → String → Parse` 字节等价。

### 配置与加密

- `configs/ini`：文件 / `io.Reader` / 字符串加载，`WriteTo` / `String` 写出，类型化 Getter
  （`GetInt` / `GetInt64` / `GetUint64` / `GetFloat64` / `GetBool` / `GetDuration` / `GetStrings`），
  `Unmarshal` / `Marshal` 结构体映射，JSON 互操作结果确定（节名与键名排序）。
- `cryptos/rsax`：RSA 密钥生成（最小 2048 位）、PEM / DER / 证书解析、OAEP 加解密、
  PKCS#1 v1.5 与 PSS 签名验签、私钥 PEM/Base64 导出；私钥落盘权限 0600，且先收紧权限再截断写入。
- `cryptos/password`：bcrypt 封装 `Hash` / `HashWithCost` / `Verify` / `Match` / `IsRehashNeeded`。

### 数学

- 统计与百分位：`Sum`、`CalculateMean`、`CalculateVariance`、`CalculateStandardDeviation`、
  `CalculateMedian`、`Count`、`CalculateRange`、`CalculateAbs`；`maths/percentile` 提供精确 R-7、
  T-Digest 近似与最近秩（`NewNearestRank`）三种口径。
- 地理：LLA / ECEF / ENU 坐标互转、Haversine 与 Vincenty 距离；GeoJSON 风格几何类型、
  线段 / 折线 / 多边形；Web Mercator、Lambert 等积与本地投影。
- 向量：`Vector1D` / `Vector2D` / `Vector3D` 及 `Vector` 接口，`CalculateDot` / `CalculateCross` /
  `Add` / `Subtract` / `GetNorm`，以及数组形式的楔积 `CalculateWedge`。
- 机器学习：欧氏距离 `CalculateEuclideanDistance`、KNN `SimpleModel`、ReLU / LeakyReLU /
  Sigmoid / Tanh 激活函数。
- 数值分析：`maths/root` 求根（二分 / 牛顿 / 弦截 / 不动点 + Aitken / Muller 复根）；
  `maths/quadrature` 积分（Romberg / Newton-Cotes / 自适应 / Gauss-Legendre，`ErrorEstimate`
  为收敛指示而非严格上界）；`maths/interpolator` 插值（Lagrange / Newton、三次样条自然 / 夹持 /
  周期 / 非扭结四种边界、分段 Hermite，越界默认拒绝并可显式选择外推策略）；`maths/ode`
  常微分方程（固定步长 RK4、自适应 RKF45，右端非有限值立即终止）。
- 信号与拟合：`maths/differentiator` 数值微分（非等距三点 / 五点 + Richardson / Savitzky-Golay）；
  `maths/filter` 滤波（滑动平均 / 中值 / Hampel / 指数平滑）；`maths/fitter` 加权最小二乘；
  `maths/optimizer` 极小化（Brent / Nelder-Mead）；`maths/spectrum` 频谱（基 2 FFT / IFFT /
  四种窗）；`maths/distribution` 概率分布（正态 / 均匀 / 指数 / 泊松）；`maths/sampler` 可注入
  随机源的采样与洗牌（同一个 `Sampler` 不可并发调用）。
- 其余子包：`maths/geography/datum`（大地基准面与椭球体）、`maths/geography/geometry/line`、
  `maths/geography/geometry/polygon`、`maths/matrix/vector`、`maths/biology/activator`、
  `maths/learning`、`maths/learning/knn`。

### 网络

- `nets`：`IsConnectionError`（超时与 DNS 解析错误不视为断连）、`GenerateIPV4s` /
  `GenerateIPV4sLimit`（默认上限 65536 个地址）、`GetNetPreferredIP`。
- `nets/codec`：原始 / 定长 / 分隔符 / EOF / 长度前缀 / uvarint / 自定义分帧；长度前缀类编解码器
  默认上限 64 MiB，超出返回 `ErrMessageTooLarge`。
- `nets/tcp`：并发安全 TCP 客户端，选项式构造、`ExecuteContext` 取消、重连与运行统计；
  `retries = 0` 表示关闭自动重连；地址访问器为 `GetLocalAddr` / `GetRemoteAddr`（不实现 `net.Conn`）。
- `nets/serial`：跨平台串口（Windows / Linux / macOS），实现 `net.Conn`；`Config.Validate`
  在 `Open` 前校验端口名、波特率、数据位、停止位、校验位、流控与读超时。
- `nets/smtpx`：文本 / HTML 邮件、附件、自定义头、STARTTLS、整体超时与 `SendRaw`；
  主题按 RFC 2047 编码；发件人、收件人、附件名与 `ContentType` 均做 CR/LF 校验，阻断邮件头与
  信封命令注入（`ErrHeaderInjection`）；IPv6 服务器地址使用 `net.JoinHostPort` 拼接。
- `nets/sshx`：命令执行、流式执行与 PTY 终端；主机密钥选项 `WithHostKeyCallback` /
  `WithKnownHosts` / `WithInsecureIgnoreHostKey`，会话与流式错误可沿 `errors.Is` 回溯；
  调试日志只记录参数长度、不输出内容。
- `nets/sshx/command`：单次会话采集主机名、内核、运行时长、负载、整机 / 逐核 CPU、
  内存与交换区、磁盘与网卡；兼容 GNU coreutils 与 BusyBox，CPU 核心编号允许稀疏、
  无 `MemAvailable` 时按 Free+Buffers+Cached 推算并标记 `AvailableEstimated`；
  `Snapshot.Format` / `WriteTo` 输出人类可读概览，超过 16 核时只输出 min/avg/max。
- `nets/mesh`：Gossip 成员发现、Pub/Sub、KV 与 RPC（协议与调优细节见
  [design/mesh.md](./docs/design/mesh.md)）。KV 遵循「读己之写」：`Put` 先写本地再广播，
  `Get` 本地优先、未命中再广播；去重键为 `(SenderID, SN, MessageType)`，序列号单调递增；
  成员视图只接受更新的 `NodeState.SN`，过期条目直接丢弃。

### 数据结构与调度

- `structures/dict`：`NewUnsafeDict` / `NewSafeDict`（`sync.RWMutex`）/ `NewSyncDict` / `NewShardSafeDict`
  四种实现，统一 `GetItems()`（`iter.Seq2`）与 `GetAll()`（`iter.Seq`）视图。
- `structures/set`：`NewUnsafeSet` / `NewSafeSet`（`sync.RWMutex`）/ `NewSyncSet`（`sync.Map`）。
- `structures/bitset`：位集合与并 / 交 / 差 / 补运算。
- `structures/bloom`：泛型布隆过滤器，可配假阳性率与并发；哈希实例数与最优哈希个数对齐。
- `structures/tree`：Trie，支持 `:` 与 `*` 路径参数。
- `structures/fsm`：泛型有限状态机，支持导入导出。
- `scheduler/queue`：基于最小堆的延迟队列；停止后 `Start` 为无操作，`AddTask` 与 `Stop`
  并发时不返回永远不执行的任务 ID。
- `scheduler/timewheel`：单级时间轮；事件循环与回调 WaitGroup 分离，`IsRunning` 报告运行状态，
  逾期重复任务按周期继续执行并在结束后移出任务表。

### 安全约束

- RSA 私钥文件权限固定 0600，最低密钥位数 2048；位数不足返回 `ErrInvalidKeyBits`。
- SMTP 邮件头、信封命令与附件字段均拒绝 CR/LF 注入。
- SSH 调试日志不输出命令参数与口令；`GetClient` 读锁保护，截断长度按 `int64` 比较。
- 测试夹具只使用文档专用地址（TEST-NET-1）与示例凭据，不含内部项目名与内网地址。

### 行为约定

- `nets/codec` 长度前缀类编解码器与 `nets.GenerateIPV4s` 均有默认上限，需要更大规模时
  显式传入上限参数。
- `ratelimiter.AllowN(n < 1)` 返回 `false`；`WaitN` 对不可满足请求快速失败。
- `stringx.HumanizeBytes` 以 1024 为进制，单位固定为 `B` / `KiB` / `MiB` / `GiB` / `TiB` / `PiB`。
- `cache` TTL 的 `Put` 在 `ttl <= 0` 时使用默认 TTL，默认 TTL 为 0 则永不过期。
- 除显式 `Must*` 版本外，公开 API 在参数非法时返回可判别错误而不 panic；入参切片不被就地改写。

### 错误模型与命名规范

- 错误哨兵统一为 `gokit.<点分路径>: <小写描述>`，以 `_ERROR_PREFIX` 为前缀、`errors.New` 创建，
  包装使用 `%w`，调用方可沿 `errors.Is` 判断；文案采用 `failed to <verb> <object>` /
  `<subject> is|are <state>` / `<subject> not found` 三句式，全小写、无句号，协议缩写大写。
- 导出函数 / 方法首词为动词原形（`Get*` / `Is*` / `Has*` / `Calculate*` / `ConvertTo*` / `New*`），
  禁止第三人称单数与名词开头；实现标准库 / 第三方接口而被钉死的方法名与日志级别方法保持接口原名。
- 导出类型用名词；单方法接口用 `-er`；泛型参数为 `T` / `K` / `V` / `R`。
- 导出常量 `UPPER_SNAKE_CASE`，私有常量 `_` + `UPPER_SNAKE_CASE`；`Err*` 哨兵是变量，按 MixedCaps 命名。
- 实现接口的类型在定义旁提供编译期断言 `var _ Xxx = (*T)(nil)`；每个包都有包注释，多文件包通常放在 `doc.go`。
- 完整规则见 [docs/architecture.md](./docs/architecture.md) 的「代码规范」章节。

### 测试与质量门禁

- 设备、外部服务与指定网卡依赖的用例统一加 `//go:build integration`，默认 `go test ./...`
  不依赖真实环境；集成用例通过 `GOKIT_TCP_*` / `GOKIT_SERIAL_PORT` / `GOKIT_SSH_*` /
  `SMTPX_*` / `GOKIT_NET_INTERFACE` 注入目标，变量缺失时自动跳过。
- 时间相关用例通过 `timex.Manual` 推进，不使用 `time.Sleep`；断言只用标准库 `testing`。
- 测试数据统一放在 `harness/testdata`；`nets/sshx/command` 的样本驱动用例语句覆盖率 84.5%，
  并已在 Ubuntu 22.04（96 核 x86_64）、嵌入式 Linux 3.10 + BusyBox 1.24（armv7l）与
  Jetson Linux 5.15（8 核 aarch64）完成实机采集验证。
- 质量门禁由 `Makefile` 提供：`make fmt` / `fmt-check` / `vet` / `build` / `test` / `race` /
  `cover` / `lint` / `check`，`make lint` 使用 `.golangci.yml`（golangci-lint v2），可直接接入 CI。

### 文档与示例

- 文档入口为根 `README.md`：`docs/start.md`（安装与包地图）、`docs/architecture.md`（分层、
  命名、错误约定与并发安全矩阵）、`docs/testing.md`（测试分层与集成测试清单）、
  `docs/api/`（`base` / `configs` / `cryptos` / `maths` / `nets` / `scheduler` / `structures`
  七份 API 参考）、`docs/design/mesh.md`（Mesh 协议与调优）。
- 三节点集群演示程序 `harness/mesh_demo`，参数为 `-id` / `-addr` / `-seeds` / `-max-peers`。
- 全仓 Go 源码统一 LF 行尾并通过 `gofmt`。