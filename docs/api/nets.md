# nets 网络 API

导入前缀：`gitee.com/leminewx/gokit/nets/...`

## nets

```go
ips, err := nets.GenerateIPV4s("192.168.1.0/24", true) // 跳过网络号与广播地址
if nets.IsConnectionError(err) { /* 连接已断开 */ }
ip, err := nets.GetNetPreferredIP("以太网", "192.168.1.") // 按网卡名与地址前缀优选
```

| 函数 | 说明 |
|------|------|
| `IsConnectionError(err error) bool` | 判断是否为连接断开类错误（可安全重连） |
| `GenerateIPV4s(cidr string, skipNetworkBroadcast bool) ([]string, error)` | 枚举 CIDR 内 IPv4，可选跳过网络号/广播地址 |
| `GetNetPreferredIP(name string, prefer ...string) (net.IP, error)` | 按网卡名称与地址前缀优选本机 IP |

`TestGetPublicIP` 依赖指定网卡与外网，已用 `integration` 标签隔离（见 testing.md）。

## nets/tcp

并发安全的 TCP 客户端，可挂接 `nets/codec` 分帧器，支持可取消调用与运行统计。

```go
c, err := tcp.NewClientWithOptions("127.0.0.1", 9000,
    tcp.WithDialTimeout(3*time.Second),
    tcp.WithReadTimeout(2*time.Second),
    tcp.WithWriteTimeout(2*time.Second),
    tcp.WithCodec(codec.NewDelimiterCodec('\n')),
    tcp.WithReconnect(time.Second, 5), // 断线重连
    tcp.WithKeepAlive(true),
)
if err != nil {
    return err
}
defer c.Close()

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
resp, err := c.ExecuteContext(ctx, []byte("PING\n"), 10*time.Millisecond)
stats := c.GetStats() // ReadBytes / WriteBytes / Reconnects / DialErrors
```

| 方法 / 选项 | 说明 |
|------|------|
| `NewClient(host string, port int, timeout ...time.Duration) (*Client, error)` | 基础构造；可选参数只设置拨号超时 |
| `NewClientWithOptions(host string, port int, options ...Option) (*Client, error)` | 选项式构造 |
| `WithDialTimeout` / `WithReadTimeout` / `WithWriteTimeout` | 拨号 / 读 / 写超时，默认 10s / 30s / 30s |
| `WithCodec(codec.Codec)` | 设置分帧编解码器，默认 `NewRawCodec(4096)` |
| `WithReconnect(interval, retries...)` | 启用重连；省略 `retries` 表示无限重连，`0` 表示关闭自动重连，`>0` 表示最大次数；客户端默认不重连 |
| `WithKeepAlive(bool)` | TCP 保活，默认开启 |
| `Write(data []byte) error` / `Read() ([]byte, error)` | 写入 / 读取一帧 |
| `Execute(req []byte, readRespInterval ...time.Duration) ([]byte, error)` | 写请求并读响应 |
| `ExecuteContext(ctx, req, readRespInterval...) ([]byte, error)` | 支持取消；`ctx` 的 deadline 与读写超时取较早者，取消时返回 `ctx.Err()` |
| `SetCodec` / `SetReadTimeout` / `SetWriteTimeout` / `SetReconnect` | 运行期调整配置 |
| `GetStats() Stats` | 累计读写字节、重连次数与拨号失败次数 |
| `IsClosed()` / `IsConnected()` / `GetLocalAddr()` / `GetRemoteAddr()` | 状态与连接地址 |
| `GetAddr() string` / `Close() error` | 远端地址 / 关闭 |

`Execute` 内部串行化请求/响应，避免并发调用交错；需要并发时请为每个 goroutine 使用独立客户端。

## nets/codec

`Codec` 接口只有两个方法：`Decode(conn net.Conn) ([]byte, error)` 与
`Encode(conn net.Conn, data []byte) error`。所有实现可直接交给 `tcp.Client.SetCodec`。

| 构造函数 | 分帧方式 |
|----------|----------|
| `NewRawCodec(readSize int)` | 原始透传，一次读固定大小 |
| `NewFixedLengthCodec(size int)` | 定长帧 |
| `NewDelimiterCodec(delim byte)` | 分隔符帧（如 `\n`） |
| `NewEOFCodec()` | 读到 EOF（短连接） |
| `NewLengthFieldCodec(lengthFieldSize int, byteOrder binary.ByteOrder, maxMessageSize int)` | 长度前缀（1/2/4/8 字节）+ 上限校验 |
| `NewUvarintLengthCodec(maxMessageSize int)` | uvarint 变长长度前缀 |
| `NewCustomCodec(decodeFn, encodeFn)` | 自定义编解码函数 |

## nets/mesh

轻量级 Gossip 集群，提供成员发现、Pub/Sub、KV 与 RPC。协议细节见
[design/mesh.md](../design/mesh.md)。

### Client

```go
c, err := mesh.NewClient(mesh.ClientOptions{
    ID:        "node-a",
    Addr:      ":9000",
    SeedAddrs: []string{":9001"},
    OnJoin:    func(id, addr string) { log.Println("加入:", id, addr) },
    OnLeave:   func(id string) { log.Println("离开:", id) },
    OnError:   func(err error) { log.Println("错误:", err) },
})
defer c.Close()

_ = c.Subscribe("sensor/:id", func(msg mesh.Message, params map[string]string) {})
_ = c.SubscribeWithAck("job/*", func(msg mesh.Message, params map[string]string) {})
_ = c.Publish("sensor/temp", 23.5)

_ = c.Put("config/version", 3)
v, err := c.Get("config/version")

_ = c.RegisterQueryHandler("status", func(msg mesh.Message) any { return "ok" })
resp, err := c.Query("status", map[string]any{"id": "a"})
```

| 方法 | 说明 |
|------|------|
| `NewClient(opts ClientOptions) (*Client, error)` | 创建并启动节点；`ID` 为空时按 hostname + 随机后缀生成 |
| `Subscribe(topic string, cb func(Message, map[string]string)) error` | 订阅主题，`:` 为单层通配、`*` 为多层通配 |
| `SubscribeWithAck(topic string, cb) error` | 带确认的订阅 |
| `Publish(topic string, value any) error` | 发布消息 |
| `Put(key string, value any) error` / `Get(key string) (any, error)` | 本地落库并广播 / 本地优先读取，未命中再广播查询 |
| `RegisterQueryHandler(path string, handler func(Message) any) error` | 注册 RPC 处理器（精确路径，不支持通配） |
| `Query(path string, payload any) (any, error)` | 跨节点查询，超时返回错误 |
| `GetMembers() []NodeState` / `GetMemberCount() int` | 成员表快照 / 在线节点数 |
| `GetHealth() HealthInfo` / `GetStats() ClientStats` | 健康信息 / 运行统计 |
| `IsRunning() bool` / `Close() error` | 运行状态 / 优雅关闭（广播离开） |
| `SetLogger(logger Logger)` | 注入日志实现 |

`ClientOptions` 字段：

| 字段 | 说明 |
|------|------|
| `ID` / `Addr` | 节点标识 / 监听地址（`host:port`） |
| `MaxPeers` | 连接数上限 |
| `SeedAddrs` | 种子节点地址列表 |
| `OnJoin` / `OnLeave` / `OnPub` / `OnError` | 事件回调 |
| `HealthCheckInterval` / `OnHealthCheck` | 健康检查间隔（0 表示禁用）与回调 |
| `GossipInterval` / `GossipFanoutSize` / `GossipPartialViewSize` / `GossipFailureTimeout` | Gossip 参数 |
| `RequestTimeout` / `DialTimeout` | 请求 / 拨号超时 |
| `MessageTTL` | 多跳广播跳数 |
| `Debug` | 调试输出 |

### Node（低层 API）

`NewNode(id, addr string, maxPeers int, seedAddresses []string, opts ...Option)` 直接操作
成员表与消息总线，方法为 `Subscribe`（可变数量回调）、`Publish`、`Put`、`Get`、
`RegisterQueryHandler`、`Query`、`GetNodeStates`、`Stop`。需要精细控制时使用，
一般业务优先用 `Client`。

### Config 与 Option

`mesh.NewDefaultConfig()` 默认值：

| 字段 | 默认值 |
|------|--------|
| `MessageTTL` | 0（`WithMessageTTL` 设置后启用多跳） |
| `MessageSendChanSize` | 128 |
| `MaxSeenAge` | 1 分钟 |
| `RequestTimeout` | 5 秒 |
| `ConnDialTimeout` | 5 秒 |
| `GossipInterval` | 2 秒 |
| `GossipFanoutSize` | 3 |
| `GossipPartialViewSize` | 20 |
| `GossipFailureTimeout` | 15 秒 |
| `Debug` | `false` |

可选：`WithDebug`、`WithMessageTTL`、`WithMessageSendChanSize`、`WithMaxSeenAge`、
`WithRequestTimeout`、`WithConnDialTimeout`、`WithGossipInterval`、
`WithGossipFanoutSize`、`WithGossipPartialViewSize`、`WithGossipFailureTimeout`、
`WithCodec`（自定义消息编解码）、`WithSendBlockTimeout`（消息发送背压超时）。

### 数据结构

| 类型 | 字段要点 |
|------|----------|
| `Message` | `SN`、`TTL`、`Type`、`SenderID`、`Path`、`Value`、`Payload`、`RequestID`、`GossipNodes` |
| `NodeState` | `SN`（最新消息序列号，用于增量合并）、`ID`、`Addr`、`Timestamp`（最后心跳时间） |
| `HealthInfo` | `Timestamp`、`NodeID`、`NodeAddr`、`OnlineNodes`、`NodeCount`、`Uptime`、`Sent`、`Recv` |
| `ClientStats` | `StartTime`、`MessagesSent`、`MessagesRecv`、`OnlineNodes`、`JoinEvents`、`LeaveEvents` |
| `MessageBus` 接口 | `Subscribe`、`Publish`、`Put`、`Get`、`RegisterQueryHandler`、`Query`、`HandleRemoteMessage`、`SetGossipHandler`、`Stop` |
| `MessageType` | `MESSAGE_TYPE_PUB` 等，用于路由与去重 |

消息经 JSON 编解码传输，`value any` 中的数值会变成 `float64`；需要强类型时在
`Publish` / `Put` 前自行序列化为 `[]byte` 或结构体。

## nets/serial

实现 `net.Conn` 的串口封装。

```go
port, err := serial.Open(serial.Config{
    Name:        "COM6",
    Baud:        115200,
    DataBits:    8,
    StopBits:    1,
    Parity:      'N',
    ReadTimeout: time.Second,
    FlowControl: serial.FLOW_CTRL_NONE,
})
if err != nil {
    return err
}
defer port.Close()

_, _ = port.Write([]byte{0x01, 0x03, 0x00, 0x00})
```

| 符号 | 说明 |
|------|------|
| `Open(config Config) (Port, error)` | 打开串口 |
| `NewConnection(cfg Config) (net.Conn, error)` | 以 `net.Conn` 形式返回 |
| `Port` 接口 | `io.ReadWriteCloser` + `SetConfig(Config) error` + `SetTimeouts(read, write time.Duration) error` |
| `Config` | `Name`、`Baud`、`DataBits`、`StopBits`、`Parity`（`N`/`O`/`E`）、`ReadTimeout`、`FlowControl` |
| `FlowControlType` | `FLOW_CTRL_NONE`、`FLOW_CTRL_HARDWARE_RTSCTS`、`FLOW_CTRL_HARDWARE_DTRDSR`、`FLOW_CTRL_SOFTWARE_XONXOFF` |
| `ErrTimeout` | 读超时错误，可用 `errors.Is` 判断 |
| `_GENERIC_READ` | Windows 设备读权限常量 |

串口实现区分平台：Windows 使用 `windows_port.go`，Linux 与 darwin 分别有
`termios_linux.go` / `termios_darwin.go`；`Config` 在 `Open` 前会校验波特率、数据位、
停止位、校验位与超时。串口用例依赖真实设备，默认不参与构建。

## nets/smtpx

```go
c := smtpx.NewClient("smtp.example.com", 465,
    smtpx.WithDialTimeout(10*time.Second),
    smtpx.WithTimeout(30*time.Second),
)
msg := &smtpx.Message{
    Subject: "注册成功",
    To:      []string{"user@example.com"},
    Body:    smtpx.GenerateRegisterUserHTMLModel1("App", "no-reply@example.com", "2026", "user@example.com", "alice", "******"),
    Attachments: []smtpx.Attachment{
        {Filename: "report.pdf", ContentType: "application/pdf", Data: pdfBytes},
    },
    Headers: map[string]string{"X-Campaign": "welcome"},
}
if err := c.SendHTML("no-reply@example.com", "授权码", msg); err != nil {
    return err
}
```

| 符号 | 说明 |
|------|------|
| `NewClient(server string, port int, options ...Option) *Client` | 创建客户端；465 使用隐式 TLS，587 建议配合 `WithStartTLS` |
| `WithStartTLS()` | 明文连接后通过 STARTTLS 升级为 TLS，适用于 587 等端口；默认按 465 隐式 TLS 连接 |
| `WithTLSConfig(*tls.Config)` | 自定义 TLS 配置（内部克隆，`nil` 忽略） |
| `WithDialTimeout(d)` / `WithTimeout(d)` | 拨号（含 TLS 握手）超时，默认 10s / 单次发送整体超时 |
| `SendText(username, password string, msg *Message) error` | 发送纯文本邮件；`username` 为空时跳过认证 |
| `SendHTML(username, password string, msg *Message) error` | 发送 HTML 邮件 |
| `SendRaw(username, password string, recipients []string, message []byte) error` | 发送已编码的完整报文 |
| `Message` | `Subject`、`To`、`Cc`、`Bcc`、`Body`、`Attachments`、`Headers` |
| `Attachment` | `Filename`、`ContentType`（空值按 `application/octet-stream`）、`Data` |
| `(*Message) GenerateText(form string) ([]byte, error)` / `GenerateHTML(form string) ([]byte, error)` | 用模板生成正文；有附件时自动构造 `multipart/mixed` |
| `(*Message) GetRecipients() []string` / `Validate() error` | 收件人合并且去重 / 校验 |
| `GenerateRegisterUserHTMLModel1(...)` / `GenerateResetPasswordHTMLModel1(...)` | 内置 HTML 模板 |
| `HTML_HEAD_TEMPLATE` | 模板头部常量 |
| `EncodeQuotedPrintable(data []byte) []byte` | quoted-printable 编码 |

主题按 RFC 2047 编码，正文使用 quoted-printable，附件按 Base64 76 列换行；
报文会自动补充 `Date` 与 `Message-ID`。SMTP 用例依赖真实账号，默认不参与构建。

收件人、自定义头、附件 `ContentType` / `Filename` 以及信封地址（`MAIL FROM` / `RCPT TO`，
含 `SendRaw`）中的 CR/LF 会被拒绝并返回 `ErrHeaderInjection`，避免邮件头或 SMTP 命令注入。

`NewClient` 用 `net.JoinHostPort` 拼接地址，IPv6 字面量可直接传入：
`NewClient("::1", 587)` 的 `GetAddr()` 返回 `[::1]:587`。

## nets/sshx

```go
c, err := sshx.NewClient("10.0.0.10", "root", "password",
    sshx.WithPort(22), sshx.WithTimeout(10*time.Second))
if err != nil {
    return err
}
defer c.Close()

out, err := c.Execute(&sshx.Command{Command: "uname -a", Timeout: 5 * time.Second})
out, err = c.RunCommand("df -h")

c.ExecuteStream(&sshx.StreamCommand{Command: "tail -f /var/log/syslog"}, handler)
```

| 符号 | 说明 |
|------|------|
| `NewClient(host, user, pwd string, opts ...Option) (*Client, error)` | 建立 SSH 连接 |
| `Execute(*Command) ([]byte, error)` / `ExecuteContext(ctx, *Command)` | 非流式命令；`ExecuteContext` 同时受 `ctx` 与 `Command.Timeout` 约束，取较早的截止时间 |
| `ExecuteStream(*StreamCommand, StreamHandler)` / `ExecuteStreamContext(ctx, ...)` | 流式命令 |
| `RunCommand(cmd string) ([]byte, error)` | 便捷执行 |
| `SetReconnect(retries int, interval ...time.Duration)` | 断线重连 |
| `IsClosed() bool` / `Close() error` / `GetClient() *ssh.Client` | 状态 / 关闭 / 底层连接 |
| `Command` | `Command`、`Timeout`、`Terminal`、`InputArgs`、`InputInterval`、`QuitSignals`、`QuitTimeout`、`StderrFilter`、`ExitCodeFilter` |
| `StreamCommand` | 同上，另有 `StdoutReadDelimit`（默认 `\n`）、`StdoutReadTimeoutAfterClose`（默认 1s） |
| `StreamHandler` 接口 | `CallOnStdin`、`CallOnStdout`、`CallOnStderr`、`CallOnDone`、`CallOnError` |
| `Terminal` | `Term`（默认 `xterm`）、`Width` 120、`Height` 40、`TtyOpIspeed` / `TtyOpOspeed` 14400 |
| `Option` | `WithPort`、`WithTimeout`、`WithDebug(maxSize ...int64)`、`WithReconnect`、`WithAuth`、`WithHostKeyCallback`、`WithKnownHosts`、`WithInsecureIgnoreHostKey` |
| 工具函数 | `TrimASNIString`、`TrimASNIBytes`、`JoinNumberSliceToString`、`FormatStringSliceToIntSlice` |

注意：`StreamHandler` 可能被并发调用，实现需自行保证线程安全。
默认使用 `ssh.InsecureIgnoreHostKey()` 便于首次连接；生产环境应通过 `WithKnownHosts` 或
`WithHostKeyCallback` 启用主机密钥校验，避免中间人攻击。
`WithDebug` 会记录执行的命令内容；`Command.InputArgs` 只记录长度而不输出内容，避免口令进入日志。

### nets/sshx/command

通过一次 SSH 会话采集 Linux 设备的通用性能指标，并内置 `/proc`、`df`、`ps`、`top` 输出解析器。
远端脚本只使用 POSIX `sh` 与 `/proc`，已在 GNU coreutils（Ubuntu）、BusyBox 1.24（嵌入式 ARM）
与 Buildroot / Jetson（aarch64）上验证。

```go
client, err := sshx.NewClient("10.0.0.10", "root", "password")
if err != nil {
    return err
}
defer client.Close()

snapshot, err := command.NewCollector(client,
    command.WithSampleInterval(2*time.Second),
    command.WithCollectTimeout(20*time.Second),
).Collect(context.Background())
if err != nil {
    return err
}

fmt.Println(snapshot.Format())    // 人类可读概览
payload, _ := json.Marshal(snapshot) // 结构体带 snake_case json tag
```

| 符号 | 说明 |
|------|------|
| `NewCollector(client *sshx.Client, opts ...Option) *Collector` | 创建采集器 |
| `WithSampleInterval(d)` | 两次 CPU 采样间隔，默认 1s；脚本按整秒 `sleep`，非整秒向上取整、不足 1s 抬升为 1s（BusyBox `sleep` 不支持小数），`Snapshot.SampleInterval` 报告实际整秒值 |
| `WithCollectTimeout(d)` | 单次采集超时，默认 20s；调用方 `ctx` 的 deadline 更短时以调用方为准；应大于采样间隔，否则会在 `sleep` 期间到期 |
| `(*Collector) Collect(ctx) (*Snapshot, error)` | 采集一次快照；各段独立解析，仅当 CPU 与内存都缺失时返回错误 |
| `(*Snapshot) Format() string` / `WriteTo(io.Writer)` | 人类可读文本 / 写入 writer |
| `ParseUptime` / `ParseLoadAverage` / `ParseMemInfo` / `ParseProcStat` / `ParseCPUInfo` / `ParseNetDev` | `/proc` 与 `uptime` 输出解析 |
| `ParseDf` / `ParsePS` / `ParseTop` / `NewTopParser` | `df -P -k`、`ps`、`top` 输出解析 |

`Snapshot` 字段：

| 字段 | 说明 |
|------|------|
| `Host` / `Kernel` / `CollectedAt` / `SampleInterval` | 主机名、`uname -srm`、采集时间与采样间隔 |
| `Uptime` / `CPUCores` / `Load` | 运行时长、核心数、1/5/15 分钟负载（含 `Running` / `Total` 任务数） |
| `CPU` / `CoreUsage` | 整机与逐核使用率（`cpu0`、`cpu3`… 核心编号允许稀疏） |
| `Memory` / `Swap` | 内存与交换区；内核早于 3.14 无 `MemAvailable` 时按 Free+Buffers+Cached 推算并置 `AvailableEstimated` |
| `Filesystems` / `Networks` | 挂载点容量（BusyBox 折行记录与含空格的挂载点均可正确合并）/ 网卡收发与丢包 |

`LoadAverage`、`CPUTimes`、`CPUUsage`、`MemoryStats` 等类型实现了 `String()`；
`Format()` 对超过 16 核的设备只输出 min/avg/max 概览，完整逐核数据仍在 `CoreUsage` 中。
解析失败统一返回 `ErrParse`，参数非法返回 `ErrInvalidArgument`，无任何分段返回 `ErrNoData`，
均可用 `errors.Is` 判断。真实设备用例由环境变量驱动（见 testing.md）。
采集脚本固定 `LC_ALL=C`；解析器本身按英文关键字匹配，直接处理带本地化表头的历史输出时需自行翻译或预先统一 locale。

## 平台差异与限制

- `serial` 在 Windows 使用 `COMx`，Linux 使用 `/dev/ttyS*` / `/dev/ttyUSB*`。
- 串口、SSH、SMTP、指定网卡的用例全部通过 `integration` 标签隔离，不参与默认测试。
- `nets/sshx/command` 的 `integration` 用例由 `GOKIT_SSH_HOST` / `GOKIT_SSH_USER` /
  `GOKIT_SSH_PASSWORD` 驱动，未设置时自动跳过；解析器本身由真实设备样本覆盖，默认测试即可验证。
  密码变量 `GOKIT_SSH_PASSWORD` 与 `GOKIT_SSH_PASS` 等价（`nets/sshx` 与 `nets/sshx/command` 均兼容）。
- `tcp.Client` 的重连只覆盖下一次操作，不保证在途请求自动重放。
