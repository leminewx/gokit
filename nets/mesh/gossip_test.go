package mesh

import (
	"sync"
	"testing"
	"time"
)

// mockTransport 模拟 Transport，仅记录调用的方法
type mockTransport struct {
	mu              sync.Mutex
	addPeerCalls    []string
	removePeerCalls []string
	sendToCalls     []sendToCall
	broadcastCalls  []Message
}

type sendToCall struct {
	addr string
	msg  Message
}

func newMockTransport() *mockTransport {
	return &mockTransport{}
}

func (m *mockTransport) Start() error                      { return nil }
func (m *mockTransport) Close()                            {}
func (m *mockTransport) AddHandler(handler MessageHandler) {}
func (m *mockTransport) RegisterPeer(addr string)          { m.AddPeer(addr) }

func (m *mockTransport) AddPeer(addr string) {
	m.mu.Lock()
	m.addPeerCalls = append(m.addPeerCalls, addr)
	m.mu.Unlock()
}

func (m *mockTransport) RemovePeer(addr string) {
	m.mu.Lock()
	m.removePeerCalls = append(m.removePeerCalls, addr)
	m.mu.Unlock()
}

func (m *mockTransport) SendTo(addr string, msg Message) error {
	m.mu.Lock()
	m.sendToCalls = append(m.sendToCalls, sendToCall{addr: addr, msg: msg})
	m.mu.Unlock()
	return nil
}

func (m *mockTransport) Broadcast(msg Message, skips ...string) error {
	m.mu.Lock()
	m.broadcastCalls = append(m.broadcastCalls, msg)
	m.mu.Unlock()
	return nil
}

// mockMessageBus 模拟 MessageBus
type mockMessageBus struct{}

func (m *mockMessageBus) Subscribe(topic string, callbacks ...func(msg Message, pathValues map[string]string)) error {
	return nil
}
func (m *mockMessageBus) Publish(topic string, value any) error { return nil }
func (m *mockMessageBus) Put(key string, value any) error       { return nil }
func (m *mockMessageBus) Get(key string) (any, error)           { return nil, nil }
func (m *mockMessageBus) RegisterQueryHandler(path string, handler func(Message) any) error {
	return nil
}
func (m *mockMessageBus) Query(path string, payload any) (any, error)      { return nil, nil }
func (m *mockMessageBus) HandleRemoteMessage(msg Message, fromAddr string) {}
func (m *mockMessageBus) SetGossipHandler(handler GossipMessageHandler)    {}
func (m *mockMessageBus) Stop()                                            {}

// mockGossipHandler 记录回调
type mockGossipHandler struct {
	mu     sync.Mutex
	joins  []joinEvent
	leaves []string
}

type joinEvent struct {
	nodeID string
	addr   string
}

func (h *mockGossipHandler) CallOnJoin(nodeID, addr string) {
	h.mu.Lock()
	h.joins = append(h.joins, joinEvent{nodeID: nodeID, addr: addr})
	h.mu.Unlock()
}

func (h *mockGossipHandler) CallOnLeave(nodeID string) {
	h.mu.Lock()
	h.leaves = append(h.leaves, nodeID)
	h.mu.Unlock()
}

func TestNewGossipService(t *testing.T) {
	transport := newMockTransport()
	bus := &mockMessageBus{}
	config := NewDefaultConfig()

	gossip := NewGossipService("self-id", "127.0.0.1:8080", transport, bus, nil, config)
	if gossip == nil {
		t.Fatal("expected non-nil Gossip")
	}

	nodes := gossip.GetNodes()
	if len(nodes) < 1 {
		t.Error("should have at least self node")
	}

	found := false
	for _, n := range nodes {
		if n.ID == "self-id" && n.Addr == "127.0.0.1:8080" {
			found = true
			break
		}
	}
	if !found {
		t.Error("self node should be registered")
	}
}

func TestGossip_RegisterHandler(t *testing.T) {
	transport := newMockTransport()
	bus := &mockMessageBus{}
	config := NewDefaultConfig()

	gossip := NewGossipService("id1", "addr1", transport, bus, nil, config)
	handler := &mockGossipHandler{}
	gossip.RegisterHandler(handler)

	// 不会panic就是成功
}

func TestGossip_HandleGossipMessage_NewNode(t *testing.T) {
	transport := newMockTransport()
	bus := &mockMessageBus{}
	config := NewDefaultConfig()

	gossip := NewGossipService("self", "self-addr", transport, bus, nil, config)
	handler := &mockGossipHandler{}
	gossip.RegisterHandler(handler)

	// 模拟收到新的 Gossip 心跳消息
	msg := Message{
		SN:       time.Now().UnixNano(),
		Type:     MESSAGE_TYPE_GOSSIP_HEARTBEAT,
		SenderID: "peer-1",
		GossipNodes: map[string]NodeState{
			"peer-1": {ID: "peer-1", Addr: "peer-1-addr", SN: 1, Timestamp: time.Now()},
		},
	}
	gossip.HandleGossipMessage(msg, "peer-1-addr")

	// 等待异步 AddPeer
	time.Sleep(50 * time.Millisecond)

	nodes := gossip.GetNodes()
	foundPeer := false
	for _, n := range nodes {
		if n.ID == "peer-1" {
			foundPeer = true
			break
		}
	}
	if !foundPeer {
		t.Error("peer-1 should be in node list after handling gossip message")
	}
}

func TestGossip_HandleGossipMessage_ExistingNode(t *testing.T) {
	transport := newMockTransport()
	bus := &mockMessageBus{}
	config := NewDefaultConfig()

	gossip := NewGossipService("self", "self-addr", transport, bus, nil, config)

	// 先添加一个已知节点
	msg1 := Message{
		SN:       1,
		Type:     MESSAGE_TYPE_GOSSIP_HEARTBEAT,
		SenderID: "peer-2",
		GossipNodes: map[string]NodeState{
			"peer-2": {ID: "peer-2", Addr: "peer-2-addr", SN: 1, Timestamp: time.Now()},
		},
	}
	gossip.HandleGossipMessage(msg1, "peer-2-addr")

	// 再次收到同节点消息，SN 更大
	msg2 := Message{
		SN:       2,
		Type:     MESSAGE_TYPE_GOSSIP_HEARTBEAT,
		SenderID: "peer-2",
		GossipNodes: map[string]NodeState{
			"peer-2": {ID: "peer-2", Addr: "peer-2-addr", SN: 5, Timestamp: time.Now()},
		},
	}
	gossip.HandleGossipMessage(msg2, "peer-2-addr")

	// 验证 SN 已更新
	nodes := gossip.GetNodes()
	for _, n := range nodes {
		if n.ID == "peer-2" && n.SN != 5 {
			t.Errorf("peer-2 SN = %d, want 5", n.SN)
		}
	}
}

func TestGossip_HandleGossipMessage_IgnoresNonGossip(t *testing.T) {
	transport := newMockTransport()
	bus := &mockMessageBus{}
	config := NewDefaultConfig()

	gossip := NewGossipService("self", "self-addr", transport, bus, nil, config)

	initialCount := len(gossip.GetNodes())

	// 发送非 Gossip 消息
	msg := Message{
		SN:       time.Now().UnixNano(),
		Type:     MESSAGE_TYPE_PUB,
		SenderID: "peer-3",
	}
	gossip.HandleGossipMessage(msg, "peer-3-addr")

	if len(gossip.GetNodes()) != initialCount {
		t.Error("non-gossip messages should not add nodes")
	}
}

func TestGossip_Stop(t *testing.T) {
	transport := newMockTransport()
	bus := &mockMessageBus{}
	config := NewDefaultConfig()

	gossip := NewGossipService("self", "self-addr", transport, bus, nil, config)
	gossip.Stop()
	// 不应panic，不会无限阻塞
}

func TestGossip_GetNodes_ReturnsCopy(t *testing.T) {
	transport := newMockTransport()
	bus := &mockMessageBus{}
	config := NewDefaultConfig()

	gossip := NewGossipService("self", "self-addr", transport, bus, nil, config)

	nodes1 := gossip.GetNodes()
	nodes1[0] = NodeState{ID: "modified"}
	nodes2 := gossip.GetNodes()

	if nodes2[0].ID == "modified" {
		t.Error("GetNodes should return a copy, not a reference")
	}
}

func TestNodeState_Fields(t *testing.T) {
	now := time.Now()
	ns := NodeState{
		SN:        12345,
		ID:        "test-node",
		Addr:      "192.168.1.1:9000",
		Timestamp: now,
	}

	if ns.SN != 12345 {
		t.Errorf("SN = %d, want 12345", ns.SN)
	}
	if ns.ID != "test-node" {
		t.Errorf("ID = %s, want test-node", ns.ID)
	}
	if ns.Addr != "192.168.1.1:9000" {
		t.Errorf("Addr = %s, want 192.168.1.1:9000", ns.Addr)
	}
	if !ns.Timestamp.Equal(now) {
		t.Error("Timestamp should match")
	}
}
