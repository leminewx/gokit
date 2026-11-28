package mesh

import (
	"sync"
	"sync/atomic"
	"time"

	"gitee.com/leminewx/gokit/structures/tree"
)

// ******************************************************************************************************************
// MessageBus：负责业务逻辑（Pub/Sub、Put/Get KV、RPC）、去重、请求-响应匹配，依赖 Transport 发送消息
// ******************************************************************************************************************

type (
	// MessageBus 消息总线接口
	MessageBus interface {
		// Subscribe 订阅消息
		Subscribe(topic string, callbacks ...func(msg Message, pathValues map[string]string)) error
		// Publish 发布消息
		Publish(topic string, value any) error
		// Put 存储数据
		Put(key string, value any) error
		// Get 获取数据
		Get(key string) (any, error)
		// RegisterQueryHandler 注册查询接口（RPC）
		RegisterQueryHandler(path string, handler func(Message) any) error
		// Query 查询数据
		Query(path string, payload any) (any, error)

		// HandleRemoteMessage 内部由 Transport 回调
		HandleRemoteMessage(msg Message, fromAddr string)
		// SetGossipHandler 设置 Gossip 消息处理器
		SetGossipHandler(handler GossipMessageHandler)
		// Stop 停止消息总线
		Stop()
	}

	// GossipMessageHandler Gossip 消息处理器
	GossipMessageHandler interface {
		HandleGossipMessage(msg Message, fromAddr string)
	}

	// Message 消息类型
	Message struct {
		// 全局唯一序列号，通常用 time.Now().UnixNano() 生成
		//   - 用于消息去重：每个节点维护一个 seen 集合，收到相同 SN 的消息直接丢弃，防止广播环路和重复处理。
		SN int64 `json:"sn"`
		// 生存时间（Time To Live），每经过一次转发减 1
		//   - 当 TTL <= 0 时不再转发
		//   - 用于限制广播范围，防止消息无限扩散，仅对需要多跳广播的消息有效
		TTL int64 `json:"ttl,omitempty"`
		// 消息类型标识，用于路由
		//   - 例如：gossip_heartbeat、gossip_sync、pub、put、get、get_resp、query_resp......
		Type MessageType `json:"type"`
		// 消息原始发送节点的唯一标识符
		//   - 用于追踪消息来源、避免回环（不发给发送者）以及响应时定位目标节点
		SenderID string `json:"sender_id"`

		// 主题路径或键名
		//   - 用于 Hive 业务：发布/订阅的主题、存储的键、可查询节点的路径（类似文件系统路径，支持通配符，可扩展）
		Path string `json:"path,omitempty"`
		// 业务数据值
		//   - 用于 pub（发布的消息内容）、put（存储的值）、get_resp（查询结果）等
		Value any `json:"value,omitempty"`
		// 附加载荷，主要用于 Query（RPC） 场景
		//   - 例如 query 中携带调用参数，query_resp 中携带返回结果
		//   - 与 Value 语义上分开，避免混淆
		Payload any `json:"payload,omitempty"`
		// 请求 ID，用于匹配请求和响应
		//   - 当节点发起 get 或 query 时生成一个本地唯一的 RequestID，并保存在 pending 表中
		//   - 响应消息必须携带相同的 RequestID，以便发起方将响应交付给等待的调用者
		RequestID int64 `json:"request_id,omitempty"`

		// 携带的节点状态快照，用于 Gossip 协议
		//   - 每个条目包含一个节点的 ID 及其状态（地址、序列号、时间戳）
		//   - 节点收到后会合并到自己的成员表中
		GossipNodes map[string]NodeState `json:"gossip_nodes,omitempty"`
	}

	// MessageType 消息类型
	MessageType string
)

const (
	MESSAGE_TYPE_PUB            MessageType = "pub"        // 发布消息（无需响应）
	MESSAGE_TYPE_PUT            MessageType = "put"        // 存储消息（无需响应）
	MESSAGE_TYPE_GET            MessageType = "get"        // 获取消息请求
	MESSAGE_TYPE_GET_RESPONSE   MessageType = "get_resp"   // 获取消息响应
	MESSAGE_TYPE_QUERY          MessageType = "query"      // 查询节点请求
	MESSAGE_TYPE_QUERY_RESPONSE MessageType = "query_resp" // 查询节点响应

	MESSAGE_TYPE_GOSSIP_SYNC      MessageType = "gossip_sync"      // Gossip 同步消息
	MESSAGE_TYPE_GOSSIP_HEARTBEAT MessageType = "gossip_heartbeat" // Gossip 心跳消息
)

// defaultMessageBus 默认消息总线
type defaultMessageBus struct {
	// 节点自身唯一标识
	id string
	// 配置信息
	config *Config
	// 底层通信
	transport Transport

	// 业务订阅：键为主题路径，值为该主题上注册的回调函数列表（当收到 pub 消息且路径匹配时，调用所有回调）
	subscribeTree *tree.Trie[[]func(Message, map[string]string)]
	subscribeLock sync.RWMutex

	// 本地键值存储：用于 put 和 get 操作（分布式场景下通过广播保证最终一致性）
	store     map[string]any
	storeLock sync.RWMutex

	// 可查询节点表：键为路径，值为处理函数（当收到 query 且路径匹配时，执行该函数并返回结果）
	//   - 实现 RPC 风格的服务注册
	queryHandlers map[string]func(Message) any
	queryLock     sync.RWMutex

	// 等待响应的请求表（键为 RequestID，值为一个 channel，用于阻塞等待响应）
	//   - 发起 get 或 query 时创建条目，收到响应后通过 channel 唤醒调用者
	pending     map[pendingKey]chan Message
	pendingLock sync.RWMutex

	// 已处理的消息去重键集合（防止因广播环路导致重复处理）
	//   - 键为发送者 ID + 序列号，避免不同节点的 SN 取值相同而误判为重复
	//   - 为避免无限增长，可定期清理（例如保留最近 1 分钟内的记录）
	seen map[seenKey]time.Time
	// 停止消息清理任务
	seenStop chan struct{}
	seenLock sync.RWMutex

	// Gossip 消息处理器
	gossipHandler GossipMessageHandler
	// 请求 ID 生成器
	requestIDGenerator int64
	// 消息序列号生成器（单调递增）
	//   - 不能直接用 time.Now().UnixNano()：Windows 等平台的系统时钟粒度较粗，
	//     连续发布的几条消息可能拿到相同的纳秒值，被接收方去重逻辑误判为重复消息
	msgSN atomic.Int64
	// 防重复关闭
	stopped int32
}

type pendingKey struct {
	senderID  string
	requestID int64
}

// seenKey 消息去重键
//   - 只用 SN 去重会跨节点冲突，故同时带上发送者 ID
//   - 还需带上消息类型：Gossip 的序列号与业务消息的序列号各自单调递增，数值区间可能重叠，
//     仅凭 (发送者, SN) 会把同号的心跳与业务消息误判为重复
//   - 多跳转发会保留原始 SenderID、SN 与 Type，因此同一消息的不同副本仍能被正确识别
type seenKey struct {
	senderID string
	sn       int64
	msgType  MessageType
}

// NewMessageBus 新建一个消息总线
func NewMessageBus(id string, transport Transport, config *Config) MessageBus {
	bus := &defaultMessageBus{
		id:            id,
		transport:     transport,
		config:        config,
		subscribeTree: tree.NewTrie[[]func(Message, map[string]string)](),
		store:         make(map[string]any),
		queryHandlers: make(map[string]func(Message) any),
		pending:       make(map[pendingKey]chan Message),
		seen:          make(map[seenKey]time.Time),
		seenStop:      make(chan struct{}),
	}

	bus.msgSN.Store(time.Now().UnixNano())
	go bus.loopCleanSeen()
	return bus
}

// nextSN 生成单调递增的消息序列号
func (own *defaultMessageBus) nextSN() int64 {
	return own.msgSN.Add(1)
}

func (own *defaultMessageBus) loopCleanSeen() {
	// 以 MaxSeenAge 为间隔进行清理
	interval := own.config.MaxSeenAge
	if interval < time.Second {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			own.cleanSeen()

		case <-own.seenStop:
			own.config.logDebug("Stopped to clean message sns")
			return
		}
	}
}

// cleanSeen 定期清理 seen（由外部调用）
func (own *defaultMessageBus) cleanSeen() {
	now := time.Now()

	own.seenLock.Lock()
	defer own.seenLock.Unlock()

	for key, ts := range own.seen {
		if now.Sub(ts) > own.config.MaxSeenAge {
			delete(own.seen, key)
		}
	}
}

// generateRequestID 生成请求 ID（本地唯一）
func (own *defaultMessageBus) generateRequestID() int64 {
	return atomic.AddInt64(&own.requestIDGenerator, 1)
}

// sendRequest 发送请求并等待响应
func (own *defaultMessageBus) sendRequest(msg Message) (Message, error) {
	// 初始化消息
	msg.SN = own.nextSN()
	msg.SenderID = own.id
	msg.RequestID = own.generateRequestID()

	// 放入请求等待队列
	key := pendingKey{senderID: own.id, requestID: msg.RequestID}
	respChan := make(chan Message, 1)
	own.pendingLock.Lock()
	own.pending[key] = respChan
	own.pendingLock.Unlock()
	defer func() {
		own.pendingLock.Lock()
		delete(own.pending, key)
		own.pendingLock.Unlock()
	}()

	// 广播消息
	if err := own.transport.Broadcast(msg); err != nil {
		return Message{}, err
	}

	// 等待响应
	select {
	case resp := <-respChan:
		return resp, nil
	case <-time.After(own.config.RequestTimeout):
		return Message{}, wrapError(ErrRequestTimeout, "to %s", msg.Path)
	}
}

// Subscribe 订阅消息
func (own *defaultMessageBus) Subscribe(topic string, callbacks ...func(msg Message, pathValues map[string]string)) error {
	calls := make([]func(msg Message, params map[string]string), 0, len(callbacks))
	for _, callback := range callbacks {
		if callback != nil {
			calls = append(calls, callback)
		}
	}

	if len(calls) == 0 {
		return wrapError(ErrNoSubscriber, "%s", topic)
	}

	// Trie 不是并发安全的，Subscribe 与 handlePub 的 Search 必须用锁隔离；
	// 同一主题重复订阅时追加而不是覆盖已有回调。
	own.subscribeLock.Lock()
	defer own.subscribeLock.Unlock()

	if existing, ok := own.subscribeTree.Get(topic); ok {
		calls = append(existing, calls...)
	}

	own.subscribeTree.Insert(topic, calls)
	return nil
}

// Publish 发布消息
func (own *defaultMessageBus) Publish(topic string, value any) error {
	return own.transport.Broadcast(Message{
		SN:       own.nextSN(),
		TTL:      int64(own.config.MessageTTL),
		Type:     MESSAGE_TYPE_PUB,
		Path:     topic,
		Value:    value,
		SenderID: own.id,
	})
}

// Put 存储消息
func (own *defaultMessageBus) Put(key string, value any) error {
	msg := Message{
		SN:       own.nextSN(),
		TTL:      int64(own.config.MessageTTL),
		Type:     MESSAGE_TYPE_PUT,
		Path:     key,
		Value:    value,
		SenderID: own.id,
	}

	// 先本地落库，保证读己之写；对端没有副本或节点数小于 2 时仍能读到
	own.handlePut(msg)

	return own.transport.Broadcast(msg)
}

// Get 获取消息
//   - 优先读本地副本；未命中时向集群广播请求，由持有该键的节点响应
func (own *defaultMessageBus) Get(key string) (any, error) {
	own.storeLock.RLock()
	val, ok := own.store[key]
	own.storeLock.RUnlock()
	if ok {
		return val, nil
	}

	// 发送消息
	resp, err := own.sendRequest(Message{Type: MESSAGE_TYPE_GET, Path: key})
	if err != nil {
		return nil, err
	}

	// 检查响应消息类型
	if resp.Type != MESSAGE_TYPE_GET_RESPONSE {
		return nil, wrapError(ErrUnexpectedResponse, "%s, want %s", resp.Type, MESSAGE_TYPE_GET_RESPONSE)
	}

	return resp.Value, nil
}

// RegisterQueryHandler 注册查询接口（RPC）
func (own *defaultMessageBus) RegisterQueryHandler(path string, handler func(Message) any) error {
	if handler == nil {
		return wrapError(ErrNilQueryHandler, "%s", path)
	}

	own.queryLock.Lock()
	own.queryHandlers[path] = handler
	own.queryLock.Unlock()
	return nil
}

// Query 查询数据
func (own *defaultMessageBus) Query(path string, payload any) (any, error) {
	// 发送消息
	resp, err := own.sendRequest(Message{Type: MESSAGE_TYPE_QUERY, Path: path, Payload: payload})
	if err != nil {
		return nil, err
	}

	// 检查响应消息类型
	if resp.Type != MESSAGE_TYPE_QUERY_RESPONSE {
		return nil, wrapError(ErrUnexpectedResponse, "%s, want %s", resp.Type, MESSAGE_TYPE_QUERY_RESPONSE)
	}
	return resp.Value, nil
}

// SetGossipHandler 设置 Gossip 消息处理器
func (own *defaultMessageBus) SetGossipHandler(handler GossipMessageHandler) {
	own.gossipHandler = handler
}

// Stop 停止消息总线
func (own *defaultMessageBus) Stop() {
	if atomic.CompareAndSwapInt32(&own.stopped, 0, 1) {
		close(own.seenStop)
	}
}

// HandleRemoteMessage 处理来自 Transport 的消息
func (own *defaultMessageBus) HandleRemoteMessage(msg Message, fromAddr string) {
	// 1. 去重（以发送者 + 序列号 + 消息类型为键）
	key := seenKey{senderID: msg.SenderID, sn: msg.SN, msgType: msg.Type}
	own.seenLock.Lock()
	if _, ok := own.seen[key]; ok {
		own.seenLock.Unlock()
		return
	}
	own.seen[key] = time.Now()
	own.seenLock.Unlock()

	// 2.多跳转发
	if msg.TTL > 0 {
		msg.TTL--
		go own.forwardMessage(msg, fromAddr)
	}

	// 3. 根据消息类型处理
	switch msg.Type {
	case MESSAGE_TYPE_PUB: // 推送消息
		own.handlePub(msg)
	case MESSAGE_TYPE_PUT: // 存储数据
		own.handlePut(msg)
	case MESSAGE_TYPE_GET: // 查询数据
		own.handleGet(msg, fromAddr)
	case MESSAGE_TYPE_QUERY: // 查询数据（RPC）
		own.handleQuery(msg, fromAddr)
	case MESSAGE_TYPE_GET_RESPONSE, MESSAGE_TYPE_QUERY_RESPONSE: // 响应查询
		key := pendingKey{senderID: own.id, requestID: msg.RequestID}
		own.pendingLock.RLock()
		ch, ok := own.pending[key]
		if own.pendingLock.RUnlock(); ok {
			select {
			case ch <- msg:
			default:
				logError("Send queue is full to %s", fromAddr)
			}
		}
	case MESSAGE_TYPE_GOSSIP_HEARTBEAT, MESSAGE_TYPE_GOSSIP_SYNC: // Gossip 消息
		if own.gossipHandler != nil {
			own.gossipHandler.HandleGossipMessage(msg, fromAddr)
		}
	}
}

func (own *defaultMessageBus) forwardMessage(msg Message, excludeAddr string) {
	if err := own.transport.Broadcast(msg, excludeAddr); err != nil {
		logError("Failed to send message: %v", err)
	}
}

func (own *defaultMessageBus) handlePub(msg Message) {
	// 与 Subscribe 的写入互斥，避免并发读写 Trie 导致 data race
	own.subscribeLock.RLock()
	callbacks, params, ok := own.subscribeTree.Search(msg.Path)
	own.subscribeLock.RUnlock()
	if !ok {
		return
	}

	for _, callback := range callbacks {
		go callback(msg, params)
	}
}

func (own *defaultMessageBus) handlePut(msg Message) {
	own.storeLock.Lock()
	own.store[msg.Path] = msg.Value
	own.storeLock.Unlock()
}

func (own *defaultMessageBus) handleGet(msg Message, fromAddr string) {
	// 读取数据
	own.storeLock.RLock()
	val, ok := own.store[msg.Path]
	if own.storeLock.RUnlock(); !ok {
		return // 本地没有该键，不响应
	}

	// 响应数据
	if err := own.transport.SendTo(fromAddr, Message{
		SN:        own.nextSN(),
		Type:      MESSAGE_TYPE_GET_RESPONSE,
		Path:      msg.Path,
		Value:     val,
		SenderID:  own.id,
		RequestID: msg.RequestID,
	}); err != nil {
		logError("Failed to send message: %v", err)
	}
}

func (own *defaultMessageBus) handleQuery(msg Message, fromAddr string) {
	// 查询数据
	own.queryLock.RLock()
	handler, ok := own.queryHandlers[msg.Path]
	if own.queryLock.RUnlock(); !ok {
		return // 本地没有该键，不响应
	}

	// 处理消息
	result := handler(msg)

	// 响应消息
	if err := own.transport.SendTo(fromAddr, Message{
		SN:        own.nextSN(),
		Type:      MESSAGE_TYPE_QUERY_RESPONSE,
		Path:      msg.Path,
		Value:     result,
		SenderID:  own.id,
		RequestID: msg.RequestID,
	}); err != nil {
		logError("Failed to send message: %v", err)
	}
}
