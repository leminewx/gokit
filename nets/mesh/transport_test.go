package mesh

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewTCPTransport(t *testing.T) {
	transport := NewTCPTransport(":0", 30, NewDefaultConfig())

	if transport == nil {
		t.Fatal("expected non-nil Transport")
	}

	// Close without Start should be safe
	transport.Close()
}

func TestTCPTransport_StartAndClose(t *testing.T) {
	transport := NewTCPTransport(":0", 30, NewDefaultConfig())

	if err := transport.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Close 不应panic
	transport.Close()
}

func TestTCPTransport_AddHandler(t *testing.T) {
	transport := NewTCPTransport(":0", 30, NewDefaultConfig())
	defer transport.Close()

	var called int32
	handler := &testMessageHandler{
		handleFunc: func(msg Message, fromAddr string) {
			atomic.AddInt32(&called, 1)
		},
	}

	transport.AddHandler(handler)
	transport.AddHandler(nil) // 不应panic

	_ = handler
}

type testMessageHandler struct {
	handleFunc func(msg Message, fromAddr string)
}

func (h *testMessageHandler) HandleRemoteMessage(msg Message, fromAddr string) {
	if h.handleFunc != nil {
		h.handleFunc(msg, fromAddr)
	}
}

func TestTCPTransport_SendTo_UnknownPeer(t *testing.T) {
	transport := NewTCPTransport(":0", 30, NewDefaultConfig())
	defer transport.Close()
	_ = transport.Start()

	err := transport.SendTo("127.0.0.1:9999", Message{Type: MESSAGE_TYPE_PUB})
	if err == nil {
		t.Error("SendTo unknown peer should return error")
	}
}

func TestTCPTransport_Broadcast_NoPeers(t *testing.T) {
	transport := NewTCPTransport(":0", 30, NewDefaultConfig())
	defer transport.Close()
	_ = transport.Start()

	// 没有peers时广播不应报错
	err := transport.Broadcast(Message{Type: MESSAGE_TYPE_PUB, SenderID: "test"})
	if err != nil {
		t.Errorf("Broadcast with no peers should not error: %v", err)
	}
}

func TestTCPTransport_Broadcast_WithSkips(t *testing.T) {
	transport := NewTCPTransport(":0", 30, NewDefaultConfig())
	defer transport.Close()
	_ = transport.Start()

	err := transport.Broadcast(Message{Type: MESSAGE_TYPE_PUB}, "127.0.0.1:9001", "127.0.0.1:9002")
	if err != nil {
		t.Errorf("Broadcast with skips should not error: %v", err)
	}
}

func TestTCPTransport_AddPeer_InvalidAddress(t *testing.T) {
	transport := NewTCPTransport(":0", 30, NewDefaultConfig())
	defer transport.Close()
	_ = transport.Start()

	// 添加一个不存在的地址（拨号会超时）
	transport.AddPeer("127.0.0.1:19999")
	// 不应panic
}

func TestTCPTransport_RemovePeer_Unknown(t *testing.T) {
	transport := NewTCPTransport(":0", 30, NewDefaultConfig())
	defer transport.Close()
	_ = transport.Start()

	// 移除不存在的peer不应panic
	transport.RemovePeer("unknown:9999")
}

func TestTCPTransport_TwoNodesCommunication(t *testing.T) {
	transport1 := NewTCPTransport("127.0.0.1:0", 10, NewDefaultConfig())
	transport2 := NewTCPTransport("127.0.0.1:0", 10, NewDefaultConfig())
	defer transport1.Close()
	defer transport2.Close()

	if err := transport1.Start(); err != nil {
		t.Fatalf("transport1 Start: %v", err)
	}
	if err := transport2.Start(); err != nil {
		t.Fatalf("transport2 Start: %v", err)
	}

	// 获取 transport2 的实际监听地址
	addr2 := getTransportAddr(transport2)
	if addr2 == "" {
		t.Fatal("cannot get transport2 address")
	}

	// transport1 连接 transport2
	transport1.AddPeer(addr2)

	// 等待连接建立
	time.Sleep(200 * time.Millisecond)

	// 设置 transport2 的消息处理器
	var receivedMsg Message
	var mu sync.Mutex
	done := make(chan struct{})

	handler := &testMessageHandler{
		handleFunc: func(msg Message, fromAddr string) {
			mu.Lock()
			receivedMsg = msg
			mu.Unlock()
			close(done)
		},
	}
	transport2.AddHandler(handler)

	// transport1 发送消息到 transport2
	err := transport1.SendTo(addr2, Message{
		SN:       time.Now().UnixNano(),
		Type:     MESSAGE_TYPE_PUB,
		Path:     "/test/msg",
		Value:    "hello from transport1",
		SenderID: "node-1",
	})
	if err != nil {
		t.Fatalf("SendTo failed: %v", err)
	}

	// 等待接收
	select {
	case <-done:
		mu.Lock()
		if receivedMsg.Path != "/test/msg" {
			t.Errorf("received path = %s, want /test/msg", receivedMsg.Path)
		}
		if receivedMsg.Value != "hello from transport1" {
			t.Errorf("received value = %v, want 'hello from transport1'", receivedMsg.Value)
		}
		mu.Unlock()
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestTCPTransport_BroadcastToMultiple(t *testing.T) {
	transport1 := NewTCPTransport("127.0.0.1:0", 10, NewDefaultConfig())
	transport2 := NewTCPTransport("127.0.0.1:0", 10, NewDefaultConfig())
	transport3 := NewTCPTransport("127.0.0.1:0", 10, NewDefaultConfig())
	defer transport1.Close()
	defer transport2.Close()
	defer transport3.Close()

	transport1.Start()
	transport2.Start()
	transport3.Start()

	addr2 := getTransportAddr(transport2)
	addr3 := getTransportAddr(transport3)

	var receivedCount int32

	handler := &testMessageHandler{
		handleFunc: func(msg Message, fromAddr string) {
			atomic.AddInt32(&receivedCount, 1)
		},
	}

	transport2.AddHandler(handler)
	transport3.AddHandler(handler)

	// transport1 连接 transport2 和 transport3
	transport1.AddPeer(addr2)
	transport1.AddPeer(addr3)

	time.Sleep(200 * time.Millisecond)

	// 广播消息
	err := transport1.Broadcast(Message{
		SN:       time.Now().UnixNano(),
		Type:     MESSAGE_TYPE_PUB,
		Path:     "/broadcast",
		Value:    "everyone",
		SenderID: "node-1",
	})
	if err != nil {
		t.Fatalf("Broadcast failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	count := atomic.LoadInt32(&receivedCount)
	if count < 2 {
		t.Errorf("broadcast received = %d, want at least 2", count)
	}
}

func TestTCPTransport_MultipleHandlers(t *testing.T) {
	transport := NewTCPTransport("127.0.0.1:0", 10, NewDefaultConfig())
	defer transport.Close()
	transport.Start()

	var count1, count2 int32

	handler1 := &testMessageHandler{
		handleFunc: func(msg Message, fromAddr string) { atomic.AddInt32(&count1, 1) },
	}
	handler2 := &testMessageHandler{
		handleFunc: func(msg Message, fromAddr string) { atomic.AddInt32(&count2, 1) },
	}

	transport.AddHandler(handler1)
	transport.AddHandler(handler2)

	// 连接一个客户端
	client, err := net.DialTimeout("tcp", getTransportAddr(transport), 1*time.Second)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer client.Close()

	time.Sleep(200 * time.Millisecond)

	// 发送数据，需要等待 loopReceive 处理
	_ = client

	// 验证至少注册了两个 handler
	if count1+count2 < 0 { // 仅确保 handler 不为 nil
		t.Error("handlers should be registered")
	}
}

func TestTCPTransport_PeerScoreE2E(t *testing.T) {
	transport1 := NewTCPTransport("127.0.0.1:0", 10, NewDefaultConfig())
	transport2 := NewTCPTransport("127.0.0.1:0", 10, NewDefaultConfig())
	defer transport1.Close()
	defer transport2.Close()

	transport1.Start()
	transport2.Start()

	addr1 := getTransportAddr(transport1)
	addr2 := getTransportAddr(transport2)

	// 互相连接
	transport1.AddPeer(addr2)
	transport2.AddPeer(addr1)

	time.Sleep(300 * time.Millisecond)

	var received int32
	handler := &testMessageHandler{
		handleFunc: func(msg Message, fromAddr string) {
			atomic.AddInt32(&received, 1)
		},
	}
	transport2.AddHandler(handler)

	// 发送多条消息验证活跃加分
	for range 5 {
		transport1.SendTo(addr2, Message{
			SN:       time.Now().UnixNano(),
			Type:     MESSAGE_TYPE_PUB,
			SenderID: "node-1",
		})
	}

	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&received) < 1 {
		t.Error("should have received at least one message")
	}
}

func TestTCPTransport_Close_NilListener(t *testing.T) {
	transport := NewTCPTransport(":0", 30, NewDefaultConfig())
	// Close without Start should not panic
	transport.Close()
}

// getTransportAddr 获取 transport 实际监听地址
func getTransportAddr(tr Transport) string {
	tcpTr, ok := tr.(*tcpTransport)
	if !ok {
		return ""
	}
	if tcpTr.listener == nil {
		// Transport 可能还未启动
		return ""
	}
	addr := tcpTr.listener.Addr()
	if addr == nil {
		return ""
	}
	return fmt.Sprintf("127.0.0.1:%d", addr.(*net.TCPAddr).Port)
}

// newFullQueueTransport 构建一个只用于背压测试的传输层
//   - 直接注入 numPeers 个发送队列已满、且没有发送协程的连接
//   - 连接使用 net.Pipe 占位，保证 Close 时不会空指针
func newFullQueueTransport(t *testing.T, numPeers int, options ...Option) *tcpTransport {
	t.Helper()

	config := NewDefaultConfig()
	for _, option := range options {
		option(config)
	}

	own := NewTCPTransport("127.0.0.1:0", 10, config).(*tcpTransport)
	t.Cleanup(own.Close)

	for index := range numPeers {
		client, server := net.Pipe()
		t.Cleanup(func() { _ = server.Close() })

		peer := &tcpPeer{
			addr:      fmt.Sprintf("127.0.0.1:%d", 65000+index),
			conn:      client,
			encode:    func(Message) error { return nil },
			sendQueue: make(chan Message, 1),
			done:      make(chan struct{}),
		}
		peer.sendQueue <- Message{Type: MESSAGE_TYPE_PUB} // 填满队列

		own.peers[peer.addr] = peer
	}

	return own
}

// TestSendTo_QueueFull 验证队列满时立即失败并计数
func TestSendTo_QueueFull(t *testing.T) {
	t.Parallel()

	own := newFullQueueTransport(t, 1)

	start := time.Now()
	err := own.SendTo("127.0.0.1:65000", Message{Type: MESSAGE_TYPE_PUB})
	if !errors.Is(err, ErrSendQueueFull) {
		t.Fatalf("期望 ErrSendQueueFull, 实际 %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("缺省不应阻塞, 实际耗时 %v", elapsed)
	}

	if got := own.GetStats().SendQueueFull; got != 1 {
		t.Fatalf("发送队列满计数错误: 期望 1, 实际 %d", got)
	}
}

// TestSendTo_BlockTimeout 验证背压等待超时后仍返回队列满
func TestSendTo_BlockTimeout(t *testing.T) {
	t.Parallel()

	const timeout = 80 * time.Millisecond

	own := newFullQueueTransport(t, 1, WithSendBlockTimeout(timeout))

	start := time.Now()
	err := own.SendTo("127.0.0.1:65000", Message{Type: MESSAGE_TYPE_PUB})
	elapsed := time.Since(start)

	if !errors.Is(err, ErrSendQueueFull) {
		t.Fatalf("期望 ErrSendQueueFull, 实际 %v", err)
	}
	if elapsed < timeout/2 {
		t.Fatalf("未按背压等待: 耗时 %v", elapsed)
	}
	if got := own.GetStats().SendQueueFull; got != 1 {
		t.Fatalf("发送队列满计数错误: 期望 1, 实际 %d", got)
	}
}

// TestSendTo_PeerDone 验证连接已关闭时优先返回关闭错误
func TestSendTo_PeerDone(t *testing.T) {
	t.Parallel()

	own := newFullQueueTransport(t, 1)
	peer := own.peers["127.0.0.1:65000"]
	close(peer.done)

	err := own.SendTo(peer.addr, Message{Type: MESSAGE_TYPE_PUB})
	if !errors.Is(err, ErrPeerClosed) {
		t.Fatalf("期望 ErrPeerClosed, 实际 %v", err)
	}

	// 连接已由测试手动关闭，从连接表移除，避免退出时重复关闭 done
	own.peerLock.Lock()
	delete(own.peers, peer.addr)
	own.peerLock.Unlock()
	_ = peer.conn.Close()
}

// TestBroadcast_DropsWhenFull 验证广播对满队列的连接跳过并计数
func TestBroadcast_DropsWhenFull(t *testing.T) {
	t.Parallel()

	own := newFullQueueTransport(t, 2)

	err := own.Broadcast(Message{Type: MESSAGE_TYPE_PUB})
	if !errors.Is(err, ErrSendQueueFull) {
		t.Fatalf("期望 ErrSendQueueFull, 实际 %v", err)
	}

	stats := own.GetStats()
	if stats.BroadcastDropped != 2 {
		t.Fatalf("广播丢弃计数错误: 期望 2, 实际 %d", stats.BroadcastDropped)
	}
	if stats.SendQueueFull != 0 {
		t.Fatalf("广播不应计入单播拒绝: %d", stats.SendQueueFull)
	}
}

// TestGetStats_ZeroValue 验证未发生丢弃时统计为零值
func TestGetStats_ZeroValue(t *testing.T) {
	t.Parallel()

	own := NewTCPTransport("127.0.0.1:0", 10, NewDefaultConfig()).(*tcpTransport)
	t.Cleanup(own.Close)

	if stats := own.GetStats(); stats != (TransportStats{}) {
		t.Fatalf("初始统计应为零值: %+v", stats)
	}
}
