package smtpx

import (
	"bytes"
	"errors"
	"io"
	"mime/quotedprintable"
	"net/textproto"
	"strings"
	"testing"
)

func TestMessageValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message Message
		wantErr bool
	}{
		{
			name:    "empty subject",
			message: Message{To: []string{"a@example.com"}},
			wantErr: true,
		},
		{
			name:    "no recipients",
			message: Message{Subject: "hi"},
			wantErr: true,
		},
		{
			name:    "valid",
			message: Message{Subject: "hi", To: []string{"a@example.com"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.message.Validate()
			if tt.wantErr && err == nil {
				t.Error("Validate 返回 nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate 返回错误: %v", err)
			}
		})
	}
}

// TestMessageValidateRejectsHeaderInjection 验证收件人与自定义头中的 CR/LF 被拒绝
//   - 回归点：这些值曾原样写入邮件头，可注入 Bcc/Content-Type，或向 SMTP 信封注入命令
func TestMessageValidateRejectsHeaderInjection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message Message
	}{
		{
			name:    "CRLF in recipient",
			message: Message{Subject: "hi", To: []string{"a@example.com\r\nBcc: evil@example.com"}},
		},
		{
			name:    "LF in cc",
			message: Message{Subject: "hi", To: []string{"a@example.com"}, Cc: []string{"b@example.com\nBcc: evil@example.com"}},
		},
		{
			name:    "CRLF in bcc",
			message: Message{Subject: "hi", To: []string{"a@example.com"}, Bcc: []string{"c@example.com\r\nRCPT TO:<evil@example.com>"}},
		},
		{
			name:    "CRLF in custom header value",
			message: Message{Subject: "hi", To: []string{"a@example.com"}, Headers: map[string]string{"X-Test": "v\r\nBcc: evil@example.com"}},
		},
		{
			name:    "colon in custom header name",
			message: Message{Subject: "hi", To: []string{"a@example.com"}, Headers: map[string]string{"X-Test: Bcc": "v"}},
		},
		{
			name:    "empty custom header name",
			message: Message{Subject: "hi", To: []string{"a@example.com"}, Headers: map[string]string{"": "v"}},
		},
		{
			name: "CRLF in attachment content type",
			message: Message{Subject: "hi", To: []string{"a@example.com"}, Attachments: []Attachment{
				{Filename: "a.txt", ContentType: "text/plain\r\nBcc: evil@example.com", Data: []byte("x")},
			}},
		},
		{
			name: "CRLF in attachment filename",
			message: Message{Subject: "hi", To: []string{"a@example.com"}, Attachments: []Attachment{
				{Filename: "a\r\nb.txt", Data: []byte("x")},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.message.Validate(); !errors.Is(err, ErrHeaderInjection) {
				t.Fatalf("Validate() = %v, want ErrHeaderInjection", err)
			}
			if _, err := tt.message.GenerateText("from@example.com"); !errors.Is(err, ErrHeaderInjection) {
				t.Fatalf("GenerateText() = %v, want ErrHeaderInjection", err)
			}
		})
	}
}

// TestGenerateRejectsHeaderInjectionInFrom 验证发件人地址中的 CR/LF 被拒绝
//   - From 不经过 Validate（由 Generate* 的入参传入），需单独校验
func TestGenerateRejectsHeaderInjectionInFrom(t *testing.T) {
	t.Parallel()

	message := Message{Subject: "hi", To: []string{"a@example.com"}, Body: []byte("body")}
	for name, generate := range map[string]func(string) ([]byte, error){
		"text": message.GenerateText,
		"html": message.GenerateHTML,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := generate("from@example.com\r\nBcc: evil@example.com"); !errors.Is(err, ErrHeaderInjection) {
				t.Fatalf("生成 %s 邮件 = %v, want ErrHeaderInjection", name, err)
			}
		})
	}
}

// TestWriteHeadersStripsCRLF 验证 writeHeaders 作为最后一道防护会剔除 CR/LF
func TestWriteHeadersStripsCRLF(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	headers := textproto.MIMEHeader{
		"To": {"a@example.com\r\nBcc: evil@example.com"},
	}
	writeHeaders(&buffer, headers, map[string]string{"X-Test\r\nBcc": "v\r\nBcc: evil@example.com"})

	output := buffer.String()
	if strings.Contains(output, "\r\nBcc:") {
		t.Fatalf("输出包含被注入的头: %q", output)
	}
	// 只应用显式的 \r\n 结束行，共 2 行（To 与 X-Test）
	if got := strings.Count(output, "\r\n"); got != 2 {
		t.Fatalf("输出行数 = %d, want 2: %q", got, output)
	}
}

func TestMessageRecipientsDeduplicated(t *testing.T) {
	t.Parallel()

	message := &Message{
		Subject: "hi",
		To:      []string{"a@example.com", "b@example.com"},
		Cc:      []string{"b@example.com", "c@example.com"},
		Bcc:     []string{"c@example.com", "d@example.com"},
	}
	if err := message.Validate(); err != nil {
		t.Fatalf("Validate 返回错误: %v", err)
	}

	recipients := message.GetRecipients()
	if len(recipients) != 4 {
		t.Fatalf("收件人 = %v, want 4 个去重地址", recipients)
	}

	seen := make(map[string]bool, len(recipients))
	for _, addr := range recipients {
		if seen[addr] {
			t.Errorf("收件人重复：%s", addr)
		}
		seen[addr] = true
	}
}

func TestQuotedPrintable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{name: "printable", in: []byte("hello world"), want: "hello world"},
		{name: "equals", in: []byte("a=b"), want: "a=3Db"},
		{name: "crlf", in: []byte("a\r\nb"), want: "a=0D=0Ab"},
		{name: "utf8", in: []byte("中"), want: "=E4=B8=AD"},
		{name: "tab and space", in: []byte("\t "), want: "\t "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := string(EncodeQuotedPrintable(tt.in)); got != tt.want {
				t.Errorf("EncodeQuotedPrintable(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestGenerateText(t *testing.T) {
	t.Parallel()

	message := &Message{
		Subject: "hello",
		To:      []string{"a@example.com"},
		Cc:      []string{"b@example.com"},
		Body:    []byte("中文正文"),
	}
	data, err := message.GenerateText("sender@example.com")
	if err != nil {
		t.Fatalf("GenerateText 返回错误: %v", err)
	}

	content := string(data)
	for _, want := range []string{
		"From: sender@example.com",
		"To: a@example.com",
		"Cc: b@example.com",
		"Subject: hello",
		"Mime-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("邮件缺少头部 %q", want)
		}
	}

	// 正文必须为 quoted-printable 编码后的字节
	if !strings.Contains(content, string(EncodeQuotedPrintable(message.Body))) {
		t.Error("邮件正文未按 quoted-printable 编码")
	}
}

func TestGenerateHTML(t *testing.T) {
	t.Parallel()

	message := &Message{
		Subject: "hello",
		To:      []string{"a@example.com"},
		Body:    []byte("<p>hi</p>"),
	}
	data, err := message.GenerateHTML("sender@example.com")
	if err != nil {
		t.Fatalf("GenerateHTML 返回错误: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Content-Type: multipart/alternative; boundary=\"") {
		t.Errorf("HTML 邮件缺少 multipart 头：%s", content)
	}
	if !strings.Contains(content, "Content-Type: text/html; charset=UTF-8") {
		t.Error("HTML 邮件缺少 text/html 部分")
	}
	if !strings.Contains(content, "<p>hi</p>") {
		t.Error("HTML 邮件缺少正文")
	}
}

func TestGenerateRejectsInvalidMessage(t *testing.T) {
	t.Parallel()

	message := &Message{Subject: "", To: []string{"a@example.com"}}
	if _, err := message.GenerateText("sender@example.com"); err == nil {
		t.Error("GenerateText 未校验邮件, want error")
	}
	if _, err := message.GenerateHTML("sender@example.com"); err == nil {
		t.Error("GenerateHTML 未校验邮件, want error")
	}
}

// 由 message_extended_test.go 迁入
func TestGenerateTextEncodesSubject(t *testing.T) {
	message := &Message{
		Subject: "注册成功",
		To:      []string{"a@example.com"},
		Body:    []byte("欢迎使用 gokit"),
	}

	data, err := message.GenerateText("sender@example.com")
	if err != nil {
		t.Fatalf("GenerateText error: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Subject: =?utf-8?") {
		t.Fatalf("主题未按 RFC 2047 编码: %s", content)
	}
	if !strings.Contains(content, "Message-ID: <") {
		t.Fatalf("缺少 Message-ID: %s", content)
	}
	if !strings.Contains(content, "Date: ") {
		t.Fatalf("缺少 Date: %s", content)
	}

	// 正文应为可逆的 quoted-printable 编码
	body := partBody(content)
	decoded, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(body)))
	if err != nil {
		t.Fatalf("解码正文失败: %v", err)
	}
	if string(decoded) != "欢迎使用 gokit" {
		t.Fatalf("正文解码 = %q", decoded)
	}
}

// 由 message_extended_test.go 迁入
func TestGenerateTextWithAttachment(t *testing.T) {
	message := &Message{
		Subject: "report",
		To:      []string{"a@example.com"},
		Body:    []byte("see attachment"),
		Attachments: []Attachment{
			{Filename: "report.txt", ContentType: "text/plain", Data: []byte("hello attachment")},
		},
	}

	data, err := message.GenerateText("sender@example.com")
	if err != nil {
		t.Fatalf("GenerateText error: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Content-Type: multipart/mixed; boundary=\"") {
		t.Fatalf("缺少 multipart/mixed 头: %s", content)
	}
	if !strings.Contains(content, "report.txt") {
		t.Fatalf("缺少附件文件名: %s", content)
	}
	if !strings.Contains(content, "Content-Transfer-Encoding: base64") {
		t.Fatalf("附件未使用 base64: %s", content)
	}
	if !strings.Contains(content, "aGVsbG8gYXR0YWNobWVudA==") {
		t.Fatalf("附件内容 base64 不正确: %s", content)
	}
}

// 由 message_extended_test.go 迁入
func TestGenerateHTMLWithAttachment(t *testing.T) {
	message := &Message{
		Subject: "hello",
		To:      []string{"a@example.com"},
		Body:    []byte("<p>hi</p>"),
		Attachments: []Attachment{
			{Filename: "logo.bin", Data: []byte{0x00, 0x01, 0x02}},
		},
	}

	data, err := message.GenerateHTML("sender@example.com")
	if err != nil {
		t.Fatalf("GenerateHTML error: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "multipart/mixed") {
		t.Fatalf("缺少外层 multipart/mixed: %s", content)
	}
	if !strings.Contains(content, "multipart/alternative") {
		t.Fatalf("缺少内层 multipart/alternative: %s", content)
	}
	if !strings.Contains(content, "application/octet-stream") {
		t.Fatalf("附件默认类型不正确: %s", content)
	}
	if !strings.Contains(content, "logo.bin") {
		t.Fatalf("缺少附件名: %s", content)
	}
}

// 由 message_extended_test.go 迁入
func TestGetRecipientsWithoutValidate(t *testing.T) {
	message := &Message{
		To:  []string{"a@example.com", "b@example.com"},
		Cc:  []string{"b@example.com"},
		Bcc: []string{"c@example.com"},
	}

	recipients := message.GetRecipients()
	if len(recipients) != 3 {
		t.Fatalf("GetRecipients = %v, want 3 个去重地址", recipients)
	}
	if recipients[0] != "a@example.com" || recipients[2] != "c@example.com" {
		t.Fatalf("GetRecipients 顺序不正确: %v", recipients)
	}
}

// 由 message_extended_test.go 迁入
// partBody 提取首个空行之后的正文，便于解码校验
func partBody(content string) string {
	_, body, ok := strings.Cut(content, "\r\n\r\n")
	if !ok {
		return ""
	}

	return strings.TrimSuffix(body, "\r\n")
}
