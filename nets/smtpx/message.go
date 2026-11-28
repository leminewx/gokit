package smtpx

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"sort"
	"strings"
	"time"
)

// _ALTERNATIVE_BOUNDARY 含附件时内层 multipart/alternative 的边界
const _ALTERNATIVE_BOUNDARY = "----=_Alternative_"

// Attachment 描述一个邮件附件
type Attachment struct {
	Filename    string // 附件文件名
	ContentType string // MIME 类型，空值按 application/octet-stream 处理
	Data        []byte // 附件内容
}

// Message 描述一封待发送的邮件
type Message struct {
	Subject     string            // 主题
	To          []string          // 主送
	Cc          []string          // 抄送
	Bcc         []string          // 密送
	Body        []byte            // 内容（GenerateText 为纯文本，GenerateHTML 为 HTML）
	Attachments []Attachment      // 附件，非空时使用 multipart/mixed
	Headers     map[string]string // 自定义头，会追加到标准头之后
}

// GetRecipients 获取所有收件人（主送+抄送+密送，已去重且保持原顺序）
func (own *Message) GetRecipients() []string {
	all := make([]string, 0, len(own.To)+len(own.Cc)+len(own.Bcc))
	all = append(all, own.To...)
	all = append(all, own.Cc...)
	all = append(all, own.Bcc...)

	seen := make(map[string]struct{}, len(all))
	recipients := make([]string, 0, len(all))
	for _, addr := range all {
		if _, ok := seen[addr]; ok {
			continue
		}

		seen[addr] = struct{}{}
		recipients = append(recipients, addr)
	}

	return recipients
}

// GenerateText 构建纯文本邮件
//   - 无附件时为单 part text/plain；有附件时为 multipart/mixed
//   - 主题按 RFC 2047 编码，正文按 quoted-printable 编码
func (own *Message) GenerateText(from string) ([]byte, error) {
	if err := own.Validate(); err != nil {
		return nil, err
	}
	if err := checkHeaderValue("from", from); err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	headers := own.newHeaders(from)
	if len(own.Attachments) == 0 {
		headers.Set("Content-Type", "text/plain; charset=utf-8")
		headers.Set("Content-Transfer-Encoding", "quoted-printable")
		writeHeaders(&buffer, headers, own.Headers)
		buffer.WriteString("\r\n")

		if err := writeQuotedPrintable(&buffer, own.Body); err != nil {
			return nil, wrapError(ErrWriteBodyFailed, "text part: %w", err)
		}

		return buffer.Bytes(), nil
	}

	writer := multipart.NewWriter(&buffer)
	headers.Set("Content-Type", "multipart/mixed; boundary=\""+writer.Boundary()+"\"")
	writeHeaders(&buffer, headers, own.Headers)
	buffer.WriteString("\r\n")

	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "text/plain; charset=utf-8")
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, wrapError(ErrWriteBodyFailed, "text part: %w", err)
	}
	if err := writeQuotedPrintable(part, own.Body); err != nil {
		return nil, wrapError(ErrWriteBodyFailed, "text part: %w", err)
	}

	if err := own.writeAttachments(writer); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, wrapError(ErrWriteBodyFailed, "close multipart: %w", err)
	}

	return buffer.Bytes(), nil
}

// GenerateHTML 构建 HTML 邮件
//   - 无附件时为 multipart/alternative；有附件时外层再包一层 multipart/mixed
func (own *Message) GenerateHTML(from string) ([]byte, error) {
	if err := own.Validate(); err != nil {
		return nil, err
	}
	if err := checkHeaderValue("from", from); err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	headers := own.newHeaders(from)
	writer := multipart.NewWriter(&buffer)
	if len(own.Attachments) == 0 {
		headers.Set("Content-Type", "multipart/alternative; boundary=\""+writer.Boundary()+"\"")
		writeHeaders(&buffer, headers, own.Headers)
		buffer.WriteString("\r\n")

		if err := writeHTMLPart(writer, own.Body); err != nil {
			return nil, err
		}
		if err := writer.Close(); err != nil {
			return nil, wrapError(ErrWriteBodyFailed, "close multipart: %w", err)
		}

		return buffer.Bytes(), nil
	}

	headers.Set("Content-Type", "multipart/mixed; boundary=\""+writer.Boundary()+"\"")
	writeHeaders(&buffer, headers, own.Headers)
	buffer.WriteString("\r\n")

	alternative, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"multipart/alternative; boundary=\"" + _ALTERNATIVE_BOUNDARY + "\""},
	})
	if err != nil {
		return nil, wrapError(ErrWriteBodyFailed, "alternative part: %w", err)
	}

	alternativeWriter := multipart.NewWriter(alternative)
	if err := alternativeWriter.SetBoundary(_ALTERNATIVE_BOUNDARY); err != nil {
		return nil, wrapError(ErrWriteBodyFailed, "set boundary: %w", err)
	}
	if err := writeHTMLPart(alternativeWriter, own.Body); err != nil {
		return nil, err
	}
	if err := alternativeWriter.Close(); err != nil {
		return nil, wrapError(ErrWriteBodyFailed, "close alternative: %w", err)
	}

	if err := own.writeAttachments(writer); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, wrapError(ErrWriteBodyFailed, "close multipart: %w", err)
	}

	return buffer.Bytes(), nil
}

// Validate 校验邮件必填项
//   - 收件人（含抄送/密送）与自定义头会以 "Key: value\r\n" 形式写入邮件，
//     含 CR/LF 即构成头注入（可伪造 Bcc、Content-Type 等），因此一并拒绝
//   - Subject 已由 RFC 2047 编码，不会产生头注入
func (own *Message) Validate() error {
	if own.Subject == "" {
		return ErrEmptySubject
	}
	if len(own.To) == 0 {
		return ErrNoRecipient
	}

	for _, addr := range own.GetRecipients() {
		if err := checkHeaderValue("recipient", addr); err != nil {
			return err
		}
	}

	for key, value := range own.Headers {
		if err := checkHeaderName(key); err != nil {
			return err
		}
		if err := checkHeaderValue("header "+key, value); err != nil {
			return err
		}
	}

	for _, attachment := range own.Attachments {
		// Content-Type 会被原样写入 part 头；文件名由 mime.FormatMediaType 转义
		if err := checkHeaderValue("attachment content type", attachment.ContentType); err != nil {
			return err
		}
		if err := checkHeaderValue("attachment filename", attachment.Filename); err != nil {
			return err
		}
	}

	return nil
}

// checkHeaderValue 校验准备写入邮件头的值不含 CR/LF
func checkHeaderValue(field, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return wrapError(ErrHeaderInjection, "%s contains CR/LF: %q", field, value)
	}

	return nil
}

// checkHeaderName 校验自定义头名称：必须是可打印 ASCII 且不含冒号与空白
func checkHeaderName(name string) error {
	if name == "" {
		return wrapError(ErrHeaderInjection, "empty header name")
	}

	for _, r := range name {
		if r <= ' ' || r > '~' || r == ':' {
			return wrapError(ErrHeaderInjection, "invalid header name: %q", name)
		}
	}

	return nil
}

// newHeaders 构造标准头（主题按 RFC 2047 编码）
func (own *Message) newHeaders(from string) textproto.MIMEHeader {
	headers := make(textproto.MIMEHeader)
	headers.Set("From", from)
	headers.Set("To", strings.Join(own.To, ", "))
	if len(own.Cc) > 0 {
		headers.Set("Cc", strings.Join(own.Cc, ", "))
	}

	headers.Set("Subject", mime.QEncoding.Encode("utf-8", own.Subject))
	headers.Set("Date", time.Now().UTC().Format(time.RFC1123Z))
	headers.Set("Mime-Version", "1.0")
	headers.Set("Message-ID", own.messageID(from))

	return headers
}

// messageID 生成带发件域名的唯一 Message-ID
func (own *Message) messageID(from string) string {
	random := make([]byte, 12)
	_, _ = rand.Read(random)

	domain := "localhost"
	if at := strings.LastIndex(from, "@"); at >= 0 && at < len(from)-1 {
		domain = from[at+1:]
	}

	return "<" + hex.EncodeToString(random) + "@" + domain + ">"
}

// writeAttachments 写入所有附件 part
func (own *Message) writeAttachments(writer *multipart.Writer) error {
	for _, attachment := range own.Attachments {
		contentType := attachment.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		header := make(textproto.MIMEHeader)
		header.Set("Content-Type", contentType)
		header.Set("Content-Transfer-Encoding", "base64")
		header.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
			"filename": attachment.Filename,
		}))

		part, err := writer.CreatePart(header)
		if err != nil {
			return wrapError(ErrWriteBodyFailed, "attachment %q: %w", attachment.Filename, err)
		}

		if err := writeBase64(part, attachment.Data); err != nil {
			return wrapError(ErrWriteBodyFailed, "attachment %q: %w", attachment.Filename, err)
		}
	}

	return nil
}

// writeHeaders 按固定顺序写入邮件头，保证输出稳定
//   - 写入前统一剔除 CR/LF，作为校验之外的最后一道防护
func writeHeaders(output io.Writer, headers textproto.MIMEHeader, extra map[string]string) {
	ordered := []string{"From", "To", "Cc", "Subject", "Date", "Mime-Version", "Message-ID", "Content-Type", "Content-Transfer-Encoding"}
	for _, key := range ordered {
		value := headers.Get(key)
		if value == "" {
			continue
		}
		_, _ = fmt.Fprintf(output, "%s: %s\r\n", key, stripCRLF(value))
	}

	keys := make([]string, 0, len(extra))
	for key := range extra {
		if headers.Get(key) != "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		_, _ = fmt.Fprintf(output, "%s: %s\r\n", stripCRLF(key), stripCRLF(extra[key]))
	}
}

// stripCRLF 剔除 CR/LF，避免邮件头注入
var _CRLF_STRIPPER = strings.NewReplacer("\r", "", "\n", "")

func stripCRLF(s string) string {
	return _CRLF_STRIPPER.Replace(s)
}

// writeHTMLPart 写入 multipart/alternative 的 HTML 部分
func writeHTMLPart(writer *multipart.Writer, body []byte) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "text/html; charset=UTF-8")
	header.Set("Content-Transfer-Encoding", "quoted-printable")

	part, err := writer.CreatePart(header)
	if err != nil {
		return wrapError(ErrWriteBodyFailed, "html part: %w", err)
	}
	if err := writeQuotedPrintable(part, body); err != nil {
		return wrapError(ErrWriteBodyFailed, "html part: %w", err)
	}

	return nil
}

// writeQuotedPrintable 以 quoted-printable 编码写入正文（含 76 列软换行）
func writeQuotedPrintable(output io.Writer, body []byte) error {
	writer := quotedprintable.NewWriter(output)
	if _, err := writer.Write(body); err != nil {
		return err
	}

	return writer.Close()
}

// writeBase64 以每行 76 字符的 base64 写入附件内容
func writeBase64(output io.Writer, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 76 {
		if _, err := io.WriteString(output, encoded[:76]+"\r\n"); err != nil {
			return err
		}
		encoded = encoded[76:]
	}

	if encoded != "" {
		if _, err := io.WriteString(output, encoded+"\r\n"); err != nil {
			return err
		}
	}

	return nil
}

// EncodeQuotedPrintable 将数据编码为 quoted-printable 格式
//   - 保留字节级可逆语义（不插入软换行），与邮件正文的流式编码器相互独立
func EncodeQuotedPrintable(data []byte) []byte {
	var buffer bytes.Buffer
	for _, b := range data {
		// 可打印字符且不为 '=' 直接写入
		if (b >= 33 && b <= 60) || (b >= 62 && b <= 126) || b == 9 || b == 32 {
			buffer.WriteByte(b)
			continue
		}

		// 其他字符编码为 =XX
		_, _ = fmt.Fprintf(&buffer, "=%02X", b)
	}

	return buffer.Bytes()
}
