# 快速开始

## 环境要求

| 项目 | 要求 |
|------|------|
| Go | 1.27.0 及以上（`go.mod` 声明值） |
| 外部依赖 | `golang.org/x/crypto`、`golang.org/x/sys`（仅 SSH、串口等平台模块使用） |
| 平台 | Windows / Linux / macOS；串口与终端相关能力依赖平台实现 |

## 安装

```bash
go get gitee.com/leminewx/gokit@latest
```

按需导入子包，不存在全局初始化，也没有必须调用的 `Init`。

## 五分钟示例

### 1. 根包与 stringx 基础工具

```go
import (
    "gitee.com/leminewx/gokit"
    "gitee.com/leminewx/gokit/stringx"
)

stringx.HumanizeBytes(1536)              // "1.50 KiB"（1024 进制，IEC 单位）
stringx.ConvertToSnakeCase("HelloWorld") // "hello_world"
gokit.GetValueWithDefault(30)            // 30（无候选值时返回默认值）
gokit.GetValueWithDefault(30, 60)        // 60（有候选值时返回第一个）
```

### 2. Goroutine 池

```go
p := pool.New(pool.WithWorkers(4), pool.WithQueueSize(128))
defer p.Release()

_ = p.Submit(func(ctx context.Context) error {
    return doWork(ctx)
})
```

### 3. 缓存

```go
lru := cache.NewLRUCache[string, int](256)
lru.Put("k", 1)
v, ok := lru.Get("k")

ttl := cache.NewTTLCache[string, string](time.Minute,
    cache.WithClock(timex.NewSystemClock()),
    cache.WithOnEvict(func(key, value string) { fmt.Println("过期:", key) }),
)
ttl.Put("token", "abc", 0) // 0 表示使用默认 TTL
```

### 4. 集合与字典

```go
s := set.NewSafeSet("a", "b")
s.Add("c")
fmt.Println(s.GetSlice()) // 顺序不保证

d := dict.NewSyncDict[string, int]()
d.Set("hits", 1)
```

### 5. Mesh 集群

```go
c, err := mesh.NewClient(mesh.ClientOptions{
    ID:        "node-a",
    Addr:      ":9000",
    SeedAddrs: []string{":9001"},
})
if err != nil {
    return err
}
defer c.Close()

_ = c.Subscribe("demo/:name", func(msg mesh.Message, pathValues map[string]string) {
    fmt.Println("收到:", msg.Path, pathValues["name"], msg.Value)
})
_ = c.Publish("demo/hello", "world")

_ = c.Put("config/version", 3)
if v, err := c.Get("config/version"); err == nil {
    fmt.Println(v) // 经 JSON 传输后为 float64
}
```

Mesh 的完整语义（Gossip、去重、KV、RPC）见 [design/mesh.md](./design/mesh.md)。

## 完整包地图

`go list ./...` 共 58 个包。

### 根与基础工具

| 包 | 说明 |
|----|------|
| `gokit` | 泛型数值约束（`Int`/`Uint`/`Float`/`Complex`/`Number`）与通用工具 |
| `stringx` | stdlib 增强：命名转换、ANSI 剪除、数值切片拼接与解析、`strings.Builder` 池 |
| `timex` | stdlib 增强：可注入时间源，`NewSystemClock()` 与测试用 `Manual` |
| `filex` | stdlib 增强：`WriteFileAtomic` / `CopyFileAtomic`（同目录临时文件 + `Sync` + `Rename`） |
| `version` | 版本号解析与比较（`Compare` / `Normalize`）、区间表达式匹配（`Match` / `Explain`） |
| `logger` | 基于 `log/slog` 的 `PrettyHandler` 与 `Logger` |
| `color` | ANSI 颜色与文本样式，支持 24 位真彩 |
| `harness/assetpath` | 定位仓库内数据目录：`GetDataDir` / `Resolve` / `ResolveOr`，适配测试与示例 |

### 通用组件

| 包 | 说明 |
|----|------|
| `cache` | `LRUCache`、`TTLCache`，支持注入时钟与淘汰回调 |
| `pool` | Goroutine 池 |
| `ratelimiter` | 令牌桶限流 |
| `progressbar` | 终端进度条 |
| `speedometer` | 环形桶速率计 |
| `stream` | 泛型流式处理（链式算子） |
| `nmea` | NMEA 0183：GGA / RMC 解析与格式化 |

### 配置与加密

| 包 | 说明 |
|----|------|
| `configs/ini` | INI / JSON 配置加载、类型化访问、序列化与保存 |
| `cryptos/rsax` | RSA 密钥生成、PEM/DER 解析、OAEP 加解密与签名验签 |
| `cryptos/password` | bcrypt 口令散列与校验（`Hash` / `Verify` / `Match` / `IsRehashNeeded`） |

### 数学

| 包 | 说明 |
|----|------|
| `maths` | 求和、均值、方差、中位数、频次、极值、绝对值 |
| `maths/percentile` | 精确百分位（R-7）、T-Digest 近似与最近秩（`NewNearestRank`） |
| `maths/geography` | LLA / ECEF / ENU 坐标与 Haversine、Vincenty 距离 |
| `maths/geography/datum` | 大地基准面与椭球体位标志（`WGS84` 等） |
| `maths/geography/geometry` | GeoJSON 风格几何类型 |
| `maths/geography/geometry/line` | 线段、折线与最近点查询 |
| `maths/geography/geometry/polygon` | 多边形与射线法包含判断 |
| `maths/geography/projection` | 平面投影（Web Mercator、Lambert 等积、本地投影）与平面几何运算 |
| `maths/matrix/vector` | 1D/2D/3D 向量与点积、叉积、模 |
| `maths/biology/activator` | 激活函数 ReLU / LeakyReLU / Sigmoid / Tanh / Softmax |
| `maths/learning` | 欧氏距离与距离函数类型 |
| `maths/learning/knn` | K-近邻 `SimpleModel` |
| `maths/root` | 求根：二分法、牛顿法、弦截法、不动点迭代（Aitken 加速）、Muller 复根 |
| `maths/quadrature` | 数值积分：Romberg、复化 / 自适应 Newton-Cotes、Gauss-Legendre |
| `maths/interpolator` | 插值：Lagrange / Newton 多项式、三次样条（自然 / 夹持 / 周期 / 非扭结）、分段 Hermite |
| `maths/ode` | 常微分方程初值问题：固定步长 RK4、自适应 Runge-Kutta-Fehlberg 4(5) |
| `maths/differentiator` | 数值微分：非等距三点、五点模板 + Richardson、Savitzky-Golay 平滑与导数 |
| `maths/filter` | 序列滤波：滑动平均 / 中值、Hampel 离群点剔除、一阶指数平滑 |
| `maths/fitter` | 加权最小二乘多项式拟合（Householder QR）与 R² |
| `maths/optimizer` | 极小化：Brent 一维搜索、Nelder-Mead 单纯形 |
| `maths/distribution` | 概率分布：正态 / 均匀 / 指数 / 泊松的 PDF / PMF、CDF 与分位数 |
| `maths/spectrum` | 频谱：基 2 FFT / IFFT、四种窗函数、单边幅度谱 |
| `maths/sampler` | 可注入随机源的采样：均匀 / 正态 / 指数 / 泊松与洗牌 |

### 网络

| 包 | 说明 |
|----|------|
| `nets` | 连接错误判定、CIDR 网段枚举、按网卡优选本机 IP |
| `nets/codec` | TCP 分帧编解码（原始、定长、分隔符、EOF、长度前缀、uvarint、自定义） |
| `nets/mesh` | Gossip 集群、Pub/Sub、KV、RPC Query |
| `nets/serial` | 跨平台串口，实现 `net.Conn` |
| `nets/smtpx` | SMTP 文本/HTML 邮件，附件、STARTTLS 与整体超时 |
| `nets/sshx` | SSH 命令执行、流式执行、PTY 终端 |
| `nets/sshx/command` | 单会话采集 CPU / 内存 / 磁盘 / 网卡指标，兼容 GNU coreutils 与 BusyBox |
| `nets/tcp` | 并发安全的 TCP 客户端，支持选项、取消与运行统计 |

### 数据结构

| 包 | 说明 |
|----|------|
| `structures/bitset` | 位集合 |
| `structures/bloom` | 泛型布隆过滤器 |
| `structures/dict` | 字典四种实现（非安全 / 读写锁 / 安全 / 分片） |
| `structures/fsm` | 泛型有限状态机 |
| `structures/set` | 集合三种实现（非安全 / 安全 / `sync.Map`） |
| `structures/tree` | Trie 前缀树（支持路径参数） |

### 调度

| 包 | 说明 |
|----|------|
| `scheduler/queue` | 基于最小堆的延迟队列 |
| `scheduler/timewheel` | 单级时间轮 |

### 非库目录

| 目录 | 说明 |
|------|------|
| `harness/mesh_demo` | 三节点集群演示程序（`main` 包） |
| `harness/testdata` | 全仓唯一测试数据目录 |

## 运行示例程序

```bash
# 节点 1（种子）
go run ./harness/mesh_demo -id node-a -addr :9000

# 节点 2（加入节点 1）
go run ./harness/mesh_demo -id node-b -addr :9001 -seeds :9000

# 节点 3
go run ./harness/mesh_demo -id node-c -addr :9002 -seeds :9000
```

参数：`-id`（节点标识）、`-addr`（监听地址）、`-seeds`（逗号分隔种子地址）、`-max-peers`（默认 10）。

## 下一步

- 了解设计约束与并发语义：[architecture.md](./architecture.md)
- 运行与编写测试：[testing.md](./testing.md)