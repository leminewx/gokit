//go:build integration

// 外部服务依赖测试：需要真实 SMTP 账号才能发送邮件，默认不参与构建与测试。
//
// 执行前需设置以下环境变量，缺省时测试会被跳过：
//   - SMTPX_HOST：SMTP 服务器地址
//   - SMTPX_PORT：SMTP 服务器端口（缺省 465）
//   - SMTPX_USER：登录账号
//   - SMTPX_PASS：登录密码
//   - SMTPX_TO：收件人地址（缺省与 SMTPX_USER 相同）
//   - SMTPX_STARTTLS：设为 1 时走 STARTTLS（明文连接后升级），适用于 587 等端口
//
// 执行方式：go test -tags integration ./nets/smtpx

package smtpx

import (
	"os"
	"strconv"
	"testing"
)

// integrationClient 根据环境变量构建集成测试客户端，缺少必要配置时跳过测试。
func integrationClient(t *testing.T) (client *Client, user, pass, to string) {
	t.Helper()

	host := os.Getenv("SMTPX_HOST")
	user = os.Getenv("SMTPX_USER")
	pass = os.Getenv("SMTPX_PASS")
	if host == "" || user == "" || pass == "" {
		t.Skip("未配置 SMTPX_HOST / SMTPX_USER / SMTPX_PASS，跳过 SMTP 集成测试")
	}

	port := 465
	if raw := os.Getenv("SMTPX_PORT"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("SMTPX_PORT 不是有效端口: %v", err)
		}
		port = value
	}

	to = os.Getenv("SMTPX_TO")
	if to == "" {
		to = user
	}

	var options []Option
	if os.Getenv("SMTPX_STARTTLS") == "1" {
		options = append(options, WithStartTLS())
	}

	return NewClient(host, port, options...), user, pass, to
}

func TestClientSendText(t *testing.T) {
	client, user, pass, to := integrationClient(t)

	if err := client.SendText(user, pass, &Message{
		Subject: "gokit 集成测试 - 纯文本",
		To:      []string{to},
		Body:    []byte("这是一封由 gokit/nets/smtpx 集成测试发送的纯文本邮件。"),
	}); err != nil {
		t.Fatalf("发送纯文本邮件失败: %v", err)
	}
}

func TestClientSendHTML(t *testing.T) {
	client, user, pass, to := integrationClient(t)

	body := GenerateRegisterUserHTML("gokit", "support@example.com", "2026", to, "集成测试", "gokit-integration")
	if err := client.SendHTML(user, pass, &Message{
		Subject: "gokit 集成测试 - HTML",
		To:      []string{to},
		Body:    body,
	}); err != nil {
		t.Fatalf("发送 HTML 邮件失败: %v", err)
	}
}
