package mesh

import (
	"math/rand"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ******************************************************************************************************************
// Transport：负责 net.Conn 连接、消息收发、编解码，通过 MessageHandler 回调将收到的消息向上传递
//
// 淘汰机制：基于 软限制 + 保护期 + 节点价值评分，平衡稳定性、公平性和收敛速度
//   - 软限制：允许连接数短暂超出 maxPeers（如超出 20%），避免频繁淘汰
//   - 保护期：新建立的连接在 protectionPeriod（如 30 秒）内不会被淘汰
//   - 节点价值评分：每个连接有一个动态分数，分数越高越不容易被淘汰
//       -- 评分依据：
//  	 	1）节点年龄（在线时间越长分数越高）
//		 	2）消息活跃度（最近收到消息的次数）
// 		 	3）节点角色（种子节点、长期稳定节点加分）
//       -- 评分机制：
//          1）连接建立时：基础分 100；如果是种子节点额外 +1000
//	 	 	2）每次收到消息（loopReceive 中）：活跃分数 +1（防刷：每分钟最多加10次）
// 		 	3）每隔一段时间（如 1 分钟）：在线时间分数 +1（上限 1000）
// 		 	4）定期衰减：每小时所有连接的分数乘以 0.9，防止无限增长
//
// 淘汰算法：
//   - 当连接数超过软限制时，从可淘汰连接（超过保护期）中随机选择候选集，淘汰其中分数最低的
//
// 主动连接：
//   - 被淘汰的连接如果仍有通信需求（比如对方发送消息），会通过 Gossip 成员表重新建立
// ******************************************************************************************************************

type (
	// Transport 底层网络通信接口
	Transport interface {
		// Start 启动网络通信
		Start() error
		// SendTo 单播给指定连接
		SendTo(addr string, msg Message) error
		// Broadcast 广播给所有连接
		Broadcast(msg Message, skips ...string) error
		// AddPeer 添加节点
		AddPeer(addr string)
		// RemovePeer 移除节点
		RemovePeer(addr string)
		// AddHandler 添加消息处理器
		AddHandler(handler MessageHandler)
		// Close 关闭忘了通信
		Close()
	}

	// MessageHandler 由上层实现，处理收到的消息
	MessageHandler interface {
		HandleRemoteMessage(msg Message, fromAddr string)
	}

	// TransportStats 传输层运行时统计
	//   - 用于观测背压丢包与编解码失败，便于定位慢节点
	TransportStats struct {
		// 单播时因发送队列满而被拒绝的消息数
		SendQueueFull int64
		// 广播时因发送队列满而被跳过的消息数
		BroadcastDropped int64
		// 编码失败次数
		EncodeFailed int64
		// 解码失败次数
		DecodeFailed int64
	}
)

type (
	// tcpTransport TCP 通信
	tcpTransport struct {
		// 节点自身监听地址
		addr string
		// 节点配置
		config *Config
		// 消息处理器
		handlers    []MessageHandler
		handlerLock sync.RWMutex

		// 允许的最大并发连接数
		//   - 超过此值时，新连接会触发淘汰策略（踢掉最不活跃的旧连接）
		maxPeers int
		// 当前活跃连接的表，以远端地址为键（快速查找某个地址是否已连接）
		peers    map[string]*tcpPeer
		peerLock sync.RWMutex

		// 上一次衰减时间
		lastScoreDecay time.Time
		// 上一次维护加分时间（与 lastScoreDecay 一样用"上次执行时间 + 间隔"判定）
		lastMaintenanceScore time.Time
		// 记录每分钟活跃次数，用于防刷
		peerActiveCount map[string]int
		activeCountLock sync.Mutex

		// TCP 监听器，接受其他节点的主动连接
		listener net.Listener
		// 关闭信号，用于优雅停止后台循环
		stopChan chan struct{}
		// 防重复关闭
		closed int32

		// 背压与编解码统计（原子读写）
		sendQueueFull    atomic.Int64
		broadcastDropped atomic.Int64
		encodeFailed     atomic.Int64
		decodeFailed     atomic.Int64
	}

	// tcpPeer TCP 连接
	tcpPeer struct {
		// 远端节点的地址字符串，例如 "192.168.1.10:9001"
		//   - 用于索引和日志输出
		addr string
		// 底层的 TCP 连接对象，用于实际读写数据
		conn net.Conn
		// encode 消息编码函数，复用同一个编码器可以减少内存分配，提高性能
		encode func(msg Message) error
		// 待发送消息的队列
		//   - 发送协程不断从这个 channel 中取出消息，序列化后写入 net.Conn
		//   - 异步发送避免阻塞主逻辑，同时允许一定程度的流量削峰（背压：队列满时丢弃或有限等待）
		sendQueue chan Message
		// 连接关闭信号：关闭后发送协程退出，发送方不再向其投递消息
		done chan struct{}
		// 该连接上一次成功收发消息的时间（UnixNano，原子读写）
		//   - 连接管理：当连接空闲过久或需要为更高优先级的连接腾出位置时，会选择 lastActive 最小的连接断开
		lastActive atomic.Int64
		// 创建时间，用于保护期
		createdAt time.Time
		// 节点价值分数（原子操作）
		score int64
		// 是否是种子节点
		isSeed bool
		// 连接关闭的标志位
		closed int32
	}
)

// NewTCPTransport 新建 TCP 通信
func NewTCPTransport(addr string, maxPeers int, cfg *Config) Transport {
	return &tcpTransport{
		addr:            addr,
		config:          cfg,
		maxPeers:        maxPeers,
		peers:           make(map[string]*tcpPeer),
		peerActiveCount: make(map[string]int),
		stopChan:        make(chan struct{}),
	}
}

// Start 启动
func (own *tcpTransport) Start() error {
	listener, err := net.Listen("tcp", own.addr)
	if err != nil {
		return err
	}

	own.listener = listener
	go own.loopAccept()
	go own.loopMaintenance()
	return nil
}

// loopAccept 接收连接
func (own *tcpTransport) loopAccept() {
	for {
		conn, err := own.listener.Accept()
		if err != nil {
			select {
			case <-own.stopChan:
				own.config.logDebug("Closed the accepted listener")
				return
			default:
				logError("Failed to accept a tcp connection: %v", err)
				continue
			}
		}

		own.config.logDebug("Accepted the connection from %s", conn.RemoteAddr())
		go own.handleConn(conn)
	}
}

// handleConn 处理连接
func (own *tcpTransport) handleConn(conn net.Conn) {
	// 判断否是为种子节点
	isSeed := false
	for _, seed := range own.config.seedAddresses {
		if seed == conn.RemoteAddr().String() {
			isSeed = true
			break
		}
	}

	own.addPeer(own.newPeer(conn.RemoteAddr().String(), conn, isSeed))
}

// newPeer 创建连接对象，统一初始化编解码器与发送队列
func (own *tcpTransport) newPeer(addr string, conn net.Conn, isSeed bool) *tcpPeer {
	return &tcpPeer{
		addr:      addr,
		conn:      conn,
		encode:    own.config.codec().GetEncoder(conn),
		sendQueue: make(chan Message, own.config.MessageSendChanSize),
		done:      make(chan struct{}),
		isSeed:    isSeed,
	}
}

// loopSend 发送循环
func (own *tcpTransport) loopSend(peer *tcpPeer) {
	for {
		select {
		case msg, ok := <-peer.sendQueue:
			if !ok {
				return
			}

			// 编码消息
			if err := peer.encode(msg); err != nil {
				// 发送失败，关闭连接
				own.encodeFailed.Add(1)
				logError("Failed to encode the message: %v", err)
				own.peerLock.Lock()
				own.removePeerWithLock(peer.addr)
				own.peerLock.Unlock()
				return
			}

			// 更新最后活跃时间
			peer.lastActive.Store(time.Now().UnixNano())
		case <-peer.done:
			return
		}
	}
}

// loopReceive 接收循环
func (own *tcpTransport) loopReceive(peer *tcpPeer) {
	decode := own.config.codec().GetDecoder(peer.conn)
	for {
		// 接收消息
		var msg Message
		if err := decode(&msg); err != nil {
			own.decodeFailed.Add(1)
			logError("Failed to decode the message: %v", err)
			own.peerLock.Lock()
			own.removePeerWithLock(peer.addr)
			own.peerLock.Unlock()
			return
		}

		// 更新活跃时间
		peer.lastActive.Store(time.Now().UnixNano())
		// 更新分数
		own.updatePeerScore(peer.addr, _PEER_ACTIVE_DELTA)

		// 处理消息
		own.handlerLock.RLock()
		handlers := append([]MessageHandler{}, own.handlers...)
		own.handlerLock.RUnlock()

		for _, handler := range handlers {
			handler.HandleRemoteMessage(msg, peer.addr)
		}
	}
}

// updatePeerScore 更新节点分数
func (own *tcpTransport) updatePeerScore(addr string, delta int64) {
	own.peerLock.RLock()
	peer, ok := own.peers[addr]
	own.peerLock.RUnlock()
	if !ok {
		return
	}

	// 防刷：对于活跃加分，限制频率
	if delta == _PEER_ACTIVE_DELTA {
		own.activeCountLock.Lock()
		count := own.peerActiveCount[addr]
		if int64(count) >= _PEER_MAX_ACTIVE_INCREMENT_PER_MINUTE {
			own.activeCountLock.Unlock()
			return
		}

		own.peerActiveCount[addr] = count + 1
		own.activeCountLock.Unlock()
	}

	newScore := atomic.AddInt64(&peer.score, delta)
	if newScore > _PEER_SCORE_MAX {
		atomic.StoreInt64(&peer.score, _PEER_SCORE_MAX)
	}
}

// loopMaintenance 定期维护任务
func (own *tcpTransport) loopMaintenance() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-own.stopChan:
			return
		case <-ticker.C:
			// 1. 重置每分钟活跃计数（防刷计数器），该锁与连接表无关，单独持有
			own.activeCountLock.Lock()
			own.peerActiveCount = make(map[string]int)
			own.activeCountLock.Unlock()

			// 2. 淘汰检查必须使用连接表的锁，避免并发读写 peers map
			own.peerLock.Lock()
			own.evictPeersIfNeededWithLock()
			own.peerLock.Unlock()

			// 3. 每小时衰减一次分数
			if time.Since(own.lastScoreDecay) >= _PEER_SCORE_INTERVAL {
				own.decayScores()
				own.lastScoreDecay = time.Now()
			}

			// 4. 通过维护的加分补偿（每 10 分钟一次额外奖励）
			//   - 用"上次执行时间 + 间隔"判定，而不是墙钟分钟取模：
			//     后者依赖 ticker 恰好落在 minute%10==0 的那一分钟内，会漏加或重复加
			if time.Since(own.lastMaintenanceScore) >= _PEER_MAINTENANCE_SCORE_INTERVAL {
				own.addMaintenanceScore()
				own.lastMaintenanceScore = time.Now()
			}
		}
	}
}

// addMaintenanceScore 通过维护的节点进行加分补偿
func (own *tcpTransport) addMaintenanceScore() {
	own.peerLock.RLock()
	peers := make([]*tcpPeer, 0, len(own.peers))
	for _, peer := range own.peers {
		peers = append(peers, peer)
	}
	own.peerLock.RUnlock()

	for _, peer := range peers {
		atomic.AddInt64(&peer.score, 1)
	}
}

// decayScores 衰减分数
func (own *tcpTransport) decayScores() {
	own.peerLock.RLock()
	peers := make([]*tcpPeer, 0, len(own.peers))
	for _, peer := range own.peers {
		peers = append(peers, peer)
	}
	own.peerLock.RUnlock()

	for _, peer := range peers {
		oldScore := atomic.LoadInt64(&peer.score)
		newScore := int64(float64(oldScore) * _PEER_SCORE_DECAY_FACTOR)
		if newScore < 0 {
			newScore = 0
		}
		atomic.StoreInt64(&peer.score, newScore)
	}
}

// SendTo 单播给指定连接
func (own *tcpTransport) SendTo(addr string, msg Message) error {
	own.peerLock.RLock()
	peer, ok := own.peers[addr]
	own.peerLock.RUnlock()
	if !ok {
		return wrapError(ErrPeerNotFound, "%s", addr)
	}

	if atomic.LoadInt32(&peer.closed) == 1 {
		return wrapError(ErrPeerClosed, "%s", addr)
	}

	// 快路径：队列有空位时立即投递
	select {
	case peer.sendQueue <- msg:
		return nil
	case <-peer.done:
		return wrapError(ErrPeerClosed, "%s", addr)
	default:
	}

	// 背压：队列已满时按配置选择丢弃或有限等待
	if own.config.SendBlockTimeout <= 0 {
		own.sendQueueFull.Add(1)
		return wrapError(ErrSendQueueFull, "%s", peer.addr)
	}

	timer := time.NewTimer(own.config.SendBlockTimeout)
	defer timer.Stop()

	select {
	case peer.sendQueue <- msg:
		return nil
	case <-peer.done:
		return wrapError(ErrPeerClosed, "%s", addr)
	case <-timer.C:
		own.sendQueueFull.Add(1)
		return wrapError(ErrSendQueueFull, "%s", peer.addr)
	}
}

// Broadcast 广播消息到所有节点
func (own *tcpTransport) Broadcast(msg Message, skips ...string) error {
	own.peerLock.RLock()
	peers := make([]*tcpPeer, 0, len(own.peers))
	for _, peer := range own.peers {
		peers = append(peers, peer)
	}
	own.peerLock.RUnlock()

	var failed error
	for _, peer := range peers {
		var isSkipped bool
		for _, skip := range skips {
			if peer.addr == skip {
				isSkipped = true
			}
		}

		if isSkipped {
			continue
		}

		select {
		case peer.sendQueue <- msg:
		case <-peer.done:
			own.config.logDebug("Peer %s is closed, skip broadcasting", peer.addr)
		default:
			own.broadcastDropped.Add(1)
			own.config.logDebug("Send queue is full to %s", peer.addr)
			if failed == nil {
				failed = wrapError(ErrSendQueueFull, "%s", peer.addr)
			}
		}
	}

	return failed
}

// GetStats 获取传输层运行时统计
func (own *tcpTransport) GetStats() TransportStats {
	return TransportStats{
		SendQueueFull:    own.sendQueueFull.Load(),
		BroadcastDropped: own.broadcastDropped.Load(),
		EncodeFailed:     own.encodeFailed.Load(),
		DecodeFailed:     own.decodeFailed.Load(),
	}
}

// AddPeer 添加节点
func (own *tcpTransport) AddPeer(addr string) {
	// 检查节点是否存在
	own.peerLock.RLock()
	_, exists := own.peers[addr]
	own.peerLock.RUnlock()
	if exists {
		own.config.logDebug("Peer %s already exists", addr)
		return
	}

	// 发起主动连接
	conn, err := net.DialTimeout("tcp", addr, own.config.ConnDialTimeout)
	if err != nil {
		logError("Failed to connect to peer %s: %v", addr, err)
		return
	}

	// 判断否是为种子节点
	isSeed := false
	for _, seed := range own.config.seedAddresses {
		if seed == addr {
			isSeed = true
			break
		}
	}

	// 添加节点
	own.addPeer(own.newPeer(addr, conn, isSeed))
}

// addPeer 注册 Peer
func (own *tcpTransport) addPeer(peer *tcpPeer) {
	// transport 已关闭时不再接纳新连接，避免遗留无人管理的连接
	if atomic.LoadInt32(&own.closed) == 1 {
		_ = peer.conn.Close()
		return
	}

	own.peerLock.Lock()
	defer own.peerLock.Unlock()

	// 避免重复注册
	if existing, exists := own.peers[peer.addr]; exists {
		// 旧连接仍存活时保留旧连接，避免双方同时拨号造成连接抖动；
		// 旧连接已关闭或长时间无收发时直接替换，否则新连接会被丢弃，
		// 要等到故障检测超时后才能恢复通信。
		if !own.isPeerStale(existing) {
			own.config.logDebug("Register the peer repeatedly: %s", peer.addr)
			_ = peer.conn.Close()
			return
		}

		own.config.logDebug("Replace the stale peer: %s", peer.addr)
		own.removePeerWithLock(peer.addr)
	}

	// 初始化分数（此时连接尚未发布到 peers 表，写入先于并发读者）
	peer.createdAt = time.Now()
	atomic.StoreInt64(&peer.score, _PEER_SCORE_INIT_BASE)
	if peer.isSeed {
		atomic.AddInt64(&peer.score, _PEER_SCORE_BONUS)
	}

	// 添加 Peer
	own.evictPeersIfNeededWithLock()
	own.peers[peer.addr] = peer

	// 启动发送和接收循环
	go own.loopSend(peer)
	go own.loopReceive(peer)
}

// isPeerStale 判断连接是否已失效。
//
// 关闭标记、连接年龄与最后一次收发时间三者结合：保护期内的新连接
// 不会被误判；超过故障检测阈值且长时间无收发的连接视为失效。
func (own *tcpTransport) isPeerStale(peer *tcpPeer) bool {
	if atomic.LoadInt32(&peer.closed) == 1 {
		return true
	}

	now := time.Now()
	if now.Sub(peer.createdAt) <= own.config.GossipFailureTimeout {
		return false
	}

	lastActive := peer.lastActive.Load()
	if lastActive == 0 {
		return false
	}

	return now.Sub(time.Unix(0, lastActive)) > own.config.GossipFailureTimeout
}

// evictPeersIfNeededWithLock 在连接数超过软限制时调用（需要持有连接锁）
func (own *tcpTransport) evictPeersIfNeededWithLock() {
	// 1.检查连接数量是否超过软限制
	softLimit := int(float64(own.maxPeers) * _PEER_SOFT_LIMIT_FACTOR)
	if len(own.peers) <= softLimit {
		return
	}

	// 2.计算需要剔除的连接数量
	toEvict := len(own.peers) - own.maxPeers
	if toEvict <= 0 {
		return
	}

	// 3.跳过保护期内的连接
	now := time.Now()
	candidates := make([]*tcpPeer, 0, len(own.peers))
	for _, peer := range own.peers {
		if now.Sub(peer.createdAt) < _PEER_PROTECTION_PERIOD {
			continue
		}

		candidates = append(candidates, peer)
	}

	// 4.保护期内连接太多，暂不淘汰
	if len(candidates) < toEvict {
		return
	}

	// 随机打乱，避免总是淘汰同一批
	rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
	// 按分数升序排序（优先淘汰低分数的连接）
	//   - score 在其他路径用 atomic 读写，此处必须同样原子读取，否则构成数据竞态
	sort.Slice(candidates, func(i, j int) bool {
		return atomic.LoadInt64(&candidates[i].score) < atomic.LoadInt64(&candidates[j].score)
	})
	// 淘汰分数最低的 toEvict 个
	for i := 0; i < toEvict && i < len(candidates); i++ {
		own.removePeerWithLock(candidates[i].addr)
	}
}

// RemovePeer 移除节点
func (own *tcpTransport) RemovePeer(addr string) {
	own.peerLock.Lock()
	own.removePeerWithLock(addr)
	own.peerLock.Unlock()
}

// removePeerWithLock 移除连接（需要持有连接锁）
func (own *tcpTransport) removePeerWithLock(addr string) {
	if peer, ok := own.peers[addr]; ok {
		// 使用 CAS 防止重复关闭
		if !atomic.CompareAndSwapInt32(&peer.closed, 0, 1) {
			return // 已被其他 goroutine 关闭
		}
		delete(own.peers, addr)
		// 关闭 done 而不是 sendQueue：发送方通过 select 感知关闭，不会 panic
		close(peer.done)
		_ = peer.conn.Close()
	}
}

// AddHandler 注册消息处理器
func (own *tcpTransport) AddHandler(handler MessageHandler) {
	if handler != nil {
		own.handlerLock.Lock()
		own.handlers = append(own.handlers, handler)
		own.handlerLock.Unlock()
	}
}

// Close 关闭通信
func (own *tcpTransport) Close() {
	if !atomic.CompareAndSwapInt32(&own.closed, 0, 1) {
		return // 已关闭
	}
	close(own.stopChan)
	if own.listener != nil {
		_ = own.listener.Close()
	}

	own.peerLock.Lock()
	defer own.peerLock.Unlock()

	// 关闭所有连接
	for _, peer := range own.peers {
		if atomic.CompareAndSwapInt32(&peer.closed, 0, 1) {
			close(peer.done)
		}
		_ = peer.conn.Close()
	}
}
