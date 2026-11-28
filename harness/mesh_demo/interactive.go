package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gitee.com/leminewx/gokit/nets/mesh"
)

// runInteractive 运行交互式命令循环
func runInteractive(client *mesh.Client, nodeID string) error {
	fmt.Println()
	fmt.Println("┌──────────────────────────────────────────────────────┐")
	fmt.Println("│           Mesh Client 交互式 Demo                     │")
	fmt.Println("├──────────────────────────────────────────────────────┤")
	fmt.Println("│  pub <topic> <msg>    发布消息到主题                  │")
	fmt.Println("│  put <key> <value>    存储键值                        │")
	fmt.Println("│  get <key>            查询键值                        │")
	fmt.Println("│  query <path> [data]  发起 RPC 查询                   │")
	fmt.Println("│  members              查看集群成员                    │")
	fmt.Println("│  stats                查看运行时统计                  │")
	fmt.Println("│  health               查看健康状态                    │")
	fmt.Println("│  help                 显示帮助                        │")
	fmt.Println("│  exit                 退出 (Ctrl+C 也可)              │")
	fmt.Println("└──────────────────────────────────────────────────────┘")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("mesh> ")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Print("mesh> ")
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			fmt.Print("mesh> ")
			continue
		}

		cmd := parts[0]

		switch cmd {
		case "pub":
			handlePub(client, parts)
		case "put":
			handlePut(client, parts)
		case "get":
			handleGet(client, parts)
		case "query":
			handleQuery(client, parts)
		case "members", "ls":
			handleMembers(client)
		case "stats":
			handleStats(client)
		case "health":
			handleHealth(client)
		case "help", "?":
			handleHelp()
		case "exit", "quit":
			log.Println("通过交互命令退出...")
			os.Exit(0)
		default:
			log.Printf("未知命令: %s (输入 help 查看帮助)", cmd)
		}

		fmt.Print("mesh> ")
	}

	return scanner.Err()
}

func handlePub(client *mesh.Client, parts []string) {
	if len(parts) < 3 {
		log.Println("用法: pub <topic> <message>")
		return
	}
	topic := parts[1]
	msg := strings.Join(parts[2:], " ")

	if err := client.Publish(topic, msg); err != nil {
		log.Printf("发布失败: %v", err)
	} else {
		log.Printf("✓ 已发布到 %s: %s", topic, msg)
	}
}

func handlePut(client *mesh.Client, parts []string) {
	if len(parts) < 3 {
		log.Println("用法: put <key> <value>")
		return
	}
	key := parts[1]

	// 尝试将 value 解析为数字，否则当作字符串
	var value any
	raw := strings.Join(parts[2:], " ")
	if n, err := strconv.ParseFloat(raw, 64); err == nil {
		value = n
	} else {
		value = raw
	}

	if err := client.Put(key, value); err != nil {
		log.Printf("存储失败: %v", err)
	} else {
		log.Printf("✓ 已存储 %s = %v", key, value)
	}
}

func handleGet(client *mesh.Client, parts []string) {
	if len(parts) < 2 {
		log.Println("用法: get <key>")
		return
	}
	key := parts[1]

	val, err := client.Get(key)
	if err != nil {
		log.Printf("查询失败: %v", err)
	} else {
		log.Printf("Get %s = %v", key, val)
	}
}

func handleQuery(client *mesh.Client, parts []string) {
	if len(parts) < 2 {
		log.Println("用法: query <path> [payload]")
		return
	}
	path := parts[1]

	var payload any
	if len(parts) > 2 {
		payload = strings.Join(parts[2:], " ")
	}

	result, err := client.Query(path, payload)
	if err != nil {
		log.Printf("RPC 查询失败: %v", err)
	} else {
		log.Printf("Query %s → %v", path, result)
	}
}

func handleMembers(client *mesh.Client) {
	members := client.GetMembers()
	fmt.Printf("──── 集群成员 (共 %d 个) ────\n", len(members))
	for i, m := range members {
		seen := time.Since(m.Timestamp).Round(time.Millisecond)
		marker := "├─"
		if i == len(members)-1 {
			marker = "└─"
		}
		fmt.Printf(" %s %s @ %s  SN=%d  (心跳: %s ago)\n", marker, m.ID, m.Addr, m.SN, seen)
	}
}

func handleStats(client *mesh.Client) {
	stats := client.GetStats()
	fmt.Println("──── 运行时统计 ────")
	fmt.Printf("  启动时间:   %s\n", stats.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  已发送:     %d 条\n", stats.MessagesSent)
	fmt.Printf("  已接收:     %d 条\n", stats.MessagesRecv)
	fmt.Printf("  在线节点:   %d\n", stats.OnlineNodes)
	fmt.Printf("  加入事件:   %d\n", stats.JoinEvents)
	fmt.Printf("  离开事件:   %d\n", stats.LeaveEvents)
}

func handleHealth(client *mesh.Client) {
	health := client.GetHealth()
	fmt.Println("──── 健康状态 ────")
	fmt.Printf("  节点 ID:     %s\n", health.NodeID)
	fmt.Printf("  节点地址:    %s\n", health.NodeAddr)
	fmt.Printf("  在线节点数:  %d\n", health.NodeCount)
	fmt.Printf("  运行时长:    %s\n", health.Uptime.Round(time.Second))
	fmt.Printf("  已发送:      %d\n", health.Sent)
	fmt.Printf("  已接收:      %d\n", health.Recv)
	fmt.Printf("  在线节点:")
	for _, id := range health.OnlineNodes {
		fmt.Printf(" %s", id)
	}
	fmt.Println()
}

func handleHelp() {
	fmt.Println("可用命令:")
	fmt.Println("  pub <topic> <msg>    发布消息")
	fmt.Println("  put <key> <value>    存储键值")
	fmt.Println("  get <key>            查询键值")
	fmt.Println("  query <path> [data]  RPC 查询")
	fmt.Println("  members              查看成员")
	fmt.Println("  stats                查看统计")
	fmt.Println("  health               健康状态")
	fmt.Println("  help                 显示此帮助")
	fmt.Println("  exit                 退出")
}
