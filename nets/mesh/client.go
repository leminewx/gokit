package mesh

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// ******************************************************************************************************************
// Client：Mesh 网络的完整客户端封装
//
// 在 Node（底层传输/消息总线/Gossip）之上提供：
//   - 生命周期管理（启动、运行、优雅关闭）
//   - 事件回调（节点加入/离开、消息到达）
//   - 连接重试与自动恢复
//   - 运行时统计信息
//   - 健康检查
//   - 结构化日志
// ******************************************************************************************************************

type (
	// Client Mesh 分布式节点客户端
	Client struct {
		// 底层 Node 实例
		node *Node
		// 客户端配置
		opts ClientOptions

		// 运行时统计
		stats ClientStats

		// 生命周期控制
		ctx    context.Context
		cancel context.CancelFunc
		wg     sync.WaitGroup

		// 状态标记
		running int32
		closed  int32

		// 日志（atomic.Pointer 存储：SetLogger 可能在后台 goroutine 打日志时被调用）
		logger atomic.Pointer[loggerHolder]
	}

	// ClientOptions 客户端配置
	ClientOptions struct {
		// 节点唯一标识，默认自动生成（基于 hostname + 随机后缀）
		ID string
		// 节点监听地址，格式 "host:port"，例如 "0.0.0.0:9000"
		Addr string
		// 最大连接数（peer 数量上限）
		MaxPeers int
		// 种子节点地址列表（首次加入集群时使用）
		SeedAddrs []string

		// 事件回调
		OnJoin  func(nodeID, addr string)       // 节点加入集群
		OnLeave func(nodeID string)             // 节点离开集群
		OnPub   func(topic string, msg Message) // 收到 Pub 消息（需配合 Subscribe）
		OnError func(err error)                 // 错误回调

		// 健康检查
		HealthCheckInterval time.Duration         // 健康检查间隔（0 表示禁用）
		OnHealthCheck       func(info HealthInfo) // 健康检查回调

		// Gossip 配置
		GossipInterval        time.Duration // Gossip 广播间隔
		GossipFanoutSize      int           // 扇出数量
		GossipPartialViewSize int           // 部分视图大小
		GossipFailureTimeout  time.Duration // 故障超时

		// 请求超时
		RequestTimeout time.Duration
		// 拨号超时
		DialTimeout time.Duration
		// 消息 TTL（多跳广播跳数）
		MessageTTL int

		// 消息发送队列缓冲大小（缺省 128）
		MessageSendChanSize int
		// 发送队列满时单播的等待时长（缺省 0，表示立即失败）
		SendBlockTimeout time.Duration
		// 消息编解码器（缺省 NewJSONCodec()）
		Codec Codec

		// 调试模式
		Debug bool
	}

	// ClientStats 客户端运行时统计
	ClientStats struct {
		// 启动时间
		StartTime time.Time
		// 已发送消息总数
		MessagesSent int64
		// 已接收消息总数
		MessagesRecv int64
		// 当前在线节点数
		OnlineNodes int64
		// 加入事件数
		JoinEvents int64
		// 离开事件数
		LeaveEvents int64
		// 因发送队列满而丢弃的消息数（背压丢包）
		MessagesDropped int64
		// 编码失败次数
		EncodeFailed int64
		// 解码失败次数
		DecodeFailed int64
	}

	// HealthInfo 健康状态信息
	HealthInfo struct {
		// 检查时间
		Timestamp time.Time
		// 本节点 ID
		NodeID string
		// 本节点地址
		NodeAddr string
		// 在线节点列表
		OnlineNodes []string
		// 在线节点数
		NodeCount int
		// 运行时长
		Uptime time.Duration
		// 消息统计
		Sent int64
		Recv int64
	}

	// Logger 日志接口（用户可注入自定义实现）
	Logger interface {
		Debug(format string, args ...any)
		Info(format string, args ...any)
		Warn(format string, args ...any)
		Error(format string, args ...any)
	}

	// loggerHolder 让 atomic.Pointer 以单一具体类型承载可替换的 Logger 实现
	//   - atomic.Value 要求所有 Store 的具体类型一致，直接存接口会在切换实现时 panic
	loggerHolder struct {
		logger Logger
	}

	// defaultLogger 默认日志实现（标准库 log）
	defaultLogger struct {
		nodeID string
		logger *log.Logger
	}
)

// NewClient 创建 Mesh 客户端
//
// 使用示例：
//
//	client, err := mesh.NewClient(mesh.ClientOptions{
//	    Addr:      "0.0.0.0:9001",
//	    MaxPeers:  10,
//	    SeedAddrs: []string{"192.168.1.1:9001"},
//	    OnJoin:    func(id, addr string) { fmt.Printf("node %s joined\n", id) },
//	    OnLeave:   func(id string) { fmt.Printf("node %s left\n", id) },
//	    Debug:     true,
//	})
//
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// 订阅主题
//	client.Subscribe("chat/room1", func(msg mesh.Message, params map[string]string) {
//	    fmt.Printf("收到: %v\n", msg.Value)
//	})
//
//	// 发布消息
//	client.Publish("chat/room1", "hello world")
//
//	// 存储/查询
//	client.Put("config/key", "value")
//	val, _ := client.Get("config/key")
//
//	// RPC 查询
//	client.RegisterQueryHandler("/rpc/status", func(msg mesh.Message) any {
//	    return map[string]any{"status": "ok"}
//	})
//	result, _ := client.Query("/rpc/status", nil)
//
//	// 获取集群信息
//	info := client.GetHealth()
//	fmt.Printf("在线节点: %d\n", info.NodeCount)
func NewClient(opts ClientOptions) (*Client, error) {
	// 参数校验
	if opts.Addr == "" {
		return nil, ErrAddrRequired
	}
	if opts.MaxPeers <= 0 {
		opts.MaxPeers = 10
	}
	if opts.ID == "" {
		opts.ID = generateNodeID(opts.Addr)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建日志
	c := &Client{
		opts:   opts,
		ctx:    ctx,
		cancel: cancel,
	}
	c.logger.Store(&loggerHolder{logger: newDefaultLogger(opts.ID)})

	// 构建底层 Node
	if err := c.buildNode(); err != nil {
		cancel()
		return nil, wrapError(ErrNodeBuild, "%w", err)
	}

	// 标记运行中
	atomic.StoreInt32(&c.running, 1)
	c.stats.StartTime = time.Now()

	// 启动后台任务
	c.startBackgroundTasks()

	c.log().Info("Mesh 客户端启动成功: id=%s, addr=%s, maxPeers=%d, seeds=%v",
		opts.ID, opts.Addr, opts.MaxPeers, opts.SeedAddrs)

	return c, nil
}

// buildNode 构建底层 Node 及配置
func (c *Client) buildNode() error {
	// 构建 Option 列表
	var optsList []Option

	if c.opts.Debug {
		optsList = append(optsList, WithDebug())
	}
	if c.opts.MessageTTL > 0 {
		optsList = append(optsList, WithMessageTTL(c.opts.MessageTTL))
	}
	if c.opts.RequestTimeout > 0 {
		optsList = append(optsList, WithRequestTimeout(c.opts.RequestTimeout))
	}
	if c.opts.DialTimeout > 0 {
		optsList = append(optsList, WithConnDialTimeout(c.opts.DialTimeout))
	}
	if c.opts.GossipInterval > 0 {
		optsList = append(optsList, WithGossipInterval(c.opts.GossipInterval))
	}
	if c.opts.GossipFanoutSize > 0 {
		optsList = append(optsList, WithGossipFanoutSize(c.opts.GossipFanoutSize))
	}
	if c.opts.GossipPartialViewSize > 0 {
		optsList = append(optsList, WithGossipPartialViewSize(c.opts.GossipPartialViewSize))
	}
	if c.opts.GossipFailureTimeout > 0 {
		optsList = append(optsList, WithGossipFailureTimeout(c.opts.GossipFailureTimeout))
	}
	if c.opts.MessageSendChanSize > 0 {
		optsList = append(optsList, WithMessageSendChanSize(c.opts.MessageSendChanSize))
	}
	if c.opts.SendBlockTimeout > 0 {
		optsList = append(optsList, WithSendBlockTimeout(c.opts.SendBlockTimeout))
	}
	if c.opts.Codec != nil {
		optsList = append(optsList, WithCodec(c.opts.Codec))
	}

	// 创建 Node
	node, err := NewNode(c.opts.ID, c.opts.Addr, c.opts.MaxPeers, c.opts.SeedAddrs, optsList...)
	if err != nil {
		return err
	}

	// 注入事件处理器（通过替换 defaultGossipHandler）
	c.injectGossipHandler(node)

	c.node = node
	return nil
}

// injectGossipHandler 将用户事件回调注入到 Node 的 Gossip 层
func (c *Client) injectGossipHandler(node *Node) {
	handler := &clientGossipHandler{
		client:  c,
		onJoin:  c.opts.OnJoin,
		onLeave: c.opts.OnLeave,
	}
	node.gossip.RegisterHandler(handler)
}

// clientGossipHandler 实现 GossipHandler 接口，桥接用户回调
type clientGossipHandler struct {
	client  *Client
	onJoin  func(nodeID, addr string)
	onLeave func(nodeID string)
}

func (h *clientGossipHandler) CallOnJoin(nodeID, addr string) {
	atomic.AddInt64(&h.client.stats.JoinEvents, 1)
	h.client.log().Info("节点加入: id=%s, addr=%s", nodeID, addr)
	if h.onJoin != nil {
		h.onJoin(nodeID, addr)
	}
}

func (h *clientGossipHandler) CallOnLeave(nodeID string) {
	atomic.AddInt64(&h.client.stats.LeaveEvents, 1)
	h.client.log().Warn("节点离开: id=%s", nodeID)
	if h.onLeave != nil {
		h.onLeave(nodeID)
	}
}

// startBackgroundTasks 启动后台任务
func (c *Client) startBackgroundTasks() {
	// 健康检查任务
	if c.opts.HealthCheckInterval > 0 {
		c.wg.Add(1)
		go c.loopHealthCheck()
	}

	// 统计更新任务
	c.wg.Add(1)
	go c.loopStatsUpdate()
}

// ─────────────────────────────────────────────
// 核心 API（Pub/Sub、KV、RPC）
// ─────────────────────────────────────────────

// Subscribe 订阅主题（支持 Trie 通配符路径，如 "chat/*"）
//
//	client.Subscribe("chat/room1", func(msg mesh.Message, params map[string]string) {
//	    fmt.Printf("收到 %s 的消息: %v\n", msg.SenderID, msg.Value)
//	})
func (c *Client) Subscribe(topic string, callback func(msg Message, pathValues map[string]string)) error {
	if !c.IsRunning() {
		return ErrClientNotRunning
	}
	return c.node.Subscribe(topic, callback)
}

// SubscribeWithAck 订阅主题并自动计数（用于内部统计）
func (c *Client) SubscribeWithAck(topic string, callback func(msg Message, pathValues map[string]string)) error {
	wrapped := func(msg Message, pathValues map[string]string) {
		atomic.AddInt64(&c.stats.MessagesRecv, 1)
		if c.opts.OnPub != nil {
			c.opts.OnPub(msg.Path, msg)
		}
		if callback != nil {
			callback(msg, pathValues)
		}
	}
	return c.Subscribe(topic, wrapped)
}

// Publish 发布消息到指定主题
//
//	client.Publish("chat/room1", "hello everyone")
func (c *Client) Publish(topic string, value any) error {
	if !c.IsRunning() {
		return ErrClientNotRunning
	}
	atomic.AddInt64(&c.stats.MessagesSent, 1)
	return c.node.Publish(topic, value)
}

// Put 存储键值对（通过广播实现分布式存储）
//
//	client.Put("config/max_connections", 100)
func (c *Client) Put(key string, value any) error {
	if !c.IsRunning() {
		return ErrClientNotRunning
	}
	atomic.AddInt64(&c.stats.MessagesSent, 1)
	return c.node.Put(key, value)
}

// Get 从集群中查询键值（向所有节点广播请求，任意持有该键的节点响应）
//
//	val, err := client.Get("config/max_connections")
func (c *Client) Get(key string) (any, error) {
	if !c.IsRunning() {
		return nil, ErrClientNotRunning
	}
	return c.node.Get(key)
}

// RegisterQueryHandler 注册 RPC 查询处理器
//
//	client.RegisterQueryHandler("/rpc/echo", func(msg mesh.Message) any {
//	    return map[string]any{"echo": msg.Payload}
//	})
func (c *Client) RegisterQueryHandler(path string, handler func(Message) any) error {
	if !c.IsRunning() {
		return ErrClientNotRunning
	}
	return c.node.RegisterQueryHandler(path, func(msg Message) any {
		atomic.AddInt64(&c.stats.MessagesRecv, 1)
		return handler(msg)
	})
}

// Query 发起 RPC 查询（向任意持有该 path 的节点发送请求并等待响应）
//
//	result, err := client.Query("/rpc/echo", "hello")
func (c *Client) Query(path string, payload any) (any, error) {
	if !c.IsRunning() {
		return nil, ErrClientNotRunning
	}
	atomic.AddInt64(&c.stats.MessagesSent, 1)
	return c.node.Query(path, payload)
}

// ─────────────────────────────────────────────
// 集群信息
// ─────────────────────────────────────────────

// GetMembers 获取当前集群成员列表
func (c *Client) GetMembers() []NodeState {
	if c.node == nil {
		return nil
	}
	return c.node.GetNodeStates()
}

// GetMemberCount 获取在线节点数量
func (c *Client) GetMemberCount() int {
	return len(c.GetMembers())
}

// GetHealth 获取健康状态快照
func (c *Client) GetHealth() HealthInfo {
	nodes := c.GetMembers()
	nodeIDs := make([]string, len(nodes))
	for i, n := range nodes {
		nodeIDs[i] = n.ID
	}

	return HealthInfo{
		Timestamp:   time.Now(),
		NodeID:      c.opts.ID,
		NodeAddr:    c.opts.Addr,
		OnlineNodes: nodeIDs,
		NodeCount:   len(nodes),
		Uptime:      time.Since(c.stats.StartTime),
		Sent:        atomic.LoadInt64(&c.stats.MessagesSent),
		Recv:        atomic.LoadInt64(&c.stats.MessagesRecv),
	}
}

// GetStats 获取运行时统计
func (c *Client) GetStats() ClientStats {
	stats := ClientStats{
		StartTime:    c.stats.StartTime,
		MessagesSent: atomic.LoadInt64(&c.stats.MessagesSent),
		MessagesRecv: atomic.LoadInt64(&c.stats.MessagesRecv),
		OnlineNodes:  int64(c.GetMemberCount()),
		JoinEvents:   atomic.LoadInt64(&c.stats.JoinEvents),
		LeaveEvents:  atomic.LoadInt64(&c.stats.LeaveEvents),
	}

	// 汇总传输层的背压与编解码统计（传输层实现可选提供 GetStats）
	if c.node != nil {
		if collector, ok := c.node.transport.(interface{ GetStats() TransportStats }); ok {
			transport := collector.GetStats()
			stats.MessagesDropped = transport.SendQueueFull + transport.BroadcastDropped
			stats.EncodeFailed = transport.EncodeFailed
			stats.DecodeFailed = transport.DecodeFailed
		}
	}

	return stats
}

// ─────────────────────────────────────────────
// 生命周期
// ─────────────────────────────────────────────

// IsRunning 客户端是否在运行中
func (c *Client) IsRunning() bool {
	return atomic.LoadInt32(&c.running) == 1 && atomic.LoadInt32(&c.closed) == 0
}

// Close 优雅关闭客户端
//
//   - 停止接受新请求
//   - 等待进行中的请求完成（有超时限制）
//   - 关闭底层 Node
//   - 清理资源
func (c *Client) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return ErrClientClosed
	}

	c.log().Info("正在关闭 Mesh 客户端...")
	atomic.StoreInt32(&c.running, 0)

	// 取消上下文，通知后台任务退出
	c.cancel()

	// 等待后台任务退出
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.log().Debug("后台任务已全部退出")
	case <-time.After(5 * time.Second):
		c.log().Warn("等待后台任务退出超时，强制关闭")
	}

	// 关闭底层 Node
	if c.node != nil {
		c.node.Stop()
	}

	c.log().Info("Mesh 客户端已关闭: 运行时长 %s, 发送 %d 条, 接收 %d 条",
		time.Since(c.stats.StartTime).Round(time.Second),
		atomic.LoadInt64(&c.stats.MessagesSent),
		atomic.LoadInt64(&c.stats.MessagesRecv),
	)

	return nil
}

// ─────────────────────────────────────────────
// 后台任务
// ─────────────────────────────────────────────

// loopHealthCheck 定期健康检查
func (c *Client) loopHealthCheck() {
	defer c.wg.Done()

	interval := c.opts.HealthCheckInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if c.opts.OnHealthCheck != nil {
				c.opts.OnHealthCheck(c.GetHealth())
			}
		}
	}
}

// loopStatsUpdate 定期更新在线节点统计
func (c *Client) loopStatsUpdate() {
	defer c.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			atomic.StoreInt64(&c.stats.OnlineNodes, int64(c.GetMemberCount()))
		}
	}
}

// ─────────────────────────────────────────────
// 日志
// ─────────────────────────────────────────────

// SetLogger 注入自定义日志实现
//   - 可在运行期调用：内部使用 atomic.Pointer 存储，与后台 goroutine 的日志读写并发安全
func (c *Client) SetLogger(logger Logger) {
	if logger != nil {
		c.logger.Store(&loggerHolder{logger: logger})
	}
}

// log 返回当前日志实现（并发安全）
func (c *Client) log() Logger {
	if holder := c.logger.Load(); holder != nil && holder.logger != nil {
		return holder.logger
	}

	return newDefaultLogger("")
}

func newDefaultLogger(nodeID string) Logger {
	return &defaultLogger{
		nodeID: nodeID,
		logger: log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
	}
}

func (l *defaultLogger) Debug(format string, args ...any) {
	l.logger.Printf("[DEBUG] [%s] "+format, append([]any{l.nodeID}, args...)...)
}

func (l *defaultLogger) Info(format string, args ...any) {
	l.logger.Printf("[INFO]  [%s] "+format, append([]any{l.nodeID}, args...)...)
}

func (l *defaultLogger) Warn(format string, args ...any) {
	l.logger.Printf("[WARN]  [%s] "+format, append([]any{l.nodeID}, args...)...)
}

func (l *defaultLogger) Error(format string, args ...any) {
	l.logger.Printf("[ERROR] [%s] "+format, append([]any{l.nodeID}, args...)...)
}

// generateNodeID 生成默认节点 ID
func generateNodeID(addr string) string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "node"
	}
	return fmt.Sprintf("%s-%s-%d", hostname, addr, time.Now().UnixNano()%100000)
}
