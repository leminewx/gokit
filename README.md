# gokit

[![Go Version](https://img.shields.io/badge/Go-1.27.0-blue.svg)](https://go.dev/)
[![Version](https://img.shields.io/badge/version-v1.1.0-green.svg)]()
[![License](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)

gokit 是一个综合性 Go 工具库，涵盖**数据结构、数学计算、网络通信、配置与加密、通用组件**，
提供生产级、类型安全、以标准库为主的工具集；仅 SSH、串口等平台模块引入
`golang.org/x/crypto` 与 `golang.org/x/sys`。

- 泛型优先：数值约束集中在根包，容器与算法均以类型参数实现。
- 按需导入：没有全局初始化，每个子包可独立使用。
- 测试友好：默认测试不依赖网络、设备或外部账号。

## 文档

| 文档 | 内容 |
|------|------|
| [docs/start.md](./docs/start.md) | 安装、最小示例、完整包地图 |
| [docs/architecture.md](./docs/architecture.md) | 分层、设计原则、并发安全矩阵、命名与错误约定 |
| [docs/testing.md](./docs/testing.md) | 测试分层、`harness/testdata` 约定、集成测试清单 |
| [docs/api/](./docs/api/base.md) | API 参考：`base` / `configs` / `cryptos` / `maths` / `nets` / `scheduler` / `structures` 七份文档 |
| [docs/design/mesh.md](./docs/design/mesh.md) | Mesh 集群协议、Gossip、去重与 KV 语义 |
| [RELEASE.md](./RELEASE.md) | 变更记录 |
| [ISSUES.md](./ISSUES.md) | 已知问题与限制 |

## 安装

```bash
go get gitee.com/leminewx/gokit@latest
```

## 版本

| 项目 | 值 |
|------|-----|
| 版本 | v1.1.0（新增 maths/percentile 泛型输入入口，向后兼容；v1.0.0 首发记录见 [RELEASE.md](./RELEASE.md)，以 Git 标签/提交为准） |
| Go 最低版本 | 1.27.0 |
| 外部依赖 | `golang.org/x/crypto`、`golang.org/x/sys` |

> **版本约定**：`gokit.VERSION` 与 Git 标签完全一致，均为三段语义化版本（当前 `v1.1.0`）；
> 依赖方按 `go get gitee.com/leminewx/gokit@v1.1.0` 或 `@latest` 拉取。

## 包概览

`go list ./...` 共 58 个包。

```
gokit/
├── gokit.go              # 根包：泛型数值约束 + 通用工具
├── stringx/              # 文本工具（stdlib 增强）：命名转换、ANSI 剪除、数值切片、Builder 池
├── timex/                # 时间源（stdlib 增强）：系统时钟与可手动推进的测试时钟
├── filex/                # 文件工具（stdlib 增强）：原子写入、原子复制（掉电安全）
├── version/              # 版本工具：版本号比较与区间表达式匹配
├── logger/               # 基于 log/slog 的彩色文本 Handler
├── color/                # ANSI 颜色与文本样式
├── cache/                # LRU / TTL 缓存
├── pool/                 # Goroutine 池
├── ratelimiter/          # 令牌桶限流
├── progressbar/          # 终端进度条
├── speedometer/          # 环形桶速率计
├── stream/               # 泛型流式处理
├── nmea/                 # NMEA 0183：GGA / RMC
├── configs/ini/          # 配置：INI / JSON 加载、类型化访问、映射与保存
├── cryptos/rsax/         # 加密：RSA 密钥、PEM/DER 解析、OAEP 加解密与签名
├── cryptos/password/     # 加密：bcrypt 密码摘要与校验
├── maths/                # 数学：统计、地理坐标、投影、几何、向量、生物启发算法、KNN、数值分析
├── nets/                 # 网络：TCP、编解码、Mesh、串口、SMTP、SSH
├── scheduler/           # 调度：延迟队列、单级时间轮
├── structures/           # 结构：字典、集合、位集、布隆过滤器、Trie、FSM
├── harness/              # 开发辅助：assetpath 数据目录定位（库）、mesh_demo 示例程序、testdata
├── docs/                 # 完整文档
└── RELEASE.md / ISSUES.md
```

## 根包

```go
import "gitee.com/leminewx/gokit"
```

| 类型/函数 | 说明 |
|-----------|------|
| `Int` / `Uint` / `Float` / `Complex` / `Number` | 泛型数值类型约束，供全库使用 |
| `ListenQuitSignal(ctx, done)` | 监听 OS 信号（SIGHUP/SIGINT/SIGTERM/SIGQUIT）并执行清理回调 |
| `GetValueWithDefault(flt, values...)` | 有候选值时取第一个，否则返回默认值 |
| `CreateEmptyFile(filename)` | 创建空文件，自动创建中间目录 |
| `IsDir(filename)` | 判断是否为目录 |

命名转换与字节格式化在 `stringx` 包：`stringx.ConvertToSnakeCase`、`stringx.HumanizeBytes`；
文件落盘与版本比较在 `filex`、`version` 包。

## 基础工具

| 包 | 关键能力 |
|----|----------|
| `stringx` | `ConvertToSnakeCase`、`HumanizeBytes`（IEC 单位 KiB/MiB/…）、`StripANSIString` / `StripANSIBytes`、`JoinInts` / `JoinUints`、`ParseNumbers[T]`、`GetBuilder` / `PutBuilder` |
| `timex` | `NewSystemClock()`；测试用 `NewManual(start)` / `NewManualAt()`，通过 `Advance` / `AdvanceTo` 驱动 |
| `filex` | `WriteFileAtomic` / `CopyFileAtomic`：同目录临时文件 + Sync + 原子改名，失败不破坏旧文件 |
| `version` | `Compare` / `Normalize` 版本号比较；`Validate` / `Match` / `NormalizeExpr` / `Explain` 版本区间表达式（支持 `x` 通配与多区间并集） |
| `color` | ANSI 颜色/样式，支持 24 位真彩与 `SetNoColor` |
| `logger` | `NewPrettyHandler` + `NewLogger`，彩色输出、属性分组与 `ReplaceAttr` |
| `harness/assetpath` | `GetDataDir` 定位 `harness/testdata`；`Resolve` / `ResolveOr` 把相对路径安全映射到数据目录（拒绝 `..` 越界） |

```go
// 配置/产物落盘：进程被杀或掉电时不会留下半截文件
_ = filex.WriteFileAtomic("conf/app.ini", data, 0o644)

// 只接受 v6.4 到 v8.0 的实现版
ok, err := version.Match(">=v6.4.0 <v8.0.0", "app_V7.0.0.0_20260401")

// 示例服务定位演示数据，越界路径回落到 data.txt
path := assetpath.ResolveOr(r.URL.Path, "data.txt")
```

## 通用组件

| 包 | 关键能力 |
|----|----------|
| `cache` | `LRUCache`（容量淘汰）、`TTLCache`（过期 + 后台清理），支持注入时钟与淘汰回调 |
| `pool` | 固定 worker 协程池，panic 恢复与优雅关闭 |
| `ratelimiter` | 令牌桶限流：`Allow` / `AllowN` / `Wait` / `WaitN` / `SetRate` |
| `progressbar` | 终端进度条，自动适配宽度，支持字节/计数两种类型 |
| `speedometer` | 环形桶速率计 |
| `stream` | 泛型流式处理（过滤、变换、去重、采样、分流、JSON、Zip） |
| `nmea` | GGA / RMC 解析、`String` 定长格式化、转经纬高 |

```go
lru := cache.NewLRUCache[string, int](256)
lru.Put("k", 1)
v, ok := lru.Get("k")

ttl := cache.NewTTLCache[string, string](time.Minute,
    cache.WithClock(timex.NewSystemClock()),
    cache.WithOnEvict(func(key, value string) { fmt.Println("过期:", key) }),
)
ttl.Put("token", "abc", 0) // 0 表示使用默认 TTL

p := pool.New(pool.WithWorkers(4), pool.WithQueueSize(128))
defer p.Release()
_ = p.Submit(func(ctx context.Context) error { return doWork(ctx) })

limiter := ratelimiter.New(100, 10)
if limiter.Allow() { /* 放行一个请求 */ }
```

## configs / cryptos

```go
ini, _ := ini.LoadFromString("[server]\ntimeout = 3s\n")
timeout, _ := ini.GetDuration("server", "timeout")
var cfg Config
_ = ini.Unmarshal(&cfg)

pub, pri, _ := rsax.GenerateRSAKeyPair(2048)
cipher, _ := rsax.EncryptOAEPWithPublicKey(pub, []byte("secret"))
sig, _ := rsax.SignWithPrivateKey(pri, crypto.SHA256, []byte("payload"))
_ = rsax.VerifyWithPublicKey(pub, crypto.SHA256, []byte("payload"), sig)

hashed, _ := password.Hash([]byte("Cictci@2022"))
_ = password.Verify(hashed, []byte("Cictci@2022")) // 不匹配时返回 ErrPasswordMismatch
```

## maths

- 统计与百分位：`Sum`、`CalculateMean`、`CalculateVariance`、`CalculateStandardDeviation`、
  `CalculateMedian`、`Count`、`CalculateRange`、`CalculateAbs`，以及 `maths/percentile` 的
  精确 R-7、T-Digest 与最近秩（`NewNearestRank`）三种口径，并提供泛型输入包装
  `TypedPercentiler[T]`（内部按 float64 计算）
- 地理：LLA / ECEF / ENU 坐标、Haversine 与 Vincenty 距离、多边形周长
- 几何与投影：GeoJSON 风格类型、线段/折线/多边形、Web Mercator、Lambert 等积与本地投影
- 向量：1D/2D/3D、点积/叉积/模/楔积
- 机器学习：欧氏距离、KNN `SimpleModel`、ReLU / LeakyReLU / Sigmoid / Tanh / Softmax 激活函数
- 数值分析：`maths/root` 求根（二分 / 牛顿 / 弦截 / 不动点 + Aitken / Muller 复根）、
  `maths/quadrature` 积分（Romberg / Newton-Cotes / 自适应 / Gauss-Legendre）、
  `maths/interpolator` 插值（Lagrange / Newton / 三次样条四种边界 / Hermite）、
  `maths/ode` 常微分方程（固定步长 RK4 / 自适应 RKF45）
- 信号与拟合：`maths/differentiator` 数值微分（非等距三点 / 五点 + Richardson / Savitzky-Golay）、
  `maths/filter` 滤波（滑动平均 / 中值 / Hampel / 指数平滑）、`maths/fitter` 加权最小二乘、
  `maths/optimizer` 极小化（Brent / Nelder-Mead）、`maths/spectrum` 频谱（FFT / IFFT / 四种窗）、
  `maths/distribution` 概率分布（正态 / 均匀 / 指数 / 泊松）、`maths/sampler` 可注入随机源采样

## nets

| 包 | 关键能力 |
|----|----------|
| `nets` | `IsConnectionError`、`GenerateIPV4s`/`GenerateIPV4sLimit`、`GetNetPreferredIP` |
| `nets/tcp` | 并发安全 TCP 客户端，支持选项、`ExecuteContext` 取消、重连与运行统计 |
| `nets/codec` | 原始 / 定长 / 分隔符 / EOF / 长度前缀 / uvarint / 自定义分帧 |
| `nets/mesh` | Gossip 成员发现、Pub/Sub、KV、RPC，详见 [设计文档](./docs/design/mesh.md) |
| `nets/serial` | 跨平台串口，实现 `net.Conn` |
| `nets/smtpx` | SMTP 文本 / HTML 邮件，附件、STARTTLS、整体超时与内置模板 |
| `nets/sshx` | SSH 命令、流式执行、PTY 终端 |
| `nets/sshx/command` | 单会话采集 CPU / 内存 / 磁盘 / 网卡等指标，兼容 GNU coreutils 与 BusyBox |

## structures

| 包 | 关键能力 |
|----|----------|
| `structures/dict` | 非安全 / 读写锁 / 安全 / 分片四种字典 |
| `structures/set` | 非安全 / 安全 / `sync.Map` 三种集合 |
| `structures/bitset` | 位集合与并/交/差/补运算 |
| `structures/bloom` | 泛型布隆过滤器，可配假阳性率与并发 |
| `structures/tree` | Trie，支持 `:` 与 `*` 路径参数 |
| `structures/fsm` | 泛型有限状态机，支持导入导出 |

## scheduler

| 包 | 关键能力 |
|----|----------|
| `scheduler/queue` | 基于最小堆的延迟队列 |
| `scheduler/timewheel` | 单级时间轮 |

## 测试

```bash
make check        # 格式检查 + vet + build + test
make race         # 全量竞态检测
make cover        # 覆盖率报告
make lint         # golangci-lint（需本地安装）
```

等价的手动命令：

```bash
gofmt -l .        # 应无输出
go build -p 1 ./...
go vet -p 1 ./...
go test -race -count=1 -p 1 -timeout 900s ./...
```

默认测试不依赖真实设备、外部服务或公网。依赖设备/账号/指定网卡的用例统一使用
`//go:build integration` 隔离，并通过环境变量注入目标（缺失时自动跳过）：

```bash
# TCP 转串口设备
$env:GOKIT_TCP_HOST="192.0.2.30"; go test -tags integration ./nets/tcp
# SSH 采集
$env:GOKIT_SSH_HOST="192.0.2.2"; $env:GOKIT_SSH_USER="root"; $env:GOKIT_SSH_PASSWORD="***"
go test -tags integration ./nets/sshx/...
# 串口
$env:GOKIT_SERIAL_PORT="COM3"; go test -tags integration ./nets/serial
```

测试数据统一放在 `harness/testdata`；依赖时间的用例通过 `timex.Manual` 推进，
不使用 `time.Sleep`。质量门禁由 `make check` / `make race` / `make lint` 提供，
可直接接入任意 CI，详见 [docs/testing.md](./docs/testing.md)。

## 示例程序

`harness/mesh_demo` 演示三节点集群的启动、发现、Pub/Sub、KV 与 RPC：

```bash
go run ./harness/mesh_demo -id node-a -addr :9000
go run ./harness/mesh_demo -id node-b -addr :9001 -seeds :9000
go run ./harness/mesh_demo -id node-c -addr :9002 -seeds :9000
```

参数：`-id`、`-addr`、`-seeds`、`-max-peers`（默认 10）。

## 许可证

MIT License © 2023 leminewx