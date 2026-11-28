package tcp

import (
	"testing"
	"time"

	"gitee.com/leminewx/gokit/nets/codec"
)

func TestWithTimeouts(t *testing.T) {
	client := &Client{}
	WithDialTimeout(3 * time.Second)(client)
	WithReadTimeout(5 * time.Second)(client)
	WithWriteTimeout(7 * time.Second)(client)

	if client.dialTimeout != 3*time.Second {
		t.Fatalf("dialTimeout = %v, want 3s", client.dialTimeout)
	}
	if client.readTimeout != 5*time.Second {
		t.Fatalf("readTimeout = %v, want 5s", client.readTimeout)
	}
	if client.writeTimeout != 7*time.Second {
		t.Fatalf("writeTimeout = %v, want 7s", client.writeTimeout)
	}
}

func TestWithTimeoutsIgnoreNonPositive(t *testing.T) {
	client := &Client{}
	WithDialTimeout(0)(client)
	WithReadTimeout(-time.Second)(client)
	WithWriteTimeout(-time.Second)(client)

	if client.dialTimeout != 0 || client.readTimeout != 0 || client.writeTimeout != 0 {
		t.Fatalf("非正超时值不应覆盖默认值: dial=%v read=%v write=%v",
			client.dialTimeout, client.readTimeout, client.writeTimeout)
	}
}

func TestWithCodec(t *testing.T) {
	client := &Client{}

	// nil 编解码器应被忽略
	WithCodec(nil)(client)
	if client.codec != nil {
		t.Fatal("nil 编解码器不应被设置")
	}

	source := codec.NewRawCodec(1024)
	WithCodec(source)(client)
	if client.codec != source {
		t.Fatal("WithCodec 未设置指定的编解码器")
	}
}

func TestWithReconnect(t *testing.T) {
	tests := []struct {
		name         string
		retries      []int
		wantInterval time.Duration
		wantRetries  int
	}{
		{"省略次数表示无限重连", nil, 2 * time.Second, -1},
		{"0 表示关闭重连", []int{0}, 2 * time.Second, 0},
		{"正数表示最大次数", []int{3}, 2 * time.Second, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			WithReconnect(2*time.Second, tt.retries...)(client)

			if client.reconnectRetryInterval != tt.wantInterval {
				t.Fatalf("reconnectRetryInterval = %v, want %v", client.reconnectRetryInterval, tt.wantInterval)
			}
			if client.maxReconnectRetries != tt.wantRetries {
				t.Fatalf("maxReconnectRetries = %d, want %d", client.maxReconnectRetries, tt.wantRetries)
			}
		})
	}
}

func TestWithReconnectIgnoreNegativeInterval(t *testing.T) {
	client := &Client{reconnectRetryInterval: time.Second}
	WithReconnect(-time.Second)(client)

	if client.reconnectRetryInterval != time.Second {
		t.Fatalf("负间隔不应覆盖原值: %v", client.reconnectRetryInterval)
	}
}

func TestWithKeepAlive(t *testing.T) {
	client := &Client{keepAlive: true}
	WithKeepAlive(false)(client)
	if client.keepAlive {
		t.Fatal("WithKeepAlive(false) 未生效")
	}

	WithKeepAlive(true)(client)
	if !client.keepAlive {
		t.Fatal("WithKeepAlive(true) 未生效")
	}
}
