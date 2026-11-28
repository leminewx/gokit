package tcp

import (
	"testing"
)

func TestClientStatsZeroValue(t *testing.T) {
	var client Client

	if got := client.GetStats(); got != (Stats{}) {
		t.Fatalf("零值客户端统计 = %+v, want 零值", got)
	}

	if client.IsClosed() {
		t.Fatal("零值客户端不应处于关闭状态")
	}
	if client.IsConnected() {
		t.Fatal("零值客户端不应处于已连接状态")
	}
	if client.GetLocalAddr() != nil {
		t.Fatal("无连接时 LocalAddr 应返回 nil")
	}
	if client.GetRemoteAddr() != nil {
		t.Fatal("无连接时 RemoteAddr 应返回 nil")
	}
}

func TestClientStatsSnapshot(t *testing.T) {
	var client Client
	client.readBytes.Store(100)
	client.writeBytes.Store(200)
	client.reconnects.Store(3)
	client.dialErrors.Store(1)

	want := Stats{ReadBytes: 100, WriteBytes: 200, Reconnects: 3, DialErrors: 1}
	if got := client.GetStats(); got != want {
		t.Fatalf("GetStats() = %+v, want %+v", got, want)
	}

	// 快照返回副本，后续计数变化不影响已获取的统计
	client.readBytes.Store(150)
	if got := client.GetStats().ReadBytes; got != 150 {
		t.Fatalf("再次 GetStats().ReadBytes = %d, want 150", got)
	}
}

func TestClientIsClosed(t *testing.T) {
	var client Client

	client.closed.Store(true)
	if !client.IsClosed() {
		t.Fatal("closed 置位后 IsClosed 应返回 true")
	}
	if client.IsConnected() {
		t.Fatal("已关闭的客户端不应报告已连接")
	}
}
