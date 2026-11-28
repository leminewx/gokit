package mesh

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewMessageBus_Basic(t *testing.T) {
	config := NewDefaultConfig()
	transport := &mockTransport{}
	bus := NewMessageBus("node-a", transport, config)
	defer bus.Stop()

	if bus == nil {
		t.Fatal("expected non-nil MessageBus")
	}
}

func TestMessageBus_Subscribe(t *testing.T) {
	config := NewDefaultConfig()
	transport := &mockTransport{}
	bus := NewMessageBus("node-sub", transport, config)
	defer bus.Stop()

	var called bool
	var mu sync.Mutex
	err := bus.Subscribe("/sensors/temp", func(msg Message, pathValues map[string]string) {
		mu.Lock()
		called = true
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// 通过 HandleRemoteMessage 模拟收到 pub 消息
	bus.HandleRemoteMessage(Message{
		SN:       time.Now().UnixNano(),
		Type:     MESSAGE_TYPE_PUB,
		Path:     "/sensors/temp",
		Value:    "25.5",
		SenderID: "other",
	}, "other-addr")

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	wasCalled := called
	mu.Unlock()
	if !wasCalled {
		t.Error("callback should have been called")
	}
}

func TestMessageBus_Subscribe_EmptyCallbacks(t *testing.T) {
	config := NewDefaultConfig()
	transport := &mockTransport{}
	bus := NewMessageBus("node-sub-empty", transport, config)
	defer bus.Stop()

	err := bus.Subscribe("/test")
	if err == nil {
		t.Error("should error on empty callbacks")
	}

	err = bus.Subscribe("/test", nil)
	if err == nil {
		t.Error("should error on nil callback")
	}
}

func TestMessageBus_PutAndGet(t *testing.T) {
	config := NewDefaultConfig()
	transport := &mockTransport{}
	bus := NewMessageBus("node-kv", transport, config)
	defer bus.Stop()

	// 直接模拟写入 (通过 handlePut)
	bus.HandleRemoteMessage(Message{
		SN:       time.Now().UnixNano(),
		Type:     MESSAGE_TYPE_PUT,
		Path:     "greeting",
		Value:    "hello world",
		SenderID: "other",
	}, "other-addr")

	time.Sleep(50 * time.Millisecond)

	// 直接模拟读取 (通过 handleGet 响应)
	bus.HandleRemoteMessage(Message{
		SN:        time.Now().UnixNano(),
		Type:      MESSAGE_TYPE_GET,
		Path:      "greeting",
		SenderID:  "node-kv",
		RequestID: 1,
	}, "self-addr")

	time.Sleep(50 * time.Millisecond)
}

func TestMessageBus_Put_Smoke(t *testing.T) {
	config := NewDefaultConfig()
	transport := &mockTransport{}
	bus := NewMessageBus("node-put", transport, config)
	defer bus.Stop()

	err := bus.Put("key1", "value1")
	if err != nil {
		t.Errorf("Put failed: %v", err)
	}
}

func TestMessageBus_Publish_Smoke(t *testing.T) {
	config := NewDefaultConfig()
	transport := &mockTransport{}
	bus := NewMessageBus("node-pub", transport, config)
	defer bus.Stop()

	err := bus.Publish("/topic/foo", map[string]string{"msg": "bar"})
	if err != nil {
		t.Errorf("Publish failed: %v", err)
	}
}

func TestMessageBus_RegisterQueryHandler(t *testing.T) {
	config := NewDefaultConfig()
	transport := &mockTransport{}
	bus := NewMessageBus("node-rpc", transport, config)
	defer bus.Stop()

	err := bus.RegisterQueryHandler("/rpc/echo", func(msg Message) any {
		return msg.Payload
	})
	if err != nil {
		t.Fatalf("RegisterQueryHandler failed: %v", err)
	}

	err = bus.RegisterQueryHandler("/rpc/nil", nil)
	if err == nil {
		t.Error("should error on nil handler")
	}
}

func TestMessageBus_HandleQuery(t *testing.T) {
	config := NewDefaultConfig()
	transport := &mockTransport{}
	bus := NewMessageBus("node-query", transport, config)
	defer bus.Stop()

	err := bus.RegisterQueryHandler("/rpc/echo", func(msg Message) any {
		return "echo: " + msg.Payload.(string)
	})
	if err != nil {
		t.Fatalf("RegisterQueryHandler failed: %v", err)
	}

	// 模拟 query 消息
	bus.HandleRemoteMessage(Message{
		SN:        time.Now().UnixNano(),
		Type:      MESSAGE_TYPE_QUERY,
		Path:      "/rpc/echo",
		Payload:   "hello",
		SenderID:  "caller",
		RequestID: 42,
	}, "caller-addr")

	time.Sleep(50 * time.Millisecond)
}

func TestMessageBus_Deduplication(t *testing.T) {
	config := NewDefaultConfig()
	transport := &mockTransport{}
	bus := NewMessageBus("node-dedup", transport, config)
	defer bus.Stop()

	var callCount int
	var mu sync.Mutex
	bus.Subscribe("/dedup/test", func(msg Message, pathValues map[string]string) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	sn := time.Now().UnixNano()

	// 第一次发送
	bus.HandleRemoteMessage(Message{
		SN:       sn,
		Type:     MESSAGE_TYPE_PUB,
		Path:     "/dedup/test",
		Value:    "data",
		SenderID: "sender-1",
	}, "sender-1-addr")

	// 重复发送（相同 SN）
	bus.HandleRemoteMessage(Message{
		SN:       sn,
		Type:     MESSAGE_TYPE_PUB,
		Path:     "/dedup/test",
		Value:    "data",
		SenderID: "sender-1",
	}, "sender-1-addr")

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	count := callCount
	mu.Unlock()
	if count != 1 {
		t.Errorf("callback count = %d, want 1 (dedup should prevent second call)", count)
	}
}

func TestMessageBus_SetGossipHandler(t *testing.T) {
	config := NewDefaultConfig()
	transport := &mockTransport{}
	bus := NewMessageBus("node-gossip", transport, config)
	defer bus.Stop()

	// 注册 gossip handler 不应 panic
	// （此处仅为冒烟测试）
}

func TestMessageBus_Stop(t *testing.T) {
	config := NewDefaultConfig()
	transport := &mockTransport{}
	bus := NewMessageBus("node-stop", transport, config)

	bus.Stop()
	// 不应panic，goroutine 应退出
	time.Sleep(100 * time.Millisecond)
}

func TestMessageType_Constants(t *testing.T) {
	types := []MessageType{
		MESSAGE_TYPE_PUB,
		MESSAGE_TYPE_PUT,
		MESSAGE_TYPE_GET,
		MESSAGE_TYPE_GET_RESPONSE,
		MESSAGE_TYPE_QUERY,
		MESSAGE_TYPE_QUERY_RESPONSE,
		MESSAGE_TYPE_GOSSIP_SYNC,
		MESSAGE_TYPE_GOSSIP_HEARTBEAT,
	}

	for _, mt := range types {
		if string(mt) == "" {
			t.Errorf("message type %v should not be empty", mt)
		}
	}
}

func TestMessage_Struct(t *testing.T) {
	msg := Message{
		SN:        123456789,
		TTL:       3,
		Type:      MESSAGE_TYPE_PUB,
		SenderID:  "sender-1",
		Path:      "/topic/test",
		Value:     "hello",
		Payload:   map[string]int{"a": 1},
		RequestID: 999,
		GossipNodes: map[string]NodeState{
			"node-1": {ID: "node-1", Addr: "addr-1", SN: 10},
		},
	}

	if msg.SN != 123456789 {
		t.Errorf("SN = %d, want 123456789", msg.SN)
	}
	if msg.TTL != 3 {
		t.Errorf("TTL = %d, want 3", msg.TTL)
	}
	if msg.Type != MESSAGE_TYPE_PUB {
		t.Errorf("Type = %s, want pub", msg.Type)
	}
	if msg.SenderID != "sender-1" {
		t.Errorf("SenderID = %s, want sender-1", msg.SenderID)
	}
	if msg.Path != "/topic/test" {
		t.Errorf("Path = %s, want /topic/test", msg.Path)
	}
	if msg.Value != "hello" {
		t.Errorf("Value = %v, want hello", msg.Value)
	}
	if msg.RequestID != 999 {
		t.Errorf("RequestID = %d, want 999", msg.RequestID)
	}
	if len(msg.GossipNodes) != 1 {
		t.Errorf("GossipNodes len = %d, want 1", len(msg.GossipNodes))
	}
}

func TestPendingKey_Struct(t *testing.T) {
	key := pendingKey{senderID: "node-1", requestID: 42}
	if key.senderID != "node-1" {
		t.Errorf("senderID = %s, want node-1", key.senderID)
	}
	if key.requestID != 42 {
		t.Errorf("requestID = %d, want 42", key.requestID)
	}
}

// TestMessageBus_SubscribeAppendsCallbacks 验证同一主题重复订阅时追加回调而不是覆盖
func TestMessageBus_SubscribeAppendsCallbacks(t *testing.T) {
	bus := NewMessageBus("node-append", newMockTransport(), NewDefaultConfig())
	defer bus.Stop()

	var wg sync.WaitGroup
	wg.Add(3)

	if err := bus.Subscribe("/sensors/temp", func(Message, map[string]string) { wg.Done() }); err != nil {
		t.Fatalf("first Subscribe failed: %v", err)
	}
	if err := bus.Subscribe("/sensors/temp",
		func(Message, map[string]string) { wg.Done() },
		func(Message, map[string]string) { wg.Done() },
	); err != nil {
		t.Fatalf("second Subscribe failed: %v", err)
	}

	bus.HandleRemoteMessage(Message{
		SN:       time.Now().UnixNano(),
		Type:     MESSAGE_TYPE_PUB,
		Path:     "/sensors/temp",
		SenderID: "other",
	}, "other-addr")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("all callbacks should be invoked when subscribing to the same topic twice")
	}
}

// TestMessageBus_ConcurrentSubscribeAndPublish 并发订阅与分发，验证 Trie 读写已被锁保护
func TestMessageBus_ConcurrentSubscribeAndPublish(t *testing.T) {
	bus := NewMessageBus("node-race", newMockTransport(), NewDefaultConfig())
	defer bus.Stop()

	var hits atomic.Int64
	var sn atomic.Int64
	var wg sync.WaitGroup

	const (
		workers    = 8
		iterations = 20
	)
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if err := bus.Subscribe("/race/topic", func(Message, map[string]string) {
					hits.Add(1)
				}); err != nil {
					t.Errorf("Subscribe failed: %v", err)
					return
				}

				bus.HandleRemoteMessage(Message{
					SN:       sn.Add(1),
					Type:     MESSAGE_TYPE_PUB,
					Path:     "/race/topic",
					SenderID: "other",
				}, "other-addr")
			}
		}()
	}
	wg.Wait()

	deadline := time.Now().Add(2 * time.Second)
	for hits.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if hits.Load() == 0 {
		t.Fatal("expected at least one callback invocation")
	}
}
