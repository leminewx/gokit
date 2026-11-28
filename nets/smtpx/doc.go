// Package smtpx 提供 SMTP 邮件发送：文本 / HTML 正文、附件、STARTTLS 与整体超时。
//
// 每次发送独立建连，客户端本身无内部可变状态，可并发调用；发件人、收件人与附件元信息
// 在发送前校验，检测到 CR/LF 注入返回 ErrHeaderInjection。超时同时覆盖拨号与整个会话。
//
// 本包不提供连接池与队列重试，批量发送由调用方按需控制并发与退避。
package smtpx
