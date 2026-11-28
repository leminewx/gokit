package sshx

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// newOptionTestClient 构造仅用于选项断言的空客户端，不发起网络连接。
func newOptionTestClient() *Client {
	return &Client{
		host:      "127.0.0.1",
		addr:      "127.0.0.1:22",
		debug:     &atomic.Bool{},
		debugSize: &atomic.Int64{},
		closed:    &atomic.Bool{},
		config:    &ssh.ClientConfig{},
	}
}

func TestWithPort(t *testing.T) {
	t.Parallel()

	client := newOptionTestClient()
	WithPort(2222)(client)

	if client.addr != "127.0.0.1:2222" {
		t.Fatalf("addr = %q, want %q", client.addr, "127.0.0.1:2222")
	}
}

func TestWithTimeout(t *testing.T) {
	t.Parallel()

	client := newOptionTestClient()
	WithTimeout(3 * time.Second)(client)

	if client.config.Timeout != 3*time.Second {
		t.Fatalf("Timeout = %s, want 3s", client.config.Timeout)
	}
}

func TestWithReconnect(t *testing.T) {
	t.Parallel()

	client := newOptionTestClient()
	WithReconnect(3, time.Second)(client)

	if client.maxReconnectRetries != 3 {
		t.Errorf("maxReconnectRetries = %d, want 3", client.maxReconnectRetries)
	}
	if client.reconnectRetryInterval != time.Second {
		t.Errorf("reconnectRetryInterval = %s, want 1s", client.reconnectRetryInterval)
	}
}

func TestWithHostKeyCallback(t *testing.T) {
	t.Parallel()

	client := newOptionTestClient()
	callback := ssh.InsecureIgnoreHostKey()
	WithHostKeyCallback(callback)(client)

	if client.config.HostKeyCallback == nil {
		t.Fatal("HostKeyCallback 未被设置")
	}
	if client.optionErr != nil {
		t.Fatalf("optionErr = %v, want nil", client.optionErr)
	}
}

func TestWithHostKeyCallbackRejectsNil(t *testing.T) {
	t.Parallel()

	client := newOptionTestClient()
	WithHostKeyCallback(nil)(client)

	if client.optionErr == nil {
		t.Fatal("nil 回调解应记录 optionErr")
	}
}

func TestWithKnownHosts(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(filename, nil, 0o600); err != nil {
		t.Fatalf("创建 known_hosts 失败: %v", err)
	}

	client := newOptionTestClient()
	WithKnownHosts(filename)(client)

	if client.optionErr != nil {
		t.Fatalf("optionErr = %v, want nil", client.optionErr)
	}
	if client.config.HostKeyCallback == nil {
		t.Fatal("HostKeyCallback 未被设置")
	}
}

func TestWithKnownHostsMissingFile(t *testing.T) {
	t.Parallel()

	// known_hosts 不存在时应在拨号前直接失败，避免带病连接。
	_, err := NewClient("127.0.0.1", "user", "pwd", WithKnownHosts(filepath.Join(t.TempDir(), "missing")))
	if err == nil {
		t.Fatal("期望返回 known_hosts 加载错误，实际为 nil")
	}
}

func TestWithInsecureIgnoreHostKey(t *testing.T) {
	t.Parallel()

	client := newOptionTestClient()
	WithInsecureIgnoreHostKey()(client)

	if client.config.HostKeyCallback == nil {
		t.Fatal("HostKeyCallback 未被设置")
	}
}
