package tcp

import (
	"time"

	"gitee.com/leminewx/gokit/nets/codec"
)

// Option 定义 TCP 客户端选项
type Option func(*Client)

// WithDialTimeout 指定拨号超时时间，默认 10s
func WithDialTimeout(timeout time.Duration) Option {
	return func(own *Client) {
		if timeout > 0 {
			own.dialTimeout = timeout
		}
	}
}

// WithReadTimeout 指定读取超时时间，默认 30s
func WithReadTimeout(timeout time.Duration) Option {
	return func(own *Client) {
		if timeout > 0 {
			own.readTimeout = timeout
		}
	}
}

// WithWriteTimeout 指定写入超时时间，默认 30s
func WithWriteTimeout(timeout time.Duration) Option {
	return func(own *Client) {
		if timeout > 0 {
			own.writeTimeout = timeout
		}
	}
}

// WithCodec 指定数据编解码器，默认使用 4096 字节的原始编解码器
func WithCodec(source codec.Codec) Option {
	return func(own *Client) {
		if source != nil {
			own.codec = source
		}
	}
}

// WithReconnect 启用自动重连
//   - interval: 重连失败后的等待间隔
//   - retries: 最大重连次数：省略表示无限重连，0 表示关闭自动重连，>0 表示最大次数
func WithReconnect(interval time.Duration, retries ...int) Option {
	return func(own *Client) {
		own.applyReconnect(interval, retries...)
	}
}

// WithKeepAlive 设置是否启用 TCP 保活，默认启用
func WithKeepAlive(enabled bool) Option {
	return func(own *Client) {
		own.keepAlive = enabled
	}
}
