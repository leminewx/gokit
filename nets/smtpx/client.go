package smtpx

import (
	"crypto/tls"
	"net"
	"net/smtp"
	"strconv"
	"time"
)

// _DEFAULT_DIAL_TIMEOUT 默认建立连接（含 TLS 握手）的超时时间
const _DEFAULT_DIAL_TIMEOUT = 10 * time.Second

// Client 定义 SMTP 客户端
type Client struct {
	addr        string
	serverName  string
	tlsConfig   *tls.Config
	dialTimeout time.Duration
	timeout     time.Duration
	startTLS    bool
}

// NewClient 新建一个 SMTP 客户端
//   - smtpServer: SMTP 服务器地址（如 "smtp.gmail.com"，IPv6 字面量可直接传 "::1"）
//   - smtpPort: SMTP 端口（465 使用隐式 TLS，587 建议配合 WithStartTLS）
//   - 默认拨号超时 10s，默认不限制整体发送时长
func NewClient(smtpServer string, smtpPort int, options ...Option) *Client {
	own := &Client{
		addr:        net.JoinHostPort(smtpServer, strconv.Itoa(smtpPort)),
		serverName:  smtpServer,
		tlsConfig:   &tls.Config{ServerName: smtpServer},
		dialTimeout: _DEFAULT_DIAL_TIMEOUT,
	}

	for _, option := range options {
		option(own)
	}

	if own.tlsConfig.ServerName == "" {
		own.tlsConfig.ServerName = own.serverName
	}

	return own
}

// GetAddr 返回 SMTP 服务器地址，格式为 host:port（IPv6 字面量带方括号，如 "[::1]:587"）
func (own *Client) GetAddr() string {
	return own.addr
}

// SendText 发送纯文本邮件
//   - username: 发件人邮箱（同时也是 SMTP 认证用户名），为空时跳过认证
//   - password: 发件人邮箱密码或授权码
//   - msg: 邮件信息，需包含主题、收件人、正文等
func (own *Client) SendText(username, password string, msg *Message) error {
	if msg == nil {
		return wrapError(ErrNilMessage, "text")
	}

	message, err := msg.GenerateText(username)
	if err != nil {
		return err
	}

	return own.send(username, password, msg.GetRecipients(), message)
}

// SendHTML 发送 HTML 邮件
//   - username: 发件人邮箱（同时也是 SMTP 认证用户名），为空时跳过认证
//   - password: 发件人邮箱密码或授权码
//   - msg: 邮件信息，需包含主题、收件人、正文等
func (own *Client) SendHTML(username, password string, msg *Message) error {
	if msg == nil {
		return wrapError(ErrNilMessage, "html")
	}

	message, err := msg.GenerateHTML(username)
	if err != nil {
		return err
	}

	return own.send(username, password, msg.GetRecipients(), message)
}

// SendRaw 发送已按 MIME 规范组装好的原始邮件
//   - recipients: 信封收件人列表（可包含抄送与密送，需自行去重）
//   - message: 完整邮件内容，必须包含头部与空行分隔的正文
func (own *Client) SendRaw(username, password string, recipients []string, message []byte) error {
	return own.send(username, password, recipients, message)
}

// send 建立连接并投递邮件
func (own *Client) send(username, password string, recipients []string, message []byte) error {
	if len(recipients) == 0 {
		return ErrNoRecipient
	}

	// 信封命令（MAIL FROM / RCPT TO）同样以行文本发送，
	// SendRaw 不经过 Message.Validate，因此这里统一再校验一次
	if err := checkHeaderValue("sender", username); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if err := checkHeaderValue("recipient", recipient); err != nil {
			return err
		}
	}

	conn, err := own.dial()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	// 整体超时通过连接截止时间实现，对 TLS 与 SMTP 会话同样生效
	if own.timeout > 0 {
		if err := conn.SetDeadline(time.Now().Add(own.timeout)); err != nil {
			return wrapError(ErrSetDeadline, "%w", err)
		}
	}

	client, err := smtp.NewClient(conn, own.serverName)
	if err != nil {
		return wrapError(ErrNewClientFailed, "%s: %w", own.serverName, err)
	}
	defer func() { _ = client.Close() }()

	// STARTTLS 模式：明文连接建立后升级加密
	if own.startTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return wrapError(ErrStartTLSUnsupported, "%s", own.addr)
		}
		if err := client.StartTLS(own.tlsConfig.Clone()); err != nil {
			return wrapError(ErrStartTLSFailed, "%s: %w", own.addr, err)
		}
	}

	// 认证：用户名与密码均为空时跳过（适用于免认证的中继服务）
	if username != "" || password != "" {
		auth := smtp.PlainAuth("", username, password, own.serverName)
		if err := client.Auth(auth); err != nil {
			return wrapError(ErrAuthFailed, "%q: %w", username, err)
		}
	}

	// 设置发件人
	if err := client.Mail(username); err != nil {
		return wrapError(ErrSenderRejected, "%q: %w", username, err)
	}

	// 设置所有收件人（包括主送、抄送和密送）
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return wrapError(ErrRecipientRejected, "%q: %w", recipient, err)
		}
	}

	// 创建数据写入器
	writer, err := client.Data()
	if err != nil {
		return wrapError(ErrDataFailed, "%w", err)
	}

	// 写入邮件内容
	if _, err := writer.Write(message); err != nil {
		return wrapError(ErrWriteBodyFailed, "%w", err)
	}

	// 关闭写入器，提交邮件
	if err := writer.Close(); err != nil {
		return wrapError(ErrFinishDataFailed, "%w", err)
	}

	return nil
}

// dial 建立到 SMTP 服务器的连接
//   - 隐式 TLS 模式直接完成握手；STARTTLS 模式返回明文连接
func (own *Client) dial() (net.Conn, error) {
	dialer := &net.Dialer{Timeout: own.dialTimeout}
	if own.startTLS {
		conn, err := dialer.Dial("tcp", own.addr)
		if err != nil {
			return nil, wrapError(ErrConnectFailed, "%s: %w", own.addr, err)
		}

		return conn, nil
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", own.addr, own.tlsConfig.Clone())
	if err != nil {
		return nil, wrapError(ErrConnectFailed, "%s: %w", own.addr, err)
	}

	return conn, nil
}
