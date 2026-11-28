package smtpx

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.nets.smtpx: "

var (
	// ErrEmptySubject 表示邮件主题为空
	ErrEmptySubject = errors.New(_ERROR_PREFIX + "empty subject")
	// ErrNoRecipient 表示邮件未填写主送收件人
	ErrNoRecipient = errors.New(_ERROR_PREFIX + "no primary recipients")
	// ErrNilMessage 表示传入的邮件对象为空
	ErrNilMessage = errors.New(_ERROR_PREFIX + "nil message")
	// ErrStartTLSFailed 表示 STARTTLS 升级失败
	ErrStartTLSFailed = errors.New(_ERROR_PREFIX + "failed to start TLS")
	// ErrStartTLSUnsupported 表示服务器不支持 STARTTLS
	ErrStartTLSUnsupported = errors.New(_ERROR_PREFIX + "server does not support STARTTLS")
	// ErrSetDeadline 表示设置连接超时失败
	ErrSetDeadline = errors.New(_ERROR_PREFIX + "failed to set deadline")
	// ErrConnectFailed 表示连接 SMTP 服务器失败
	ErrConnectFailed = errors.New(_ERROR_PREFIX + "failed to connect to SMTP server")
	// ErrNewClientFailed 表示创建 SMTP 客户端失败
	ErrNewClientFailed = errors.New(_ERROR_PREFIX + "failed to create SMTP client")
	// ErrAuthFailed 表示 SMTP 认证失败
	ErrAuthFailed = errors.New(_ERROR_PREFIX + "failed to authenticate")
	// ErrSenderRejected 表示服务器拒绝发件人地址
	ErrSenderRejected = errors.New(_ERROR_PREFIX + "sender address is rejected")
	// ErrRecipientRejected 表示服务器拒绝收件人地址
	ErrRecipientRejected = errors.New(_ERROR_PREFIX + "recipient address is rejected")
	// ErrDataFailed 表示无法开始传递邮件正文
	ErrDataFailed = errors.New(_ERROR_PREFIX + "failed to initiate data transfer")
	// ErrWriteBodyFailed 表示写入邮件正文失败
	ErrWriteBodyFailed = errors.New(_ERROR_PREFIX + "failed to write message body")
	// ErrFinishDataFailed 表示提交邮件正文失败
	ErrFinishDataFailed = errors.New(_ERROR_PREFIX + "failed to finish data transfer")
	// ErrHeaderInjection 表示邮件头字段含 CR/LF 或非法字符，存在头注入风险
	ErrHeaderInjection = errors.New(_ERROR_PREFIX + "header injection detected")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
