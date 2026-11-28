package tcp

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestNewClientWithOptionsDefaults(t *testing.T) {
	host, port := startEchoServer(t)

	client, err := NewClientWithOptions(host, port)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	if client.dialTimeout != 10*time.Second || client.readTimeout != 30*time.Second || client.writeTimeout != 30*time.Second {
		t.Fatalf("默认超时不正确: dial=%v read=%v write=%v", client.dialTimeout, client.readTimeout, client.writeTimeout)
	}
	if !client.keepAlive {
		t.Fatal("默认应启用 keepalive")
	}
	if !client.IsConnected() || client.IsClosed() {
		t.Fatal("新建客户端应处于已连接且未关闭状态")
	}
	if client.GetLocalAddr() == nil || client.GetRemoteAddr() == nil {
		t.Fatal("已连接时应能获取本地与远端地址")
	}
}

func TestNewClientWithOptionsApplied(t *testing.T) {
	host, port := startEchoServer(t)

	client, err := NewClientWithOptions(host, port,
		WithDialTimeout(2*time.Second),
		WithReadTimeout(time.Second),
		WithWriteTimeout(time.Second),
		WithReconnect(50*time.Millisecond, 3),
		WithKeepAlive(false),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	if client.dialTimeout != 2*time.Second || client.readTimeout != time.Second || client.writeTimeout != time.Second {
		t.Fatalf("选项未生效: dial=%v read=%v write=%v", client.dialTimeout, client.readTimeout, client.writeTimeout)
	}
	if client.maxReconnectRetries != 3 || client.reconnectRetryInterval != 50*time.Millisecond {
		t.Fatalf("重连选项未生效: retries=%d interval=%v", client.maxReconnectRetries, client.reconnectRetryInterval)
	}
	if client.keepAlive {
		t.Fatal("WithKeepAlive(false) 未生效")
	}
}

func TestClient_ExecuteContextSuccess(t *testing.T) {
	host, port := startEchoServer(t)

	client, err := NewClient(host, port, time.Second)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.ExecuteContext(ctx, []byte("ping"), 10*time.Millisecond)
	if err != nil {
		t.Fatalf("ExecuteContext failed: %v", err)
	}
	if string(resp) != "ping" {
		t.Fatalf("expected echo %q, got %q", "ping", string(resp))
	}

	stats := client.GetStats()
	if stats.WriteBytes != 4 || stats.ReadBytes != 4 {
		t.Fatalf("统计不正确: %+v", stats)
	}
}

func TestClient_ExecuteContextCancel(t *testing.T) {
	// 服务端接受连接但永不响应，用于模拟读阻塞
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	client, err := NewClient(addr.IP.String(), addr.Port, time.Second)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = client.ExecuteContext(ctx, []byte("ping"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("取消响应过慢: %v", elapsed)
	}
}

func TestClient_ExecuteContextAlreadyCancelled(t *testing.T) {
	host, port := startEchoServer(t)

	client, err := NewClient(host, port, time.Second)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := client.ExecuteContext(ctx, []byte("ping")); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestClient_Deadline(t *testing.T) {
	client := &Client{}

	// 非正超时表示不设限：无 ctx 截止时间时返回零值，即无 deadline
	if got := client.deadline(context.Background(), 0); !got.IsZero() {
		t.Fatalf("timeout=0 应返回零值 deadline, got %v", got)
	}
	if got := client.deadline(context.Background(), -time.Second); !got.IsZero() {
		t.Fatalf("负超时应返回零值 deadline, got %v", got)
	}

	if got := client.deadline(context.Background(), time.Second); time.Until(got) <= 0 {
		t.Fatalf("正超时应得到未来的 deadline, got %v", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	contextDeadline, _ := ctx.Deadline()

	// 无显式超时或超时更宽时，以 ctx 截止时间为准
	if got := client.deadline(ctx, 0); !got.Equal(contextDeadline) {
		t.Fatalf("timeout=0 应使用 ctx 截止时间, got %v want %v", got, contextDeadline)
	}
	if got := client.deadline(ctx, time.Hour); !got.Equal(contextDeadline) {
		t.Fatalf("超时超过 ctx 时应使用 ctx 截止时间, got %v want %v", got, contextDeadline)
	}
	if got := client.deadline(ctx, time.Millisecond); time.Until(got) > time.Second {
		t.Fatalf("超时早于 ctx 时应使用超时值, got %v", got)
	}
}

// TestClient_ZeroReadTimeoutMeansNoDeadline 验证 SetReadTimeout(0) 表示不设读超时，而不是立即超时
//   - 回归点：deadline 曾返回 time.Now()，使读立即返回 i/o timeout 并丢弃连接
func TestClient_ZeroReadTimeoutMeansNoDeadline(t *testing.T) {
	// 服务端接受连接但永不发送数据
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	accepted := make(chan struct{})
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		close(accepted)
		t.Cleanup(func() { _ = conn.Close() })
		_, _ = conn.Read(make([]byte, 1))
	}()

	addr := listener.Addr().(*net.TCPAddr)
	client, err := NewClient(addr.IP.String(), addr.Port, time.Second)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	<-accepted
	client.SetReadTimeout(0)

	readErr := make(chan error, 1)
	go func() {
		_, err := client.Read()
		readErr <- err
	}()

	select {
	case err := <-readErr:
		t.Fatalf("timeout=0 时 Read 不应立即返回, got %v", err)
	case <-time.After(150 * time.Millisecond):
	}

	// 关闭连接应让阻塞中的 Read 立即返回错误
	_ = client.Close()
	select {
	case err := <-readErr:
		if err == nil {
			t.Fatal("关闭连接后 Read 应返回错误")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("关闭连接后 Read 未返回")
	}
}

func TestClient_CloseUpdatesState(t *testing.T) {
	host, port := startEchoServer(t)

	client, err := NewClient(host, port, time.Second)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}
	if !client.IsClosed() || client.IsConnected() {
		t.Fatal("Close 后应处于已关闭且未连接状态")
	}
	if client.GetLocalAddr() != nil || client.GetRemoteAddr() != nil {
		t.Fatal("Close 后不应返回连接地址")
	}

	if _, err := client.ExecuteContext(context.Background(), []byte("ping")); !errors.Is(err, ErrConnClosed) {
		t.Fatalf("expected ErrConnClosed, got %v", err)
	}
}

// startEchoServer 启动本地回显服务，避免依赖真实设备
func startEchoServer(t *testing.T) (host string, port int) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			go func() {
				defer conn.Close()
				buf := make([]byte, 1024)
				for {
					n, err := conn.Read(buf)
					if err != nil {
						return
					}
					if _, err := conn.Write(buf[:n]); err != nil {
						return
					}
				}
			}()
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port
}

func TestClient_ExecuteWithLocalEcho(t *testing.T) {
	host, port := startEchoServer(t)

	client, err := NewClient(host, port, time.Second)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	resp, err := client.Execute([]byte("ping"), 10*time.Millisecond)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if string(resp) != "ping" {
		t.Fatalf("expected echo %q, got %q", "ping", string(resp))
	}
}

func TestClient_CloseUnblocksPendingRead(t *testing.T) {
	// 服务端只接受连接、永不响应，用于模拟读阻塞
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	client, err := NewClient(addr.IP.String(), addr.Port, time.Second)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() {
		select {
		case conn := <-accepted:
			_ = conn.Close()
		default:
		}
	}()

	done := make(chan struct{})
	go func() {
		_, _ = client.Execute([]byte("ping"))
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	_ = client.Close()

	select {
	case <-done:
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Fatalf("Close blocked for %v while a read was pending", elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Close did not unblock the pending read")
	}
}

// TestApplyReconnectSemantics 验证重连参数语义：省略=无限、0=关闭、>0=最大次数
//   - 回归点：显式传 0 曾被误判为无限重连
func TestApplyReconnectSemantics(t *testing.T) {
	cases := []struct {
		name    string
		retries []int
		want    int
	}{
		{"omitted means infinite", nil, -1},
		{"zero disables", []int{0}, 0},
		{"negative disables", []int{-1}, 0},
		{"positive limits", []int{5}, 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &Client{}
			client.applyReconnect(time.Second, tc.retries...)

			if client.maxReconnectRetries != tc.want {
				t.Fatalf("maxReconnectRetries = %d, want %d", client.maxReconnectRetries, tc.want)
			}
			if client.reconnectRetryInterval != time.Second {
				t.Fatalf("reconnectRetryInterval = %v, want %v", client.reconnectRetryInterval, time.Second)
			}
		})
	}
}

// TestSetReconnectZeroDisables 验证 SetReconnect(interval, 0) 可关闭已启用的无限重连
func TestSetReconnectZeroDisables(t *testing.T) {
	client := &Client{}

	client.SetReconnect(time.Second)
	if client.maxReconnectRetries != -1 {
		t.Fatalf("omitted retries should be infinite, got %d", client.maxReconnectRetries)
	}

	client.SetReconnect(time.Second, 0)
	if client.maxReconnectRetries != 0 {
		t.Fatalf("explicit zero should disable reconnect, got %d", client.maxReconnectRetries)
	}
}

// TestNewClientReconnectDisabledByDefault 验证新建客户端默认不自动重连
func TestNewClientReconnectDisabledByDefault(t *testing.T) {
	host, port := startEchoServer(t)

	client, err := NewClient(host, port, time.Second)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	if client.maxReconnectRetries != 0 {
		t.Fatalf("default reconnect should be disabled, got %d", client.maxReconnectRetries)
	}
}

func TestPackageSmoke(t *testing.T) {
	// 包级烟雾测试：确保包可正常编译与初始化
	t.Log("tcp package loaded")
}
