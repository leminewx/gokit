package sshx

import (
	"bytes"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
)

// captureLog 临时把标准 logger 的输出重定向到缓冲区，用例结束后恢复
//   - 会修改全局 logger，因此调用方不能使用 t.Parallel
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	flags := log.Flags()
	log.SetFlags(0)
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(flags)
	})

	return &buf
}

func TestClientIsClosed(t *testing.T) {
	client := newOptionTestClient()
	if client.IsClosed() {
		t.Fatal("新建的客户端不应处于关闭状态")
	}

	client.closed.Store(true)
	if !client.IsClosed() {
		t.Fatal("closed 置位后 IsClosed 应返回 true")
	}
}

func TestClientDisableDebug(t *testing.T) {
	client := newOptionTestClient()
	client.debug.Store(true)

	client.DisableDebug()
	if client.debug.Load() {
		t.Fatal("DisableDebug 未关闭调试模式")
	}
}

func TestClientGetClientNilBeforeConnect(t *testing.T) {
	client := newOptionTestClient()
	if client.GetClient() != nil {
		t.Fatal("尚未建立连接时 GetClient 应返回 nil")
	}
}

func TestClientRunCommandAfterClose(t *testing.T) {
	client := newOptionTestClient()
	client.closed.Store(true)

	if _, err := client.RunCommand("echo hi"); !errors.Is(err, ErrClosed) {
		t.Fatalf("关闭后 RunCommand() error = %v, want ErrClosed", err)
	}
}

// TestDebugRedactsInputArgs 验证调试日志只记录输入长度，不泄露输入内容（可能含口令）
func TestDebugRedactsInputArgs(t *testing.T) {
	buf := captureLog(t)

	client := newOptionTestClient()
	client.debug.Store(true)
	client._debug("Input arguments >>> [REDACTED, %d bytes]", len("s3cr3t-p@ss"))

	out := buf.String()
	if strings.Contains(out, "s3cr3t") {
		t.Fatalf("调试日志泄露了输入内容: %q", out)
	}
	if !strings.Contains(out, "[REDACTED, 11 bytes]") {
		t.Fatalf("调试日志缺少脱敏占位: %q", out)
	}
}

// TestDebugTruncateAndNonPositiveSize 验证截断逻辑与非法长度不再越界
//   - size <= 0 视为不限制；size 为负数时旧实现会执行 info[:负数] 直接 panic
func TestDebugTruncateAndNonPositiveSize(t *testing.T) {
	buf := captureLog(t)

	client := newOptionTestClient()
	client.debug.Store(true)

	// 负数：应输出完整信息而不是 panic
	client.debugSize.Store(-1)
	client._debug("negative size, host=%s", "h1")
	if strings.Contains(buf.String(), "......") {
		t.Fatalf("非法长度不应触发截断: %q", buf.String())
	}

	// 正数且小于总长：应截断并追加省略号
	buf.Reset()
	client.debugSize.Store(6)
	client._debug("0123456789")
	if got, want := buf.String(), "[DEBUG......\n"; got != want {
		t.Fatalf("截断结果 = %q, want %q", got, want)
	}

	// 长度等于总长：应输出全文
	buf.Reset()
	full := "[DEBUG] [SSH] [127.0.0.1:22] exact"
	client.debugSize.Store(int64(len(full)))
	client._debug("exact")
	if got := buf.String(); got != full+"\n" {
		t.Fatalf("长度等于总长时应输出全文, got %q", got)
	}
}

// TestDebugDisabled 验证关闭调试后不产生任何输出
func TestDebugDisabled(t *testing.T) {
	buf := captureLog(t)

	client := newOptionTestClient()
	client._debug("should be silent")

	if buf.Len() != 0 {
		t.Fatalf("调试关闭时不应输出: %q", buf.String())
	}
}
