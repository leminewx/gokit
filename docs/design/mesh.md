# Mesh 集群设计

`nets/mesh` 是一个去中心化、无外部依赖的轻量集群库：节点通过 TCP 互连，用 Gossip 维护
成员表，用广播实现 Pub/Sub 与 KV，用请求-响应实现 RPC。本文说明协议行为、关键算法与限制。

## 组件分层

```
Client            // 对外门面：配置映射、事件回调、健康检查、统计
├── Gossip        // 成员表维护、心跳、部分视图传播、故障检测
├── MessageBus    // 消息去重、路由、Pub/Sub、KV、RPC、多跳转发
└── Transport     // TCP 连接管理、JSON 编解码、广播/定向发送、连接评分
```

- `NewClient(ClientOptions)` 依次构建 `Config` → `Transport` → `MessageBus` → `Gossip`，
  并注册 Gossip 消息处理器；`Close()` 逆序停止并广播离开。
- `NewNode(...)` 是低层入口，直接暴露 `MessageBus` 的 `Subscribe/Publish/Put/Get/Query`，
  不提供 `Client` 的事件回调与统计。

## 消息模型

`Message` 经 `encoding/json` 在连接上传输：

| 字段 | 作用 |
|------|------|
| `SN` | 发送方单调递增序列号，用于去重 |
| `TTL` | 剩余跳数，`> 0` 时转发并减 1 |
| `Type` | 消息类型（见下表） |
| `SenderID` | 原始发送者，多跳转发时保持不变 |
| `Path` | 主题 / 键 / RPC 路径 |
| `Value` / `Payload` | 业务值 / RPC 参数与结果 |
| `RequestID` | 请求-响应匹配（仅发送方本地唯一） |
| `GossipNodes` | 心跳携带的部分成员视图 |

| `MessageType` | 方向 | 说明 |
|---------------|------|------|
| `MESSAGE_TYPE_PUB` | 广播 | 发布 |
| `MESSAGE_TYPE_PUT` | 广播 | 写 KV |
| `MESSAGE_TYPE_GET` | 广播 | 读 KV 请求 |
| `MESSAGE_TYPE_GET_RESPONSE` | 定向 | 读响应 |
| `MESSAGE_TYPE_QUERY` | 广播 | RPC 请求 |
| `MESSAGE_TYPE_QUERY_RESPONSE` | 定向 | RPC 响应 |
| `MESSAGE_TYPE_GOSSIP_HEARTBEAT` / `MESSAGE_TYPE_GOSSIP_SYNC` | 扇出 | 心跳 / 全量同步 |

## 消息去重

每个节点维护 `seen` 表，键为三元组：

```go
type seenKey struct {
    senderID string
    sn       int64
    msgType  MessageType
}
```

设计要点：

1. **必须带消息类型**。Gossip 有自己的序列号计数器，业务消息总线也有一个；两者都单调递增，
   数值区间会重叠，只用 `(SenderID, SN)` 会把同号的心跳与业务消息误判为重复。
2. **序列号必须用单调计数器**，不能用 `time.Now().UnixNano()`。Windows 等平台的系统时钟
   粒度较粗，连续发布的消息可能拿到相同纳秒值，被误判为重复而丢弃。
   `SN` 初始化为 `time.Now().UnixNano()`，之后每次 `Add(1)`。
3. **多跳副本保留原始 `SenderID`、`SN`、`Type`**，因此同一消息经不同路径到达时仍能去重。
4. `seen` 由后台协程按 `MaxSeenAge`（默认 1 分钟）清理，避免无限增长；
   清理间隔最小 1 秒。

## Gossip 成员管理

### 心跳与部分视图

每个 `GossipInterval`（默认 2 秒），节点随机选择 `GossipFanoutSize`（默认 3）个邻居发送心跳，
心跳携带最多 `GossipPartialViewSize`（默认 20）个本地已知节点的状态。接收方合并视图，
从而让成员信息在 `O(log N)` 轮内收敛。

### incarnation 与时间戳刷新

`NodeState.SN` 是该节点最新消息序列号，充当 incarnation：

- 已知节点：仅当收到的 `SN` **大于**本地记录时才刷新 `Timestamp` 与 `Addr`。
  这样旧视图无法把已停止节点的时间戳不断推后。
- 第三方节点：**沿用消息中携带的观测时间**建档，而不是本地接收时间；
  若该观测时间距现在已超过 `GossipFailureTimeout`（默认 15 秒），直接丢弃不建档。

这一约束是故障检测能收敛的前提：如果第三方节点用本地时间戳，已停止节点会被其他节点的
过期视图反复「复活」，永远无法被剔除（livelock）。

### 故障检测

超过 `GossipFailureTimeout` 未刷新时间戳的节点会被移除，并触发 `OnLeave`；
新出现的节点触发 `OnJoin`。`GetMemberCount()` / `GetMembers()` 读取本地成员表快照。

### 连接管理

- `MaxPeers` 限制同时连接数，软限制允许超出 20%。
- 连接带评分：新连接有 30 秒保护期；种子节点有初始激励分；活跃分每分钟最多累加 10 次；
  分数按小时衰减。超出上限时优先淘汰低分连接。
- 种子地址只用于首次接入，之后成员发现完全依赖 Gossip。

## KV 语义

```go
_ = c.Put("config/version", 3)   // 本地落库 + 广播 PUT
v, err := c.Get("config/version") // 本地优先；未命中再广播 GET
```

- **写**：`Put` 先写本地副本再广播，保证「读己之写」；对端节点数小于 2 时也能读到。
- **读**：`Get` 先查本地；未命中时向直连邻居广播 `GET`，持有该键的节点定向回 `GET_RESP`。
- **一致性**：最终一致，无版本向量与冲突解决；并发写同一键时后到者覆盖。
- **类型**：值经 JSON 往返，数值会变成 `float64`；需要保类型时存 `[]byte` 或结构体。
- **无持久化**：进程退出即丢失，需要持久化时由调用方落盘。

## Pub/Sub

- `Subscribe(topic, cb)` 把回调插入 Trie，支持两种通配符：
  - `:` 单层通配，例如 `sensor/:id`，匹配值通过 `pathValues["id"]` 回传；
  - `*` 多层通配，例如 `sensor/*`。
- 同一主题重复订阅会追加回调，不覆盖。
- `Publish(topic, value)` 广播 `PUB`，命中订阅的回调在**独立 goroutine** 中执行，
  因此回调需自行保证并发安全，且不要执行长阻塞操作。
- Trie 非并发安全，`Subscribe` 与投递时的 `Search` 用读写锁隔离。

## RPC

- `RegisterQueryHandler(path, handler)` 以**精确路径**注册（不做通配匹配）。
- `Query(path, payload)` 广播 `QUERY`，命中处理器的节点把返回值写入 `QUERY_RESP.Value`。
- 请求通过 `RequestID` 与本地 `pending` 表匹配，`RequestTimeout`（默认 5 秒）到期返回
  `request timeout`。
- 多个节点注册同一路径时，先到的响应生效，其余被丢弃。

## 多跳与请求范围（重要限制）

- `PUB` / `PUT` 的转发由 `MessageTTL` 控制：默认 `0`，即**只发给直连邻居**；
  需要多跳传播时设置 `MessageTTL > 0`（`ClientOptions.MessageTTL` 或 `WithMessageTTL`）。
- `GET` / `QUERY` 请求**不转发**（`sendRequest` 不设置 TTL），响应也定向回到
  「上一跳」地址。因此请求-响应语义只在**直连邻居**范围内可靠。
- 集群规模较大时：
  - Pub/Sub 提高 `MessageTTL`，并适当增大 `GossipFanoutSize`；
  - KV 读与 RPC 应保证每个节点都有本地副本（写扩散 + `Put`），或把 `MaxPeers`
    设置到足以形成全连接，或在上层实现路由。

## 配置与调优

| 参数 | 默认 | 调优建议 |
|------|------|----------|
| `MessageTTL` | 0 | 需要多跳广播时设为 2~4；过大会放大流量 |
| `MessageSendChanSize` | 128 | 高吞吐场景增大，避免发送阻塞 |
| `MaxSeenAge` | 1 分钟 | 小于 `GossipInterval` 会导致重复投递，保持默认或更大 |
| `RequestTimeout` | 5 秒 | 跨机房或高负载时增大 |
| `ConnDialTimeout` | 5 秒 | 弱网环境增大 |
| `GossipInterval` | 2 秒 | 大规模集群可增大以降低开销；故障检测精度随之下降 |
| `GossipFanoutSize` | 3 | 收敛速度与流量的折中，一般 2~4 |
| `GossipPartialViewSize` | 20 | 与集群规模相关，20~30 为工程惯例 |
| `GossipFailureTimeout` | 15 秒 | 应显著大于 `GossipInterval`，否则易误判 |

测试中通常缩短为：`GossipInterval = 200ms`、`GossipFailureTimeout = 2s`，
把故障检测从数十秒压到数秒（见 `mesh_integration_test.go`）。

## 已知限制

1. 无认证与加密，TCP 上明文传输 JSON；跨公网部署需自行加隧道或 TLS 代理。
2. `Get` / `Query` 仅单跳可用（见上节），大集群需调用方配合。
3. 无持久化、无冲突解决、无事务。
4. 依赖节点间时钟大致同步：`NodeState.Timestamp` 来自发送方，时钟漂移会影响故障检测。
5. 回调在独立 goroutine 执行，顺序不保证；需要顺序消费时在回调内自行排队。
6. 成员表规模、`seen` 表大小与消息体大小都没有硬上限，超大规模场景需评估内存。

## 排障清单

| 现象 | 排查方向 |
|------|----------|
| 订阅收不到消息 | 主题通配符是否用 `:`/`*`（不是 `#`）；`MessageTTL` 是否为 0 导致跨跳不通 |
| `Get` 超时 | 键是否只在非直连节点；写是否已通过 `Put` 本地落库；`RequestTimeout` 是否过短 |
| 节点长期不消失 | `GossipFailureTimeout` 是否过大；心跳是否被时钟漂移或网络分区阻断 |
| 偶发重复投递 | `MaxSeenAge` 是否小于 `GossipInterval`；SN 是否被错误地按时间生成 |
| 高并发下丢消息 | `MessageSendChanSize` 是否过小；回调是否阻塞导致分发堆积 |

集成测试与事件驱动写法见 [../testing.md](../testing.md)，完整 E2E 用例在
`nets/mesh/mesh_integration_test.go`。
