package mesh

import (
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"
)

// ******************************************************************************************************************
// Gossip：负责成员管理、心跳传播、故障检测，也注册为 MessageHandler 处理 Gossip 消息
// ******************************************************************************************************************

type (
	// Gossip 连接管理接口（Gossip 协议）
	Gossip interface {
		// Start 启动
		Start()
		// RegisterHandler 注册处理程序（用于通知上层节点变更）
		RegisterHandler(handler GossipHandler)
		// GetNodes 获取节点列表
		GetNodes() []NodeState
		// Stop 停止
		Stop()

		// HandleGossipMessage 处理 Gossip 消息（由 MessageBus 回调）
		HandleGossipMessage(msg Message, fromAddr string)
	}

	// GossipHandler Gossip 处理器接口
	GossipHandler interface {
		// CallOnJoin 加入集群时调用
		CallOnJoin(nodeID, addr string)
		// CallOnLeave 退出集群时调用
		CallOnLeave(nodeID string)
	}

	// NodeState 节点状态
	NodeState struct {
		// 该节点生成的最新消息序列号
		//   - Gossip 传播时，接收方会比较自己的记录：如果收到的 SN 更大，则更新本地状态，表明该节点的信息更新
		SN int64
		// 节点唯一标识符，通常是一个随机生成的字符串或 UUID
		ID string
		// 节点的网络地址，格式 "host:port"（其他节点通过此地址建立 TCP 连接）
		Addr string
		// 最后一次收到该节点心跳（或任何消息）的时间
		//   - 用于故障检测：如果超过一定时间（如 15 秒）未更新，则认为该节点已失效，将其从成员表中删除
		Timestamp time.Time
	}
)

type defaultGossip struct {
	// 服务ID
	id string
	// 服务地址
	addr string
	// 配置信息
	config *Config
	// 用于接收 gossip 消息（实际通过 transport 回调处理）
	bus MessageBus
	// 通信网络
	transport Transport
	// 种子节点
	seedAddresses []string
	// Gossip 节点
	nodes map[string]*NodeState
	// 节点加入和退出集群时的处理器
	handler GossipHandler
	// 服务停止通道
	stopChan chan struct{}
	// 节点状态序列号（incarnation）：每次心跳递增，接收方据此判断节点是否仍存活
	sn atomic.Int64
	// 防重复关闭
	stopped int32
	lock    sync.RWMutex
}

// NewGossipService 新建一个 Gossip 服务
func NewGossipService(id, addr string, transport Transport, bus MessageBus, seedAddresses []string, config *Config) Gossip {
	gossip := &defaultGossip{
		id:            id,
		addr:          addr,
		transport:     transport,
		bus:           bus,
		config:        config,
		seedAddresses: seedAddresses,
		nodes:         make(map[string]*NodeState),
		stopChan:      make(chan struct{}),
	}

	// 注册自身
	gossip.sn.Store(time.Now().UnixNano())
	gossip.nodes[id] = &NodeState{ID: id, Addr: addr, SN: gossip.sn.Load(), Timestamp: time.Now()}
	return gossip
}

// nextSN 生成单调递增的节点状态序列号
func (own *defaultGossip) nextSN() int64 {
	return own.sn.Add(1)
}

// HandleGossipMessage 实现 GossipMessageHandler 接口，处理来自 MessageBus 的 Gossip 消息
func (own *defaultGossip) HandleGossipMessage(msg Message, fromAddr string) {
	// 只处理 Gossip 消息
	if msg.Type == MESSAGE_TYPE_GOSSIP_SYNC || msg.Type == MESSAGE_TYPE_GOSSIP_HEARTBEAT {
		own.handleHeartbeat(msg, fromAddr)
	}
}

// handleHeartbeat 处理心跳消息
func (own *defaultGossip) handleHeartbeat(msg Message, fromAddr string) {
	// 更新发送者心跳时间
	now := time.Now()

	// 优先使用 Gossip 消息中声明的地址（监听地址），回退为 transport 层的连接地址（可能是临时端口）
	nodeAddr := fromAddr
	if sn, ok := msg.GossipNodes[msg.SenderID]; ok && sn.Addr != "" {
		nodeAddr = sn.Addr
	}

	own.lock.Lock()

	if state, ok := own.nodes[msg.SenderID]; ok {
		state.Timestamp = now
		// 如果地址变更，更新地址（避免因临时端口导致后续通信失败）
		if nodeAddr != state.Addr {
			state.Addr = nodeAddr
		}
		if sn, ok := msg.GossipNodes[msg.SenderID]; ok && sn.SN > state.SN {
			state.SN = sn.SN
		}
	} else {
		// 新节点，需要主动连接
		state := &NodeState{
			ID:        msg.SenderID,
			Addr:      nodeAddr,
			Timestamp: now,
		}
		own.nodes[msg.SenderID] = state

		if own.handler != nil {
			go own.handler.CallOnJoin(msg.SenderID, nodeAddr)
		}

		// 只有在监听地址与当前连接地址不同时才新建连接
		if nodeAddr != fromAddr {
			go own.transport.AddPeer(nodeAddr)
		}
	}

	own.lock.Unlock()

	// 合并 gossip 节点信息（在锁外收集需要添加的节点，避免双重加锁）
	var newJoins []NodeState
	own.lock.Lock()
	for id, state := range msg.GossipNodes {
		if existing, ok := own.nodes[id]; ok {
			if state.SN > existing.SN {
				// 序列号递增说明该节点仍在产生心跳，刷新观测时间，防止间接节点被误剔除
				existing.SN = state.SN
				existing.Timestamp = now
				if state.Addr != "" {
					existing.Addr = state.Addr
				}
			}

			continue
		}

		// 第三方节点：沿用消息携带的观测时间，不能用本地接收时间
		//   - 若用本地时间，已停止节点会被其他节点的过期视图反复“复活”，永远不会被故障检测剔除
		//   - 观测时间已过期的条目也不再建档，等上游自行清理即可
		if now.Sub(state.Timestamp) > own.config.GossipFailureTimeout {
			continue
		}

		own.nodes[id] = &NodeState{
			ID:        state.ID,
			SN:        state.SN,
			Addr:      state.Addr,
			Timestamp: state.Timestamp,
		}
		newJoins = append(newJoins, state)
	}
	own.lock.Unlock()

	// 在锁外通知 handler 并注册 peers
	for _, node := range newJoins {
		if own.handler != nil {
			go own.handler.CallOnJoin(node.ID, node.Addr)
		}
		go own.transport.AddPeer(node.Addr)
	}
}

// Start 启动 Gossip 服务
func (own *defaultGossip) Start() {
	// 连接种子节点
	for _, seed := range own.seedAddresses {
		if seed != own.addr {
			own.transport.AddPeer(seed)
		}
	}

	// 如果有种子节点，立即发送 SYNC 消息宣告自身存在
	// （避免新节点等待第一次 Gossip 间隔后才被种子发现）
	if len(own.seedAddresses) > 0 {
		go own.initialSync()
	}

	// 启动 Gossip 定时任务
	go own.loopGossip()
	go own.loopDetectFailure()
}

// initialSync 发送初始 SYNC 消息，向种子节点宣告自身存在
func (own *defaultGossip) initialSync() {
	// 等待 TCP 连接建立完成
	time.Sleep(200 * time.Millisecond)

	own.lock.RLock()
	sn := own.nextSN()
	owns := map[string]NodeState{
		own.id: {ID: own.id, Addr: own.addr, SN: sn, Timestamp: time.Now()},
	}
	own.lock.RUnlock()

	if err := own.transport.Broadcast(Message{
		SN:          sn,
		Type:        MESSAGE_TYPE_GOSSIP_SYNC,
		SenderID:    own.id,
		GossipNodes: owns,
	}); err != nil {
		logError("Failed to send initial sync: %v", err)
	}
}

func (own *defaultGossip) loopGossip() {
	ticker := time.NewTicker(own.config.GossipInterval)
	defer ticker.Stop()

	for {
		select {
		case <-own.stopChan:
			return
		case <-ticker.C:
			own.propagate()
		}
	}
}

// propagate 携带部分视图，扇出到部分节点
func (own *defaultGossip) propagate() {
	// 获取不含自身的所有节点
	peers := own.getNodesWithoutOneself()
	peerLen := len(peers)
	if peerLen == 0 {
		return
	}

	// 构建部分视图并始终附加自身的 NodeState（接收方需要从中获取真实的监听地址）
	buildMsg := func(partial map[string]NodeState) Message {
		if partial == nil {
			partial = make(map[string]NodeState)
		}
		sn := own.nextSN()
		partial[own.id] = NodeState{ID: own.id, Addr: own.addr, SN: sn, Timestamp: time.Now()}
		return Message{
			SN:          sn,
			Type:        MESSAGE_TYPE_GOSSIP_HEARTBEAT,
			SenderID:    own.id,
			GossipNodes: partial,
		}
	}

	// 随机取样 N 个节点
	fanoutSize := own.config.GossipFanoutSize
	if peerLen > fanoutSize {
		// 打乱取样
		rand.Shuffle(peerLen, func(i, j int) { peers[i], peers[j] = peers[j], peers[i] })
		// 获取前 N 个需要广播的节点
		fanoutPeers := peers[:fanoutSize]
		// 获取不包含广播节点的 M 个视图
		partial := own.getPartialView(fanoutPeers)
		msg := buildMsg(partial)
		// 扇出到各个广播节点
		for _, peer := range fanoutPeers {
			if err := own.transport.SendTo(peer.Addr, msg); err != nil {
				logError("Failed to send heartbeat to peer %s: %v", peer.Addr, err)
			}
		}
		return
	}

	// 节点数不超过扇出数，全量广播（也携带部分视图）
	partial := own.getPartialView(nil)
	msg := buildMsg(partial)
	if err := own.transport.Broadcast(msg); err != nil {
		logError("Failed to broadcast heartbeat: %v", err)
	}
}

func (own *defaultGossip) getPartialView(fanoutPeers []NodeState) map[string]NodeState {
	// 收集符合条件的候选节点
	own.lock.RLock()
	candidates := make([]NodeState, 0, len(own.nodes))
	for _, node := range own.nodes {
		// 忽略自身
		if node.Addr == own.addr {
			continue
		}

		// 忽略广播节点
		var skip bool
		for _, peer := range fanoutPeers {
			if peer.Addr == node.Addr {
				skip = true
				break
			}
		}

		if !skip {
			candidates = append(candidates, *node)
		}
	}
	own.lock.RUnlock()

	views := make(map[string]NodeState)

	// 随机取样
	partialSize := own.config.GossipPartialViewSize
	if len(candidates) > partialSize {
		rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
		for i := range partialSize {
			peer := candidates[i]
			views[peer.ID] = peer
		}

		return views
	}

	// 全部获取
	for _, peer := range candidates {
		views[peer.ID] = peer
	}

	return views
}

// loopDetectFailure 循环检测故障
//   - 超时没有更新心跳，则剔除
func (own *defaultGossip) loopDetectFailure() {
	ticker := time.NewTicker(own.config.GossipFailureTimeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-own.stopChan:
			return
		case <-ticker.C:
			now := time.Now()
			own.lock.Lock()
			for id, state := range own.nodes {
				if id == own.id {
					continue
				}

				// 检测心跳是否超时
				if now.Sub(state.Timestamp) > own.config.GossipFailureTimeout {
					delete(own.nodes, id)
					if own.handler != nil {
						go own.handler.CallOnLeave(id)
					}
				}
			}
			own.lock.Unlock()
		}
	}
}

// RegisterHandler 注册处理程序（用于通知上层节点变更）
func (own *defaultGossip) RegisterHandler(handler GossipHandler) {
	own.lock.Lock()
	own.handler = handler
	own.lock.Unlock()
}

// GetNodes 获取所有 Gossip 节点信息
func (own *defaultGossip) GetNodes() []NodeState {
	own.lock.RLock()
	nodes := make([]NodeState, 0, len(own.nodes))
	for _, state := range own.nodes {
		nodes = append(nodes, *state)
	}
	own.lock.RUnlock()

	return nodes
}

// getNodesWithoutOneself 获取不含本身的所有 Gossip 节点信息
func (own *defaultGossip) getNodesWithoutOneself() []NodeState {
	own.lock.RLock()
	nodes := make([]NodeState, 0, len(own.nodes))
	for _, state := range own.nodes {
		if state.ID != own.id {
			nodes = append(nodes, *state)
		}
	}
	own.lock.RUnlock()

	return nodes
}

// Stop 停止 Gossip 服务
func (own *defaultGossip) Stop() {
	if atomic.CompareAndSwapInt32(&own.stopped, 0, 1) {
		close(own.stopChan)
	}
}
