//go:build integration

// 环境依赖测试：需要指定本机网卡并能够访问外部地址，默认不参与构建与测试。
//
// 执行方式：
//
//	$env:GOKIT_NET_INTERFACE="以太网"; $env:GOKIT_NET_PROBE_ADDR="8.8.8.8:80"
//	go test -tags integration -run TestGetPublicIP -v ./nets
//
// 未设置 GOKIT_NET_INTERFACE 时用例自动跳过；GOKIT_NET_PROBE_ADDR 缺省为 8.8.8.8:80。

package nets

import (
	"net"
	"os"
	"testing"
)

func TestGetPublicIP(t *testing.T) {
	interfaceName := os.Getenv("GOKIT_NET_INTERFACE")
	if interfaceName == "" {
		t.Skip("未设置 GOKIT_NET_INTERFACE，跳过网卡依赖用例")
	}

	probeAddr := os.Getenv("GOKIT_NET_PROBE_ADDR")
	if probeAddr == "" {
		probeAddr = "8.8.8.8:80"
	}

	ip, err := GetNetPreferredIP(interfaceName)
	if err != nil {
		t.Fatalf("GetNetPreferredIP(%q) 失败: %v", interfaceName, err)
	}
	t.Logf("选中的源 IP: %s", ip)

	// 绑定该 IP 发起连接，验证路由可用
	dialer := &net.Dialer{LocalAddr: &net.TCPAddr{IP: ip}}
	conn, err := dialer.Dial("tcp", probeAddr)
	if err != nil {
		t.Fatalf("绑定 %s 连接 %s 失败: %v", ip, probeAddr, err)
	}
	if err := conn.Close(); err != nil {
		t.Errorf("关闭连接失败: %v", err)
	}
}
