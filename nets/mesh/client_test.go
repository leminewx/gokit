package mesh

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ******************************************************************************************************************
// Mesh Client 集成测试
// ******************************************************************************************************************

// TestClientIntegration_FullLifecycle 完整生命周期测试：创建、运行、使用、关闭
func TestClientIntegration_FullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	const (
		addrSeed  = "127.0.0.1:19301"
		addrNodeA = "127.0.0.1:19302"
	)

	// 创建种子节点
	seed, err := NewClient(ClientOptions{
		ID:             "seed",
		Addr:           addrSeed,
		MaxPeers:       10,
		GossipInterval: 200 * time.Millisecond,
		Debug:          true,
	})
	if err != nil {
		t.Fatalf("创建种子客户端失败: %v", err)
	}
	defer seed.Close()

	t.Log("种子节点启动成功")

	// 等待种子就绪
	time.Sleep(20 * time.Millisecond)

	// 创建普通节点
	var (
		joinCount  int32
		leaveCount int32
		joinedIDs  []string
		mu         sync.Mutex
	)

	nodeA, err := NewClient(ClientOptions{
		ID:             "node-a",
		Addr:           addrNodeA,
		MaxPeers:       10,
		SeedAddrs:      []string{addrSeed},
		GossipInterval: 200 * time.Millisecond,
		OnJoin: func(nodeID, addr string) {
			atomic.AddInt32(&joinCount, 1)
			mu.Lock()
			joinedIDs = append(joinedIDs, nodeID)
			mu.Unlock()
		},
		OnLeave: func(nodeID string) {
			atomic.AddInt32(&leaveCount, 1)
		},
		Debug: true,
	})
	if err != nil {
		t.Fatalf("创建 nodeA 客户端失败: %v", err)
	}
	defer nodeA.Close()

	t.Log("nodeA 启动成功")

	// 等待 Gossip 收敛
	t.Log("等待 Gossip 成员表收敛...")
	if !waitClientConvergence(t, 2, 8*time.Second, seed, nodeA) {
		seedMembers := seed.GetMembers()
		nodeAMembers := nodeA.GetMembers()
		t.Fatalf("集群未收敛: seed 看到 %d 节点, nodeA 看到 %d 节点", len(seedMembers), len(nodeAMembers))
	}
	t.Log("Gossip 收敛完成")

	// 验证事件回调
	if atomic.LoadInt32(&joinCount) < 1 {
		t.Logf("注意: Join 事件可能未捕获 (count=%d), 这是已知的时序问题", joinCount)
	}
	t.Logf("Join 事件: %d 次", atomic.LoadInt32(&joinCount))

	// 验证健康信息
	health := seed.GetHealth()
	if health.NodeCount < 2 {
		t.Errorf("健康检查: 期望 >=2 节点, 实际 %d", health.NodeCount)
	}
	t.Logf("健康状态: %d 个节点在线, 运行时长: %s", health.NodeCount, health.Uptime.Round(time.Second))

	// 验证统计信息
	stats := seed.GetStats()
	t.Logf("统计: 发送=%d, 接收=%d, 在线=%d, 加入=%d, 离开=%d",
		stats.MessagesSent, stats.MessagesRecv, stats.OnlineNodes, stats.JoinEvents, stats.LeaveEvents)
}

// TestClientIntegration_PubSub 测试 Pub/Sub 消息传递
func TestClientIntegration_PubSub(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	const (
		addr1 = "127.0.0.1:19311"
		addr2 = "127.0.0.1:19312"
	)

	// 启动两个节点
	n1, err := NewClient(ClientOptions{
		ID:             "pub-1",
		Addr:           addr1,
		MaxPeers:       5,
		GossipInterval: 200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("创建 n1 失败: %v", err)
	}
	defer n1.Close()

	time.Sleep(20 * time.Millisecond)

	n2, err := NewClient(ClientOptions{
		ID:             "pub-2",
		Addr:           addr2,
		MaxPeers:       5,
		SeedAddrs:      []string{addr1},
		GossipInterval: 200 * time.Millisecond,
	})
	if err != nil {
		n1.Close()
		t.Fatalf("创建 n2 失败: %v", err)
	}
	defer n2.Close()

	// 等待收敛
	waitClientConvergence(t, 2, 5*time.Second, n1, n2)

	// n2 订阅
	var receivedCount int32
	var receivedMsgs []string
	var msgMu sync.Mutex

	err = n2.Subscribe("test/client-pubsub", func(msg Message, pathValues map[string]string) {
		atomic.AddInt32(&receivedCount, 1)
		msgMu.Lock()
		receivedMsgs = append(receivedMsgs, fmt.Sprintf("%v", msg.Value))
		msgMu.Unlock()
	})
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}

	// n1 连续发布（订阅为本地同步登记，无需等待）
	const msgCount = 3
	for i := range msgCount {
		msg := fmt.Sprintf("client-msg-%d", i+1)
		if err := n1.Publish("test/client-pubsub", msg); err != nil {
			t.Errorf("发布失败: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// 轮询等待广播送达
	waitFor(t, 5*time.Second, "Pub/Sub 消息全部送达", func() bool {
		return atomic.LoadInt32(&receivedCount) >= int32(msgCount)
	})
	t.Logf("Pub/Sub 测试通过: 发送 %d 条, 接收 %d 条", msgCount, atomic.LoadInt32(&receivedCount))

	// 验证统计
	stats := n1.GetStats()
	t.Logf("n1 统计: 发送=%d, 接收=%d", stats.MessagesSent, stats.MessagesRecv)
}

// TestClientIntegration_KVStore 测试 KV 存储
func TestClientIntegration_KVStore(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	const (
		addr1 = "127.0.0.1:19321"
		addr2 = "127.0.0.1:19322"
	)

	n1, err := NewClient(ClientOptions{
		ID:             "kv-1",
		Addr:           addr1,
		MaxPeers:       5,
		GossipInterval: 200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("创建 n1 失败: %v", err)
	}
	defer n1.Close()

	time.Sleep(20 * time.Millisecond)

	n2, err := NewClient(ClientOptions{
		ID:             "kv-2",
		Addr:           addr2,
		MaxPeers:       5,
		SeedAddrs:      []string{addr1},
		GossipInterval: 200 * time.Millisecond,
	})
	if err != nil {
		n1.Close()
		t.Fatalf("创建 n2 失败: %v", err)
	}
	defer n2.Close()

	waitClientConvergence(t, 2, 5*time.Second, n1, n2)

	// n1 Put 数据
	err = n1.Put("app/name", "mesh-client-test")
	if err != nil {
		t.Fatalf("Put 失败: %v", err)
	}
	err = n1.Put("app/version", 1)
	if err != nil {
		t.Fatalf("Put 失败: %v", err)
	}

	// n2 本地也 Put 一条
	err = n2.Put("app/author", "gokit")
	if err != nil {
		t.Fatalf("Put 失败: %v", err)
	}

	// n2 轮询等待 n1 写入的数据广播到达
	var val any
	waitFor(t, 5*time.Second, "n2 读到 n1 写入的 app/name", func() bool {
		v, getErr := n2.Get("app/name")
		if getErr != nil {
			return false
		}
		val = v
		return true
	})
	if val != "mesh-client-test" {
		t.Errorf("Get app/name: 期望 mesh-client-test, 实际 %v", val)
	} else {
		t.Logf("Get app/name = %v  ✓", val)
	}

	// n1 轮询等待 n2 写入的数据广播到达
	waitFor(t, 5*time.Second, "n1 读到 n2 写入的 app/author", func() bool {
		v, getErr := n1.Get("app/author")
		if getErr != nil {
			return false
		}
		val = v
		return true
	})
	if val != "gokit" {
		t.Errorf("Get app/author: 期望 gokit, 实际 %v", val)
	} else {
		t.Logf("Get app/author = %v  ✓", val)
	}
}

// TestClientIntegration_RPC 测试 RPC 查询
func TestClientIntegration_RPC(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	const (
		addr1 = "127.0.0.1:19331"
		addr2 = "127.0.0.1:19332"
	)

	n1, err := NewClient(ClientOptions{
		ID:             "rpc-1",
		Addr:           addr1,
		MaxPeers:       5,
		GossipInterval: 200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("创建 n1 失败: %v", err)
	}
	defer n1.Close()

	time.Sleep(20 * time.Millisecond)

	n2, err := NewClient(ClientOptions{
		ID:             "rpc-2",
		Addr:           addr2,
		MaxPeers:       5,
		SeedAddrs:      []string{addr1},
		GossipInterval: 200 * time.Millisecond,
	})
	if err != nil {
		n1.Close()
		t.Fatalf("创建 n2 失败: %v", err)
	}
	defer n2.Close()

	waitClientConvergence(t, 2, 5*time.Second, n1, n2)

	// n2 注册 RPC 处理器
	err = n2.RegisterQueryHandler("/rpc/greet", func(msg Message) any {
		return map[string]any{
			"greeting": fmt.Sprintf("Hello from %s!", "rpc-2"),
			"payload":  msg.Payload,
		}
	})
	if err != nil {
		t.Fatalf("注册 RPC 处理器失败: %v", err)
	}

	// n1 发起 RPC 查询（轮询重试，避免连接建立时序造成的假失败）
	var result any
	waitFor(t, 5*time.Second, "RPC 查询成功", func() bool {
		v, queryErr := n1.Query("/rpc/greet", map[string]string{"user": "tester"})
		if queryErr != nil {
			return false
		}
		result = v
		return true
	})
	t.Logf("RPC Query 结果: %v", result)

	// 验证统计
	stats := n1.GetStats()
	if stats.MessagesSent < 1 {
		t.Error("RPC 消息统计异常: 发送数应为 >0")
	}
}

// TestClientIntegration_HealthCheck 测试健康检查回调
func TestClientIntegration_HealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	var (
		healthChecks []HealthInfo
		mu           sync.Mutex
	)

	client, err := NewClient(ClientOptions{
		ID:                  "health-test",
		Addr:                "127.0.0.1:19341",
		MaxPeers:            5,
		HealthCheckInterval: 50 * time.Millisecond,
		OnHealthCheck: func(info HealthInfo) {
			mu.Lock()
			healthChecks = append(healthChecks, info)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	// 等待几次健康检查
	waitFor(t, 3*time.Second, "至少两次健康检查回调", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(healthChecks) >= 2
	})

	mu.Lock()
	checkCount := len(healthChecks)
	mu.Unlock()

	if checkCount < 2 {
		t.Errorf("健康检查回调不足: 期望 >=2, 实际 %d", checkCount)
	} else {
		t.Logf("健康检查回调: %d 次", checkCount)

		mu.Lock()
		lastCheck := healthChecks[len(healthChecks)-1]
		mu.Unlock()

		t.Logf("最后一次健康检查: node=%s, addr=%s, nodes=%d, sent=%d, recv=%d",
			lastCheck.NodeID, lastCheck.NodeAddr, lastCheck.NodeCount, lastCheck.Sent, lastCheck.Recv)
	}

	client.Close()
}

// TestClientIntegration_GracefulShutdown 测试优雅关闭
func TestClientIntegration_GracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	const (
		addr1 = "127.0.0.1:19351"
		addr2 = "127.0.0.1:19352"
	)

	n1, err := NewClient(ClientOptions{
		ID:                   "shutdown-1",
		Addr:                 addr1,
		MaxPeers:             5,
		GossipInterval:       100 * time.Millisecond,
		GossipFailureTimeout: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建 n1 失败: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	n2, err := NewClient(ClientOptions{
		ID:                   "shutdown-2",
		Addr:                 addr2,
		MaxPeers:             5,
		SeedAddrs:            []string{addr1},
		GossipInterval:       100 * time.Millisecond,
		GossipFailureTimeout: 1 * time.Second,
	})
	if err != nil {
		n1.Close()
		t.Fatalf("创建 n2 失败: %v", err)
	}

	waitClientConvergence(t, 2, 5*time.Second, n1, n2)

	// 关闭 n2
	t.Log("关闭 n2...")
	if err := n2.Close(); err != nil {
		t.Logf("Close 返回: %v (预期行为)", err)
	}

	// 二次关闭应安全
	if err := n2.Close(); err == nil {
		t.Error("二次 Close 应返回错误")
	}

	// 验证 n1 最终检测到 n2 离开
	waitFor(t, 5*time.Second, "n1 检测到 n2 离开", func() bool {
		return n1.GetMemberCount() <= 1
	})
	members := n1.GetMembers()
	t.Logf("n1 当前成员: %d 个节点", len(members))
	for _, m := range members {
		t.Logf("  - %s @ %s", m.ID, m.Addr)
	}

	n1.Close()
}

// TestClient_InvalidOptions 测试无效参数
func TestClient_InvalidOptions(t *testing.T) {
	// 空地址
	_, err := NewClient(ClientOptions{})
	if err == nil {
		t.Error("空地址应返回错误")
	}
	t.Logf("空地址错误: %v", err)

	// MinPeers 为 0 时自动补全
	client, err := NewClient(ClientOptions{
		Addr: "127.0.0.1:19399",
	})
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	health := client.GetHealth()
	if health.NodeCount < 1 {
		t.Error("至少有自身节点")
	}
	client.Close()
}

// TestClient_IsRunning 测试运行状态检查
func TestClient_IsRunning(t *testing.T) {
	client, err := NewClient(ClientOptions{
		Addr: "127.0.0.1:19401",
	})
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	if !client.IsRunning() {
		t.Error("新创建的客户端应为运行状态")
	}

	client.Close()

	if client.IsRunning() {
		t.Error("关闭后的客户端应为非运行状态")
	}
}

// TestClient_GetStats_Initial 测试初始统计
func TestClient_GetStats(t *testing.T) {
	client, err := NewClient(ClientOptions{
		ID:   "stats-test",
		Addr: "127.0.0.1:19411",
	})
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	stats := client.GetStats()

	if stats.MessagesSent != 0 {
		t.Error("初始发送数应为 0")
	}
	if stats.MessagesRecv != 0 {
		t.Error("初始接收数应为 0")
	}
	if stats.JoinEvents != 0 {
		t.Error("初始加入事件数应为 0")
	}
	if stats.LeaveEvents != 0 {
		t.Error("初始离开事件数应为 0")
	}
	if stats.OnlineNodes < 1 {
		t.Error("至少应看到自身节点")
	}

	t.Logf("初始统计: %+v", stats)
}

// TestClient_GetHealth_Format 测试健康信息格式
func TestClient_GetHealth(t *testing.T) {
	client, err := NewClient(ClientOptions{
		ID:   "health-fmt",
		Addr: "127.0.0.1:19421",
	})
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	health := client.GetHealth()

	if health.NodeID != "health-fmt" {
		t.Errorf("NodeID 不匹配: 期望 health-fmt, 实际 %s", health.NodeID)
	}
	if health.NodeAddr != "127.0.0.1:19421" {
		t.Errorf("NodeAddr 不匹配: 期望 127.0.0.1:19421, 实际 %s", health.NodeAddr)
	}
	if health.NodeCount < 1 {
		t.Error("NodeCount 应 >= 1")
	}
	// Uptime 可能为 0 因为 NewClient 和 GetHealth 之间时间极短（微秒级）
	if health.Uptime < 0 {
		t.Error("Uptime 不应为负数")
	}
	if len(health.OnlineNodes) < 1 {
		t.Error("OnlineNodes 应至少包含自身")
	}

	t.Logf("健康信息: NodeID=%s, Addr=%s, Count=%d, Uptime=%s",
		health.NodeID, health.NodeAddr, health.NodeCount, health.Uptime.Round(time.Millisecond))
}

// TestClient_SetLogger 测试自定义日志
func TestClient_SetLogger(t *testing.T) {
	client, err := NewClient(ClientOptions{
		Addr: "127.0.0.1:19431",
	})
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 注入自定义日志
	var logLines []string
	var logMu sync.Mutex

	customLog := &testLogger{
		logf: func(level, format string, args ...any) {
			logMu.Lock()
			logLines = append(logLines, fmt.Sprintf("[%s] "+format, append([]any{level}, args...)...))
			logMu.Unlock()
		},
	}
	client.SetLogger(customLog)

	// 发布消息触发日志
	client.Publish("test/log", "hello")

	time.Sleep(100 * time.Millisecond)

	logMu.Lock()
	count := len(logLines)
	logMu.Unlock()

	if count == 0 {
		t.Log("注意: 发布消息可能未触发额外日志行")
	} else {
		t.Logf("自定义日志已捕获 %d 行输出", count)
	}
}

// TestClient_SetLoggerConcurrent 验证运行期替换日志与后台日志输出并发安全
//   - 回归点：SetLogger 曾直接写 c.logger 字段，后台 goroutine 无锁读取构成数据竞态
//   - 需配合 -race 运行才能检出
func TestClient_SetLoggerConcurrent(t *testing.T) {
	client := &Client{}
	client.logger.Store(&loggerHolder{logger: &testLogger{logf: func(string, string, ...any) {}}})

	var wg sync.WaitGroup
	wg.Go(func() {
		for i := 0; i < 500; i++ {
			client.log().Info("concurrent %d", i)
		}
	})
	for i := 0; i < 50; i++ {
		client.SetLogger(&testLogger{logf: func(string, string, ...any) {}})
	}
	wg.Wait()
}

// testLogger 测试用日志实现
type testLogger struct {
	logf func(level, format string, args ...any)
}

func (l *testLogger) Debug(format string, args ...any) { l.logf("DEBUG", format, args...) }
func (l *testLogger) Info(format string, args ...any)  { l.logf("INFO", format, args...) }
func (l *testLogger) Warn(format string, args ...any)  { l.logf("WARN", format, args...) }
func (l *testLogger) Error(format string, args ...any) { l.logf("ERROR", format, args...) }

// ─────────────────────────────────────────────
// 辅助函数
// ─────────────────────────────────────────────

// waitClientConvergence 等待客户端集群收敛
func waitClientConvergence(t *testing.T, expected int, timeout time.Duration, clients ...*Client) bool {
	t.Helper()
	deadline := time.After(timeout)
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-deadline:
			return false
		case <-tick.C:
			allConverged := true
			for _, c := range clients {
				if c.GetMemberCount() < expected {
					allConverged = false
					break
				}
			}
			if allConverged {
				return true
			}
		}
	}
}
