# 已知问题与限制

本文件记录当前版本中**已知且有意保留**的限制、平台差异与使用风险。
缺陷修复进展见 [RELEASE.md](./RELEASE.md)。

## Mesh（nets/mesh）

| 问题 | 影响 | 规避方式 |
|------|------|----------|
| `Get` / `Query` 请求不转发 | 请求-响应只在直连邻居范围内可靠，跨跳读取会超时 | 写扩散保证本地副本；增大 `MaxPeers` 形成全连接；或在上层实现路由 |
| `PUB` / `PUT` 默认 `MessageTTL = 0` | 消息只发给直连邻居 | 需要多跳时设置 `MessageTTL > 0` |
| 无认证与加密 | TCP 上明文传输 JSON，端口暴露即可被读写 | 仅部署在可信内网，或用隧道 / TLS 代理 |
| 无持久化 | 进程退出后 KV 与成员信息全部丢失 | 由调用方落盘或重建 |
| 无冲突解决 | 并发写同一键，后到者覆盖 | 调用方加版本号 / 时间戳字段 |
| 值经 JSON 往返 | 数值类型变为 `float64`，结构体字段可能丢失（未导出/无标签） | 存 `[]byte` 或显式结构体并检查类型断言 |
| 订阅回调在独立 goroutine 执行 | 顺序不保证，回调之间可能并发 | 回调内自行排队或加锁，避免长阻塞 |
| `MaxSeenAge` 小于心跳间隔 | 去重窗口过短会导致重复投递 | 保持 `MaxSeenAge >= GossipInterval`（默认 1 分钟 / 2 秒） |
| 依赖节点时钟大致同步 | 时钟漂移会延长或误判故障检测 | 启用 NTP；增大 `GossipFailureTimeout` |
| 成员表、`seen`、消息体无硬上限 | 超大规模场景内存不可控 | 控制集群规模与消息大小 |

## 平台与运行环境

| 问题 | 说明 |
|------|------|
| Windows 系统时钟粒度粗 | 连续 `time.Now().UnixNano()` 可能返回相同值。库内消息序列号已改用单调计数器；依赖时间或定时器的新代码应通过 `timex.Clock` 注入时间源，测试使用 `timex.Manual` |
| `progressbar` 使用 ANSI 序列 | 仅适用于 ANSI 终端；写入文件/日志时请 `SetWriter` 并关闭光标控制 |
| 串口平台差异 | Windows 使用 `COMx`，Linux 使用 `/dev/ttyS*`、`/dev/ttyUSB*`，macOS 使用 `/dev/cu.*`；权限与驱动由系统决定 |
| `serial.Config` 校验 | `Parity` 仅接受 `N` / `O` / `E`，数据位 5–8、停止位 1–2、波特率为正；非法配置返回 `ErrInvalidConfig` |
| `nets.GetNetPreferredIP` | 依赖具体网卡名称与前缀，跨机器不可移植 |

## 非并发安全类型

以下类型未提供内部同步，多个 goroutine 共享时需调用方加锁：

- `structures/bitset.BitSet`
- `structures/fsm.FSM`（同一实例的 `Trigger` 需串行）
- `structures/tree.Trie`（Mesh 内部已用读写锁保护）
- `stream.Stream`（每条流应由单一消费者处理，`FanOut` 子流不能阻塞）
- `structures/dict.UnsafeDict`、`structures/set.UnsafeSet`

## 生命周期与并发语义

| 行为 | 说明 |
|------|------|
| `queue.DelayQueue` / `timewheel.SingleWheel` 停止后不可重启 | `Stop` 会永久置位 `stopped`，后续 `Start` 为无操作，`AddTask` 返回 0；需要重启请新建实例 |
| 时间轮回调异步执行 | 回调在独立 goroutine 运行；当 `period` 短于执行耗时时，同一任务的多次执行可能重叠，需要互斥时由回调自行加锁 |
| `pool.Release` 会先取消上下文 | 依赖 `Task(ctx)` 的任务会被提前取消以便尽快退出；清理逻辑应幂等，已入队但未执行的任务仍会执行 |
| `ratelimiter.WaitN` 不保证公平 | 等待者以定时器轮询令牌，多个并发等待者之间没有 FIFO 顺序，极端情况下可能饥饿 |
| `stream.FanOut` 元素副本 | 首个子流使用原始元素，其余子流使用快照副本；子流内部的原地修改不会影响兄弟子流，但也不回写到源流 |
| `sshx` 重连会持有内部锁 | 重连期间其他会话操作会被阻塞；`Close` 可打断重连间隔等待，但仍需等待一次拨号超时（默认 30s） |
| `sshx.ExecuteContext` 同时受 `ctx` 与 `Command.Timeout` 约束 | 两者同时设置时取较早的截止时间；只设 `ctx` 时由调用方负责取消与超时 |
| `mesh` 初始同步固定等待 | `Start` 后的首次 `SYNC` 固定等待 200ms 让 TCP 连接建立，极慢网络下可能错过首轮发现，下一轮 Gossip 会自动补齐 |

## 包名与目录

全部可导入包直接位于模块根目录，目录名与包名一致；标准库增强包以 `x` 结尾
（`stringx` / `timex` / `filex` / `smtpx` / `sshx` / `rsax`），同一领域的多实现目录以 `s`
结尾（`maths` / `nets` / `cryptos` / `configs` / `structures`），详见
[docs/architecture.md](./docs/architecture.md)。

## 测试环境依赖

以下用例默认不参与构建，需要 `-tags integration` 且准备真实环境后才能运行（设备信息
全部通过环境变量注入，变量缺失时自动跳过，详见 [docs/testing.md](./docs/testing.md)）：

- `nets/net_env_test.go`：`GOKIT_NET_INTERFACE` 指定的本机网卡 + `GOKIT_NET_PROBE_ADDR`
- `nets/tcp/client_integration_test.go`：`GOKIT_TCP_HOST` / `GOKIT_TCP_PORT` 指定的 TCP 设备
- `nets/serial/port_integration_test.go`：`GOKIT_SERIAL_PORT` 指定的本机串口
- `nets/smtpx/client_integration_test.go`：`SMTPX_HOST` / `SMTPX_USER` / `SMTPX_PASS` 指定的 SMTP 账号
- `nets/sshx/*_test.go`：`GOKIT_SSH_HOST` / `GOKIT_SSH_USER` / `GOKIT_SSH_PASS` 指定的 SSH 主机
- `nets/sshx/command/collect_live_test.go`：同上（兼容 `GOKIT_SSH_PASSWORD`）

## 安全提示

- 仓库中不包含任何真实私钥；`cryptos/rsax` 的测试密钥写入 `harness/testdata`
  并在用例结束时删除。
- 设备地址、SMTP 账号、SSH 口令等仅存在于带 `integration` 标签的测试文件中，
  不应复制到业务代码或提交到公开仓库。
- Mesh 与串口都是明文协议，不要直接暴露到公网。
- `sshx` 默认使用 `ssh.InsecureIgnoreHostKey()`，生产环境必须通过 `WithKnownHosts`
  或 `WithHostKeyCallback` 启用主机密钥校验。
- `smtpx` 465 端口走隐式 TLS，587 需配合 `WithStartTLS`；证书校验失败会在拨号阶段报错。
- `smtpx.SendRaw` 只校验信封地址中的 CR/LF，不校验原始报文内容；邮件头与正文需由调用方
  自行按 MIME 规范编码，不要直接拼入未经校验的用户输入。
- `cryptos/rsax` 仅生成不小于 2048 位的密钥（1024 位仍可解析），私钥落盘权限为 `0600`。
- `sshx` 的调试日志会记录命令文本（`InputArgs` 仅记录长度）；命令本身可能包含口令等敏感信息，
  生产环境请谨慎开启 `WithDebug`。

## 文档与 API 语义易错点

- `gokit.GetValueWithDefault(flt, values...)` 返回**第一个候选值** `values[0]`
  （不是「第一个非零值」）；无候选值时才返回 `flt`。
- Mesh 订阅通配符是 `:`（单层）与 `*`（多层），不是 `#`；`RegisterQueryHandler`
  使用精确路径，不支持通配。
- `progressbar.Update` 只接受递增，回退会被忽略；重复 `Finish` 不会产生额外输出。
- `cache.TTLCache` 的过期清理是懒执行的，`GetLength` 可能包含尚未清理的过期项；
  `Put` / `Delete` / `Clear` / `Purge` 的返回值是「是否淘汰」或「处理数量」，已存在的
  调用语句无需修改，但需要结果时应显式接收。
- `smtpx.Message.GetRecipients` 会自动去重并保持 To / Cc / Bcc 的原始顺序，不再依赖 `Validate`。
- `tcp.WithReconnect(interval, retries...)`：省略 `retries` 为无限重连，`0` 关闭自动重连，`>0` 为最大次数；
  新建客户端默认不重连。
- `ratelimiter.WaitN` 在请求永远无法满足时返回 `ErrUnattainable`，不要将其当作暂时性错误重试。
- `bloom.Config.FalsePositiveRate` 必须位于 (0, 1)，否则会被静默回退为 0.01。
- `progressbar.GetProgress` 在 `total <= 0` 时返回 0；渲染同样按 0% 处理并直接结束。
- `sshx/command` 的 `Collect` 各段尽力解析，单段失败不会整体报错，仅当 CPU 与内存都缺失时返回错误；
  `Memory.AvailableEstimated == true` 表示可用内存由 Free + Buffers + Cached 推算（内核早于 3.14）。
  `WithCollectTimeout` 需要大于采样间隔，否则会在 `sleep` 期间被 `ctx` 取消（此时通常只剩前几段）。
- `command.ParseDf` 只支持 `df -k` / `df -P -k` 的固定列，`df -h` 的人类可读单位无法还原字节数；
  折行记录与含空格的挂载点会被合并，但表头需为英文（`Collect` 的脚本已固定 `LC_ALL=C`，
  自行调用 `df` 时需同样处理，否则本地化表头会被跳过或拼进首条记录名）。
- `command.ParsePS` 处理 `ps aux` 时，如果命令经 PTY 执行，`ps` 会按 `Terminal.Width`（默认 120）
  截断 `COMMAND` 列；需要完整命令行时使用 `ps -ww` 或增大 `Terminal.Width`。
