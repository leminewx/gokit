package smtpx

import (
	"strings"
	"testing"
)

// TestHTMLHeadTemplate 验证公共邮件头模板包含移动端与邮件客户端兼容声明
func TestHTMLHeadTemplate(t *testing.T) {
	for _, want := range []string{
		`http-equiv="X-UA-Compatible"`,
		`name="viewport"`,
		`http-equiv="Content-Type"`,
		`@media only screen`,
	} {
		if !strings.Contains(HTML_HEAD_TEMPLATE, want) {
			t.Fatalf("HTML_HEAD_TEMPLATE 缺少 %q", want)
		}
	}
}

func TestGenerateRegisterUserHTML(t *testing.T) {
	const (
		appName       = "DemoApp"
		appEmail      = "support@example.com"
		copyrightDate = "2026"
		email         = "user@example.com"
		username      = "张三"
		password      = "p@ssw0rd"
	)

	html := string(GenerateRegisterUserHTML(appName, appEmail, copyrightDate, email, username, password))

	tests := []struct {
		name string
		want string
	}{
		{"文档类型", "<!DOCTYPE html>"},
		{"语言", `lang="zh-CN"`},
		{"标题含应用名", appName + " 账号注册成功"},
		{"用户名", username},
		{"注册邮箱", email},
		{"初始密码", password},
		{"版权日期", copyrightDate},
		{"帮助邮箱", "mailto:" + appEmail},
		{"公共头部模板", `name="viewport"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(html, tt.want) {
				t.Fatalf("注册邮件 HTML 缺少 %q", tt.want)
			}
		})
	}
}

func TestGenerateResetPasswordHTML(t *testing.T) {
	const (
		appName       = "DemoApp"
		appEmail      = "support@example.com"
		copyrightDate = "2026"
		email         = "user@example.com"
		username      = "张三"
		password      = "new-p@ssw0rd"
	)

	html := string(GenerateResetPasswordHTML(appName, appEmail, copyrightDate, email, username, password))

	tests := []struct {
		name string
		want string
	}{
		{"文档类型", "<!DOCTYPE html>"},
		{"标题含应用名", appName + " 重置密码成功"},
		{"用户名", username},
		{"注册邮箱", email},
		{"新密码", password},
		{"版权日期", copyrightDate},
		{"帮助邮箱", "mailto:" + appEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(html, tt.want) {
				t.Fatalf("重置密码邮件 HTML 缺少 %q", tt.want)
			}
		})
	}
}

func TestGenerateHTMLNotShared(t *testing.T) {
	first := GenerateRegisterUserHTML("a", "b", "c", "d", "e", "f")
	second := GenerateRegisterUserHTML("a", "b", "c", "d", "e", "f")

	if len(first) == 0 || len(second) == 0 {
		t.Fatal("生成的 HTML 不应为空")
	}

	// 修改其中一份，另一份不应被连带修改，否则说明共用同一块底层数组
	first[0] = 'X'
	if second[0] == 'X' {
		t.Fatal("每次调用应返回独立的字节切片，实际共用了底层数组")
	}
}
