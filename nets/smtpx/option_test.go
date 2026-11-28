package smtpx

import (
	"crypto/tls"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	client := NewClient("smtp.example.com", 465)

	if got := client.GetAddr(); got != "smtp.example.com:465" {
		t.Fatalf("GetAddr() = %q, want %q", got, "smtp.example.com:465")
	}
	if client.startTLS {
		t.Fatal("默认应使用隐式 TLS，startTLS 应为 false")
	}
	if client.dialTimeout != 10*time.Second {
		t.Fatalf("默认拨号超时 = %v, want 10s", client.dialTimeout)
	}
	if client.timeout != 0 {
		t.Fatalf("默认整体超时 = %v, want 0", client.timeout)
	}
	if client.tlsConfig.ServerName != "smtp.example.com" {
		t.Fatalf("默认 ServerName = %q, want %q", client.tlsConfig.ServerName, "smtp.example.com")
	}
}

func TestWithTLSConfig(t *testing.T) {
	config := &tls.Config{ServerName: "mail.example.com", InsecureSkipVerify: true}
	client := NewClient("smtp.example.com", 465, WithTLSConfig(config))

	if client.tlsConfig.ServerName != "mail.example.com" {
		t.Fatalf("WithTLSConfig 未生效: %q", client.tlsConfig.ServerName)
	}
	if client.tlsConfig == config {
		t.Fatal("应克隆 TLS 配置，不能持有调用方对象")
	}

	// 配置未指定 ServerName 时回退到服务器地址
	fallback := NewClient("smtp.example.com", 465, WithTLSConfig(&tls.Config{}))
	if fallback.tlsConfig.ServerName != "smtp.example.com" {
		t.Fatalf("ServerName 回退失败: %q", fallback.tlsConfig.ServerName)
	}

	// nil 配置应被忽略，保留默认配置
	ignored := NewClient("smtp.example.com", 465, WithTLSConfig(nil))
	if ignored.tlsConfig == nil || ignored.tlsConfig.ServerName != "smtp.example.com" {
		t.Fatalf("nil 配置不应覆盖默认值: %+v", ignored.tlsConfig)
	}
}

func TestNewClientDefaultAddr(t *testing.T) {
	tests := []struct {
		server string
		port   int
		want   string
	}{
		{"smtp.example.com", 465, "smtp.example.com:465"},
		{"127.0.0.1", 25, "127.0.0.1:25"},
		{"::1", 587, "[::1]:587"},
		{"2001:db8::1", 25, "[2001:db8::1]:25"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			client := NewClient(tt.server, tt.port)
			if got := client.GetAddr(); got != tt.want {
				t.Fatalf("GetAddr() = %q, want %q", got, tt.want)
			}

			// 地址必须能被标准库无损解析，否则 dial 阶段必然失败
			host, port, err := net.SplitHostPort(client.GetAddr())
			if err != nil {
				t.Fatalf("GetAddr() = %q 不是合法的 host:port: %v", client.GetAddr(), err)
			}
			if host != tt.server {
				t.Fatalf("SplitHostPort host = %q, want %q", host, tt.server)
			}
			if port != strconv.Itoa(tt.port) {
				t.Fatalf("SplitHostPort port = %q, want %s", port, strconv.Itoa(tt.port))
			}
		})
	}
}

func TestWithTLSConfigIsolatedFromCaller(t *testing.T) {
	config := &tls.Config{ServerName: "mail.example.com"}
	client := NewClient("smtp.example.com", 465, WithTLSConfig(config))

	// 克隆之后修改调用方配置，不应影响客户端持有的副本
	config.ServerName = "mutated.example.com"
	config.InsecureSkipVerify = true

	if client.tlsConfig.ServerName != "mail.example.com" {
		t.Fatalf("客户端配置与调用方共享: ServerName = %q", client.tlsConfig.ServerName)
	}
	if client.tlsConfig.InsecureSkipVerify {
		t.Fatal("客户端配置与调用方共享: InsecureSkipVerify 被联动修改")
	}
}

func TestWithDialTimeout(t *testing.T) {
	client := NewClient("smtp.example.com", 465, WithDialTimeout(3*time.Second))
	if client.dialTimeout != 3*time.Second {
		t.Fatalf("dialTimeout = %v, want 3s", client.dialTimeout)
	}

	// 非正值应被忽略，保留默认值
	client = NewClient("smtp.example.com", 465, WithDialTimeout(0))
	if client.dialTimeout != 10*time.Second {
		t.Fatalf("dialTimeout = %v, want 默认 10s", client.dialTimeout)
	}
}

func TestWithTimeout(t *testing.T) {
	client := NewClient("smtp.example.com", 465, WithTimeout(30*time.Second))
	if client.timeout != 30*time.Second {
		t.Fatalf("timeout = %v, want 30s", client.timeout)
	}

	// 0 表示不限制，属于合法取值
	client = NewClient("smtp.example.com", 465, WithTimeout(0))
	if client.timeout != 0 {
		t.Fatalf("timeout = %v, want 0", client.timeout)
	}

	// 负值应被忽略
	client = NewClient("smtp.example.com", 465, WithTimeout(-time.Second))
	if client.timeout != 0 {
		t.Fatalf("timeout = %v, want 0", client.timeout)
	}
}

func TestWithStartTLS(t *testing.T) {
	client := NewClient("smtp.example.com", 587, WithStartTLS())
	if !client.startTLS {
		t.Fatal("WithStartTLS 未生效")
	}
}
