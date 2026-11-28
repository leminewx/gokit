package mesh

// ******************************************************************************************************************
// Node：一个去中心化、部分连通、多跳路由、支持 Gossip 的拓扑结构
// ******************************************************************************************************************

// Node 去中心化的分布式节点
type Node struct {
	// 节点ID
	id string
	// 节点地址
	addr string
	// 全局配置
	config *Config
	// 底层通信接口
	transport Transport
	// 消息总线接口
	bus MessageBus
	// Gossip 接口
	gossip Gossip
}

// defaultGossipHandler 默认的节点变更处理器（空实现）
type defaultGossipHandler struct{}

func (h *defaultGossipHandler) CallOnJoin(nodeID, addr string) {}
func (h *defaultGossipHandler) CallOnLeave(nodeID string)      {}

// NewNode 新建一个去中心化的分布式节点
func NewNode(id, addr string, maxPeers int, seedAddresses []string, opts ...Option) (*Node, error) {
	// 创建配置
	config := NewDefaultConfig()
	config.seedAddresses = seedAddresses
	for _, opt := range opts {
		opt(config)
	}

	// 创建底层通信、消息总线、Gossip 管理
	transport := NewTCPTransport(addr, maxPeers, config)
	bus := NewMessageBus(id, transport, config)
	gossip := NewGossipService(id, addr, transport, bus, seedAddresses, config)

	// 注册消息处理器：Transport → Bus → Gossip
	transport.AddHandler(bus)
	bus.SetGossipHandler(gossip)

	// 注册 Gossip 节点变更处理器（可选，供上层监听节点加入/离开）
	gossip.RegisterHandler(&defaultGossipHandler{})

	// 启动底层通信模块
	if err := transport.Start(); err != nil {
		return nil, err
	}

	// 启动 Gossip 服务
	gossip.Start()

	// 创建节点
	return &Node{
		id:        id,
		addr:      addr,
		transport: transport,
		gossip:    gossip,
		bus:       bus,
		config:    config,
	}, nil
}

// Subscribe 订阅消息
func (own *Node) Subscribe(topic string, callbacks ...func(msg Message, pathValues map[string]string)) error {
	return own.bus.Subscribe(topic, callbacks...)
}

// Publish 推送消息
func (own *Node) Publish(topic string, value any) error {
	return own.bus.Publish(topic, value)
}

// Put 存储数据
func (own *Node) Put(key string, value any) error {
	return own.bus.Put(key, value)
}

// Get 查询数据
func (own *Node) Get(key string) (any, error) {
	return own.bus.Get(key)
}

// RegisterQueryHandler 注册查询接口（RPC）
func (own *Node) RegisterQueryHandler(path string, handler func(Message) any) error {
	return own.bus.RegisterQueryHandler(path, handler)
}

// Query 查询数据（RPC）
func (own *Node) Query(path string, payload any) (any, error) {
	return own.bus.Query(path, payload)
}

// GetNodeStates 获取所有节点状态
func (own *Node) GetNodeStates() []NodeState {
	return own.gossip.GetNodes()
}

// Stop 停止当前节点
func (own *Node) Stop() {
	own.gossip.Stop()
	own.bus.Stop()
	own.transport.Close()
}
