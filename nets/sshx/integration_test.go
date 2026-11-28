//go:build integration

// 外部服务依赖测试：需要可登录的真实 SSH 服务器才能运行，默认不参与构建与测试；
// 本文件同时提供其他集成用例复用的 testClient 辅助函数。
//
// 设备信息通过环境变量注入，避免凭据明文入库：
//
//	$env:GOKIT_SSH_HOST="10.0.0.10"
//	$env:GOKIT_SSH_PASS="******"
//	go test -tags integration -run TestLive ./nets/sshx
//
// 支持的环境变量：
//   - GOKIT_SSH_HOST：目标主机（必填，缺省则所有集成用例跳过）
//   - GOKIT_SSH_PASS：登录密码（必填，缺省则所有集成用例跳过；兼容 GOKIT_SSH_PASSWORD）
//   - GOKIT_SSH_PORT：端口，默认 22
//   - GOKIT_SSH_USER：用户名，默认 root
//
// 注意：session_test.go / stream_test.go 中的旧用例面向特定 Android 设备
// （依赖 /system/bin/tester、passwd、ssh-keygen 等），适配任意设备请用：
//
//	go test -tags integration -run TestLive ./nets/sshx

package sshx

import (
	"os"
	"strconv"
	"testing"
	"time"
)

// liveTarget 集成测试目标设备
type liveTarget struct {
	host string
	port int
	user string
	pass string
}

// envOrDefault 读取环境变量，未设置时返回默认值
func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return def
}

// loadLiveTarget 从环境变量读取设备信息，缺少必填项时跳过测试
func loadLiveTarget(t *testing.T) liveTarget {
	t.Helper()

	host := os.Getenv("GOKIT_SSH_HOST")
	pass := os.Getenv("GOKIT_SSH_PASS")
	if pass == "" {
		pass = os.Getenv("GOKIT_SSH_PASSWORD") // 与 nets/sshx/command 的变量名保持一致
	}
	if host == "" || pass == "" {
		t.Skip("skip integration test: GOKIT_SSH_HOST / GOKIT_SSH_PASS not set")
	}

	port := 22
	if raw := os.Getenv("GOKIT_SSH_PORT"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			port = v
		}
	}

	return liveTarget{
		host: host,
		port: port,
		user: envOrDefault("GOKIT_SSH_USER", "root"),
		pass: pass,
	}
}

// 兼容既有集成用例的变量（同样来自环境变量）
var (
	testHost = os.Getenv("GOKIT_SSH_HOST")
	testUser = envOrDefault("GOKIT_SSH_USER", "root")
	testPass = envOrDefault("GOKIT_SSH_PASS", os.Getenv("GOKIT_SSH_PASSWORD"))
)

// 辅助函数：创建客户端
func testClient(t *testing.T) *Client {
	t.Helper()

	target := loadLiveTarget(t)
	client, err := NewClient(target.host, target.user, target.pass,
		WithPort(target.port),
		WithDebug(1024),
		WithTimeout(10*time.Second),
	)

	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	return client
}
