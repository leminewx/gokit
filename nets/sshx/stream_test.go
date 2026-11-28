//go:build integration

// 外部服务依赖测试：需要可登录的真实 SSH 服务器才能运行，默认不参与构建与测试。
//
// 执行方式：go test -tags integration ./nets/sshx

package sshx

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockStreamHandler 用于流式测试的模拟处理器
type mockStreamHandler struct {
	mu          sync.Mutex
	stdoutLines []string
	stderrLines []string
	errors      []error
	doneErr     error
}

func (m *mockStreamHandler) CallOnStdin(input string) string { return input }

func (m *mockStreamHandler) CallOnStdout(line string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stdoutLines = append(m.stdoutLines, line)
	fmt.Println("stdout >>> ", line)
}

func (m *mockStreamHandler) CallOnStderr(line string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stderrLines = append(m.stderrLines, line)
	fmt.Println("stderr >>> ", line)
}

func (m *mockStreamHandler) CallOnError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors = append(m.errors, err)
	fmt.Println("error >>> ", err)
}

func (m *mockStreamHandler) CallOnDone(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.doneErr = err
	fmt.Println("done >>> ", err)
}

// ------------------ 流式接口测试 ------------------

func TestExecuteByStream_Simple(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	handler := &mockStreamHandler{}
	client.ExecuteStream(&StreamCommand{
		Command: "echo hello world",
	}, handler)

	if len(handler.stdoutLines) != 1 || !strings.Contains(handler.stdoutLines[0], "hello world") {
		t.Errorf("unexpected stdout: %d, %v", len(handler.stdoutLines), handler.stdoutLines)
	}
	if handler.doneErr != nil {
		t.Errorf("doneErr should be nil, got %v", handler.doneErr)
	}
}

func TestExecuteByStream_WithInput(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	handler := &mockStreamHandler{}
	client.ExecuteStream(&StreamCommand{
		Command:       "cat",
		InputArgs:     []string{"line1", "line2"},
		InputInterval: 100 * time.Millisecond,
	}, handler)

	expected := []string{"line1", "line2"}
	if len(handler.stdoutLines) != len(expected) {
		t.Errorf("expected %d lines, got %d", len(expected), len(handler.stdoutLines))
	} else {
		for i, line := range handler.stdoutLines {
			if !strings.Contains(line, expected[i]) {
				t.Errorf("stdout[%d] = %q, want %q", i, line, expected[i])
			}
		}
	}
}

func TestExecuteByStream_Timeout(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	handler := &mockStreamHandler{}
	client.ExecuteStream(&StreamCommand{
		Command: "sleep 5",
		Timeout: 1 * time.Second,
	}, handler)

	// 应该触发取消
	if handler.doneErr != nil {
		t.Error("expected doneErr nil")
	}
}

func TestExecuteByStream_Cancel(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	handler := &mockStreamHandler{}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()
	client.ExecuteStreamContext(ctx, &StreamCommand{
		Command: "sleep 5",
	}, handler)

	if handler.doneErr != nil {
		t.Error("expected doneErr nil")
	}
}

func TestExecuteByStream_Stderr(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	handler := &mockStreamHandler{}
	client.ExecuteStream(&StreamCommand{
		Command: "sh -c 'echo error >&2'",
	}, handler)

	if len(handler.stdoutLines) != 1 || !strings.Contains(handler.stdoutLines[0], "error") {
		t.Errorf("unexpected stderr: %v", handler.stderrLines)
	}
	if handler.doneErr != nil {
		t.Errorf("doneErr should be nil, got %v", handler.doneErr)
	}
}

func TestExecuteByStream_LongOutput(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()

	handler := &mockStreamHandler{}
	client.ExecuteStream(&StreamCommand{
		Command: "seq 1 100",
	}, handler)

	if len(handler.stdoutLines) != 100 {
		t.Errorf("expected 100 lines, got %d", len(handler.stdoutLines))
	}
	// 验证第一行和最后一行
	if handler.stdoutLines[0] != "1" {
		t.Errorf("first line = %q, want '1'", handler.stdoutLines[0])
	}
	if handler.stdoutLines[99] != "100" {
		t.Errorf("last line = %q, want '100'", handler.stdoutLines[99])
	}
}

func TestExecuteByStream_Testers(t *testing.T) {
	client := testClient(t)
	defer func() { _ = client.Close() }()
	wg := &sync.WaitGroup{}
	handler := &mockStreamHandler{}

	_, _ = client.Execute(&Command{
		Command: "killall -9 tester",
	})

	for i := range 3 {
		wg.Go(func() {
			client.ExecuteStream(&StreamCommand{
				Command: "/system/bin/tester -r",
				Timeout: time.Second * time.Duration(3+i*2),
			}, handler)
		})
	}

	wg.Wait()
}
