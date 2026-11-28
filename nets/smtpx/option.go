package smtpx

import (
	"crypto/tls"
	"time"
)

// Option 定义 SMTP 客户端选项
type Option func(*Client)

// WithTLSConfig 指定 TLS 配置
//   - 非空配置会被克隆，客户端不会修改调用方的配置对象
//   - 若配置未设置 ServerName，则沿用构造时的服务器地址
func WithTLSConfig(config *tls.Config) Option {
	return func(own *Client) {
		if config != nil {
			own.tlsConfig = config.Clone()
		}
	}
}

// WithDialTimeout 指定建立连接（含 TLS 握手）的超时时间，默认 10s
func WithDialTimeout(timeout time.Duration) Option {
	return func(own *Client) {
		if timeout > 0 {
			own.dialTimeout = timeout
		}
	}
}

// WithTimeout 指定单次发送的整体超时时间
//   - 覆盖 TLS 握手、认证与数据传输全过程，0 表示不限制
func WithTimeout(timeout time.Duration) Option {
	return func(own *Client) {
		if timeout >= 0 {
			own.timeout = timeout
		}
	}
}

// WithStartTLS 使用明文连接后通过 STARTTLS 升级为加密连接
//   - 适用于 587 等端口；默认使用 465 端口的隐式 TLS
func WithStartTLS() Option {
	return func(own *Client) {
		own.startTLS = true
	}
}
