package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitee.com/leminewx/gokit/nets/mesh"
)

// ******************************************************************************************************************
// Mesh Client Demo
//
// 启动方式（先启动种子节点，再启动普通节点）：
//
//	种子节点：  go run main.go -addr :9001 -id seed
//	普通节点：  go run main.go -addr :9002 -id node1 -seeds :9001
//
// 功能演示：
//   - Gossip 自动发现
//   - Pub/Sub 消息广播
//   - KV 存储 (Put/Get)
//   - RPC 查询
//   - 健康检查
//   - 优雅关闭
// ******************************************************************************************************************

func main() {
	// ──── 解析命令行参数 ────
	cfg := parseFlags()

	log.SetFlags(log.Ltime | log.Lmicroseconds)
	log.Printf("========================================")
	log.Printf("  Mesh Client Demo")
	log.Printf("  ID:     %s", cfg.id)
	log.Printf("  Addr:   0.0.0.0%s", cfg.addr)
	log.Printf("  Seeds:  %v", cfg.seedAddrs)
	log.Printf("  MaxPeers: %d", cfg.maxPeers)
	log.Printf("========================================")
	log.Println()

	// ──── 创建 Mesh 客户端 ────
	var (
		joinCount  int
		leaveCount int
	)

	client, err := mesh.NewClient(mesh.ClientOptions{
		ID:                  cfg.id,
		Addr:                "0.0.0.0" + cfg.addr,
		MaxPeers:            cfg.maxPeers,
		SeedAddrs:           cfg.seedAddrs,
		Debug:               true,
		GossipInterval:      1 * time.Second,
		HealthCheckInterval: 10 * time.Second,
		OnJoin: func(nodeID, addr string) {
			joinCount++
			log.Printf("[EVENT] ✓ 节点加入: id=%s, addr=%s (累计加入: %d)", nodeID, addr, joinCount)
		},
		OnLeave: func(nodeID string) {
			leaveCount++
			log.Printf("[EVENT] ✗ 节点离开: id=%s (累计离开: %d)", nodeID, leaveCount)
		},
		OnHealthCheck: func(info mesh.HealthInfo) {
			log.Printf("[HEALTH] 在线节点: %d, 运行: %s, 发送: %d, 接收: %d",
				info.NodeCount, info.Uptime.Round(time.Second), info.Sent, info.Recv)
		},
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	// ──── 注册 RPC 处理器 ────
	if err := client.RegisterQueryHandler("/rpc/status", handleStatus); err != nil {
		log.Printf("注册 status 处理器失败: %v", err)
	}
	if err := client.RegisterQueryHandler("/rpc/echo", handleEcho); err != nil {
		log.Printf("注册 echo 处理器失败: %v", err)
	}

	// ──── 订阅 Pub/Sub 主题 ────
	if err := client.Subscribe("demo/announce", func(msg mesh.Message, params map[string]string) {
		log.Printf("[PUB]   收到广播: topic=%s, value=%v, from=%s", msg.Path, msg.Value, msg.SenderID)
	}); err != nil {
		log.Printf("订阅 demo/announce 失败: %v", err)
	}

	// ──── 启动后发送一条加入通告 ────
	time.Sleep(2 * time.Second) // 等待 Gossip 发现
	if err := client.Publish("demo/announce", fmt.Sprintf("节点 %s 已就绪", cfg.id)); err != nil {
		log.Printf("发布加入通告失败: %v", err)
	}

	// ──── 定期输出集群状态 ────
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for client.IsRunning() {
			<-ticker.C
			if !client.IsRunning() {
				return
			}

			members := client.GetMembers()
			log.Printf("─── 集群状态 (共 %d 节点) ───", len(members))
			for _, m := range members {
				seen := time.Since(m.Timestamp).Round(time.Millisecond)
				log.Printf("  ├─ %s @ %s (上次心跳: %s ago)", m.ID, m.Addr, seen)
			}

			stats := client.GetStats()
			log.Printf("  统计: 发送=%d 接收=%d 加入事件=%d 离开事件=%d",
				stats.MessagesSent, stats.MessagesRecv, stats.JoinEvents, stats.LeaveEvents)
		}
	}()

	// ──── 交互式命令循环 ────
	go func() {
		if err := runInteractive(client, cfg.id); err != nil {
			log.Printf("交互循环退出: %v", err)
		}
	}()

	// ──── 等待退出信号 ────
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	log.Printf("收到信号 %v，正在关闭...", sig)

	// 发布离开通告
	if err := client.Publish("demo/announce", fmt.Sprintf("节点 %s 正在离开", cfg.id)); err != nil {
		log.Printf("发布离开通告失败: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 关闭客户端
	if err := client.Close(); err != nil {
		log.Printf("关闭异常: %v", err)
	}

	log.Println("Demo 已退出，再见！")
}

// ──── RPC 处理器 ────

func handleStatus(msg mesh.Message) any {
	return map[string]any{
		"service": "mesh-demo",
		"status":  "running",
		"time":    time.Now().Format(time.RFC3339),
	}
}

func handleEcho(msg mesh.Message) any {
	return map[string]any{
		"echo": msg.Payload,
	}
}
