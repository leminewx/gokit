//go:build integration

// 设备依赖测试：需要真实可用的 TCP 转串口设备才能运行，默认不参与构建与测试；
// 本地回环用例见 client_local_test.go。
//
// 执行方式（未设置的变量对应用例将自动跳过）：
//
//	$env:GOKIT_TCP_HOST="192.0.2.30"; $env:GOKIT_TCP_PORT="50700"; go test -tags integration ./nets/tcp

package tcp

import (
	"os"
	"strconv"
	"testing"
	"time"
)

// integrationClient 按环境变量建立被测客户端，缺省时跳过用例
func integrationClient(t *testing.T) *Client {
	t.Helper()

	host := os.Getenv("GOKIT_TCP_HOST")
	if host == "" {
		t.Skip("未设置 GOKIT_TCP_HOST，跳过设备依赖用例")
	}

	const defaultPort = 50700
	port := defaultPort
	if raw := os.Getenv("GOKIT_TCP_PORT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > 65535 {
			t.Fatalf("GOKIT_TCP_PORT 非法: %q", raw)
		}

		port = parsed
	}

	client, err := NewClient(host, port, 10*time.Second)
	if err != nil {
		t.Fatalf("连接 %s:%d 失败: %v", host, port, err)
	}

	t.Cleanup(func() { _ = client.Close() })

	return client
}

func TestClientIntegration(t *testing.T) {
	client := integrationClient(t)

	for _, cmd := range []string{"AT+CMPWR?\r\n", "AT+CMPWR=?\r\n", "AT+CGMR\r\n", "AT+ISN\r\n"} {
		reply, err := client.Execute([]byte(cmd), 100*time.Millisecond)
		if err != nil {
			t.Fatalf("执行 %q 失败: %v", cmd, err)
		}

		t.Logf("%s => %s", cmd, string(reply))
	}
}
