package sshx

import (
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Option 定义 SSH 客户端的配置项。
type Option func(*Client)

// WithPort 设置连接端口。
func WithPort(port int) Option {
	return func(client *Client) {
		client.addr = fmt.Sprintf("%s:%d", client.host, port)
	}
}

// WithTimeout 设置拨号超时时间。
func WithTimeout(timeout time.Duration) Option {
	return func(client *Client) {
		client.config.Timeout = timeout
	}
}

// WithDebug 设置调试模式。
//   - 调试日志会记录执行的命令内容，但输入参数仅记录长度，不输出具体内容
//     （输入可能是口令），命令本身也可能包含敏感信息，生产环境请谨慎开启。
//   - maxSize 限制单条日志的最大字节数，省略或非正数表示不限制。
func WithDebug(maxSize ...int64) Option {
	return func(client *Client) {
		client.debug.Store(true)
		if len(maxSize) > 0 && maxSize[0] > 0 {
			client.debugSize.Store(maxSize[0])
		}
	}
}

// WithReconnect 设置自动重连参数。
//   - retries: 最大连续重试次数（-1 表示无限重连；0 表示不启用重连；>0 表示最大次数）
//   - interval: 重试间隔；留空或负值表示保持默认值
func WithReconnect(retries int, interval ...time.Duration) Option {
	return func(client *Client) {
		client.maxReconnectRetries = retries
		if len(interval) > 0 && interval[0] >= 0 {
			client.reconnectRetryInterval = interval[0]
		}
	}
}

// WithAuth 覆盖默认的密码认证方式，可用于追加私钥、证书等认证方法。
func WithAuth(auth ...ssh.AuthMethod) Option {
	return func(client *Client) {
		if len(auth) > 0 {
			client.config.Auth = auth
		}
	}
}

// WithHostKeyCallback 设置主机密钥校验回调。
//
// 这是推荐的证书校验方式：可传入 knownhosts.New 的返回值，
// 或自定义回调实现固定指纹校验。
func WithHostKeyCallback(callback ssh.HostKeyCallback) Option {
	return func(client *Client) {
		if callback == nil {
			client.optionErr = wrapError(ErrInvalidOption, "host key callback must not be nil")
			return
		}

		client.config.HostKeyCallback = callback
	}
}

// WithKnownHosts 使用 OpenSSH known_hosts 文件校验主机密钥。
//
// 文件不存在或格式错误时，NewClient 会返回对应错误而不发起连接。
func WithKnownHosts(filename string) Option {
	return func(client *Client) {
		callback, err := knownhosts.New(filename)
		if err != nil {
			client.optionErr = wrapError(ErrInvalidOption, "load known_hosts %q: %w", filename, err)
			return
		}

		client.config.HostKeyCallback = callback
	}
}

// WithInsecureIgnoreHostKey 显式关闭主机密钥校验。
//
// 该选项仅用于受控网络或测试环境；生产环境请改用 WithKnownHosts
// 或 WithHostKeyCallback，否则连接可能被中间人劫持。
func WithInsecureIgnoreHostKey() Option {
	return func(client *Client) {
		client.config.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}
}
