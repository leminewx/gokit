//go:build integration

// 外部服务依赖测试：需要可登录的真实 SSH 服务器才能运行，默认不参与构建与测试。
//
// 执行方式：go test -tags integration ./nets/sshx

package sshx

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// ------------------ 块式接口测试 ------------------

func TestExecute_Simple(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	if _, err := client.Execute(&Command{
		Command: "top -b -n 1",
	}); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestExecute_WithInput(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	out, err := client.Execute(&Command{
		Command:       "cat",
		InputArgs:     []string{"line1", "line2"},
		InputInterval: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	expected := "line1\r\nline2\r\n"
	if string(out) != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

func TestExecute_Timeout(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	_, err := client.Execute(&Command{
		Command: "sleep 10",
		Timeout: 3 * time.Second,
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestExecute_Cancel(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()

	_, err := client.ExecuteContext(ctx, &Command{
		Command: "sleep 5",
	})
	if err == nil {
		t.Fatal("expected cancel error, got nil")
	}
	if !strings.Contains(err.Error(), "Process exited with status 129 from signal HUP") {
		t.Errorf("expected 'Process exited with status 129 from signal HUP', got '%v'", err)
	}
}

func TestExecute_ErrorCommand(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	_, err := client.Execute(&Command{
		Command: "false", // 退出码 1
	})
	if err == nil {
		t.Fatal("expected error from false command")
	}
	if !strings.Contains(err.Error(), "Process exited with status 1") {
		t.Errorf("expected exit code 1, got %v", err)
	}
}

func TestExecute_ExitCodeFilter(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	out, err := client.Execute(&Command{
		Command:        "false",
		ExitCodeFilter: func(code int) bool { return code == 1 }, // 忽略退出码 1
	})
	if err != nil {
		t.Fatalf("Execute failed with filter: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestExecute_StderrFilter(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	out, err := client.Execute(&Command{
		Command:      "sh -c 'echo error >&2'",                // 输出到 stderr，但退出码 0
		StderrFilter: func(stderr error) bool { return true }, // 忽略 stderr 输出（即使命令成功但有 stderr）
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(out) == 0 {
		t.Errorf("expected stdout 'error\r\n', got '%q'", out)
	}
}

func TestClient_Execute(t *testing.T) {
	client, err := NewClient(testHost, testUser, testPass, WithDebug())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	if _, err := client.Execute(&Command{
		Command: "ls -la /system/etc",
	}); err != nil {
		t.Error(err)
	}

	if _, err := client.Execute(&Command{
		Command: "ip a",
	}); err != nil {
		t.Error(err)
	}

	if _, err := client.Execute(&Command{
		Command: "ps | grep omCore",
	}); err != nil {
		t.Error(err)
	}

	if _, err := client.Execute(&Command{
		Command: "top -n 1 | grep omCore",
	}); err != nil {
		t.Error(err)
	}

	if res, err := client.Execute(&Command{
		Command: "killall -9 tester",
	}); err == nil {
		t.Error(res)
	}

	if _, err := client.Execute(&Command{
		Command:        "kill -9 123456789",
		ExitCodeFilter: func(code int) bool { return code == 1 },
	}); err != nil {
		t.Error(err)
	}

	if _, err := client.Execute(&Command{
		Command: "top -d 1 -n 3",
	}); err != nil {
		t.Error(err)
	}
}

func TestClient_ExecuteWithInputArgs(t *testing.T) {
	client, err := NewClient(testHost, testUser, testPass, WithDebug())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	if _, err := client.Execute(&Command{
		Command:       "passwd",
		InputArgs:     []string{testPass, testPass},
		InputInterval: time.Millisecond * 500,
	}); err != nil {
		t.Error(err)
	}

	_, _ = client.Execute(&Command{Command: "rm /system/.ssh/id_rsa"})
	if _, err := client.Execute(&Command{
		Command:       "ssh-keygen -t rsa",
		InputArgs:     []string{"", "pass", "pass"}, // 第一个空串表示回车接受默认路径
		InputInterval: time.Millisecond * 700,
		StderrFilter: func(stderr error) bool {
			return strings.Contains(stderr.Error(), "command succeeded but stderr produced")
		},
	}); err != nil {
		t.Error(err)
	}
}

func TestClient_ExecuteWithTimeout(t *testing.T) {
	client, err := NewClient(testHost, testUser, testPass, WithDebug(1024))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	_, _ = client.Execute(&Command{Command: "killall -9 tester"})
	if res, err := client.Execute(&Command{
		Command: "/system/bin/tester -r",
		Timeout: 1 * time.Second,
	}); err == nil {
		t.Error(res)
	}

	if _, err := client.Execute(&Command{
		Command:        "ping 192.0.2.1",
		Timeout:        5 * time.Second,
		ExitCodeFilter: func(code int) bool { return code == 130 },
	}); err != nil {
		t.Error(err)
	}

	if _, err := client.Execute(&Command{
		Command: "top",
		Timeout: 5 * time.Second,
	}); err != nil {
		t.Error(err)
	}
}

func TestClient_ExecuteContext(t *testing.T) {
	client, err := NewClient(testHost, testUser, testPass, WithDebug(1024))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*7)
	defer cancel()

	_, _ = client.Execute(&Command{Command: "killall -9 tester"})
	if _, err = client.ExecuteContext(ctx, &Command{
		Command:        "/system/bin/tester -r",
		ExitCodeFilter: func(code int) bool { return 129 == code },
	}); err != nil {
		t.Error(err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*7)
	go func() {
		<-time.After(time.Second * 3)
		cancel()
	}()

	if _, err := client.ExecuteContext(ctx, &Command{
		Command:        "top",
		ExitCodeFilter: func(code int) bool { return 129 == code },
	}); err != nil {
		t.Error(err)
	}
}

func TestClient_ExecuteContextWithInputArgs(t *testing.T) {
	client, err := NewClient(testHost, testUser, testPass, WithDebug())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*7)
	defer cancel()

	if _, err = client.ExecuteContext(ctx, &Command{
		Command:   "top",
		InputArgs: []string{"1"},
	}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*7)
	defer cancel()

	if _, err = client.ExecuteContext(ctx, &Command{
		Command:   "top",
		InputArgs: []string{"1"},
		ExitCodeFilter: func(code int) bool {
			switch code {
			case 0, 141:
				return true
			default:
				return false
			}
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestExecute_Testers(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	_, _ = client.Execute(&Command{Command: "killall -9 tester"})
	wg := &sync.WaitGroup{}
	for i := range 3 {
		wg.Go(func() {
			_, _ = client.Execute(&Command{
				Command: "/system/bin/tester -r",
				Timeout: time.Second * time.Duration(5*(i+1)),
			})
		})
	}

	wg.Wait()
}
