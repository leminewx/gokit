package smtpx

import (
	"errors"
	"testing"
)

func TestSendNilMessage(t *testing.T) {
	client := NewClient("smtp.example.com", 465)

	if err := client.SendText("a@example.com", "secret", nil); err == nil {
		t.Fatal("SendText(nil) 应返回错误")
	}
	if err := client.SendHTML("a@example.com", "secret", nil); err == nil {
		t.Fatal("SendHTML(nil) 应返回错误")
	}
}

// TestSendRejectsEnvelopeInjection 验证信封命令中的 CR/LF 会在拨号前被拒绝
//   - SendRaw 不经过 Message.Validate，必须在 send 内部做最后一道校验
func TestSendRejectsEnvelopeInjection(t *testing.T) {
	t.Parallel()

	client := NewClient("smtp.example.com", 465)
	tests := []struct {
		name       string
		sender     string
		recipients []string
	}{
		{
			name:       "收件人含 CRLF",
			sender:     "from@example.com",
			recipients: []string{"a@example.com\r\nRCPT TO:<evil@example.com>"},
		},
		{
			name:       "发件人含 CRLF",
			sender:     "from@example.com\r\nRCPT TO:<evil@example.com>",
			recipients: []string{"a@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := client.SendRaw(tt.sender, "secret", tt.recipients, []byte("body")); !errors.Is(err, ErrHeaderInjection) {
				t.Fatalf("SendRaw() = %v, want ErrHeaderInjection", err)
			}
		})
	}
}
