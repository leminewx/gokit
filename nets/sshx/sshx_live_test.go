//go:build integration

// 面向真实设备的集成用例（设备无关：仅使用 POSIX 基础命令），
// 通过环境变量指定设备后运行：
//
//	$env:GOKIT_SSH_HOST="10.0.0.10"
//	$env:GOKIT_SSH_PASS="******"
//	go test -tags integration -run TestLive -v ./nets/sshx
//
// 与 session_test.go / stream_test.go 中面向特定 Android 设备的用例不同，
// 本文件的断言只依赖 sh 内建与 coreutils/toybox 共有命令，适配 Linux 与 Android。

package sshx

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// liveClient 建立关闭时自动回收的客户端（关闭重连，避免用例互相干扰）
func liveClient(t *testing.T, opts ...Option) *Client {
	t.Helper()

	target := loadLiveTarget(t)
	options := []Option{
		WithPort(target.port),
		WithTimeout(10 * time.Second),
		WithReconnect(0),
	}
	options = append(options, opts...)

	client, err := NewClient(target.host, target.user, target.pass, options...)
	if err != nil {
		t.Fatalf("connect %s:%d failed: %v", target.host, target.port, err)
	}

	t.Cleanup(func() { _ = client.Close() })
	return client
}

// TestLive_Probe 打印设备基础信息，便于定位设备类型与可用命令
func TestLive_Probe(t *testing.T) {
	client := liveClient(t)

	out, err := client.RunCommand("uname -a 2>/dev/null; cat /etc/os-release 2>/dev/null | head -n 2; id -un")
	if err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	t.Logf("device info:\n%s", out)
}

// TestLive_RunCommand 验证非交互式命令执行与多行输出
func TestLive_RunCommand(t *testing.T) {
	client := liveClient(t)

	out, err := client.RunCommand("printf 'a\\nb\\n'")
	if err != nil {
		t.Fatalf("RunCommand: %v", err)
	}
	if !strings.Contains(string(out), "a") || !strings.Contains(string(out), "b") {
		t.Fatalf("unexpected output: %q", out)
	}

	// 退出码非零必须返回错误
	if _, err := client.RunCommand("exit 3"); err == nil {
		t.Fatal("non-zero exit should return error")
	}

	// 空命令由远端 shell 处理，不 panic
	if _, err := client.RunCommand(""); err != nil {
		t.Logf("empty command err (tolerated): %v", err)
	}
}

// TestLive_Execute 验证 PTY 执行、ANSI 清除与基础错误分支
func TestLive_Execute(t *testing.T) {
	client := liveClient(t)

	// 空命令校验
	if _, err := client.Execute(&Command{}); !errors.Is(err, ErrEmptyCommand) {
		t.Fatalf("empty command err = %v, want ErrEmptyCommand", err)
	}
	if _, err := client.ExecuteContext(nil, &Command{Command: "echo hi"}); err != nil {
		t.Fatalf("nil ctx should fall back to background: %v", err)
	}

	// 带颜色输出：PTY 下会带 ANSI 转义，handleOutput 会剥离
	out, err := client.Execute(&Command{Command: "printf '\\033[31mred\\033[0m\\n'"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(string(out), "\x1b[") {
		t.Fatalf("ANSI escape not stripped: %q", out)
	}
	if !strings.Contains(string(out), "red") {
		t.Fatalf("stdout = %q, want contains red", out)
	}

	// 退出码过滤
	if _, err := client.Execute(&Command{
		Command:        "exit 7",
		ExitCodeFilter: func(code int) bool { return code == 7 },
	}); err != nil {
		t.Fatalf("exit code filter: %v", err)
	}

	// stderr 产生但退出码 0：默认视为错误，可用 StderrFilter 忽略
	if _, err := client.Execute(&Command{Command: "sh -c 'echo boom >&2'"}); err != nil {
		t.Logf("stderr surfaced as error (expected): %v", err)
	}
	if _, err := client.Execute(&Command{
		Command:      "sh -c 'echo boom >&2'",
		StderrFilter: func(error) bool { return true },
	}); err != nil {
		t.Fatalf("stderr filter: %v", err)
	}
}

// TestLive_ExecuteTimeoutAndCancel 验证超时与上下文取消能及时返回
func TestLive_ExecuteTimeoutAndCancel(t *testing.T) {
	client := liveClient(t)

	start := time.Now()
	if _, err := client.Execute(&Command{
		Command: "sleep 10",
		Timeout: 2 * time.Second,
	}); err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed := time.Since(start); elapsed > 8*time.Second {
		t.Fatalf("timeout took too long: %s", elapsed)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := client.ExecuteContext(ctx, &Command{Command: "sleep 10"}); err == nil {
		t.Fatal("expected cancel error")
	}
}

// TestLive_ExecuteStdin 验证输入参数写入远端 stdin
func TestLive_ExecuteStdin(t *testing.T) {
	client := liveClient(t)

	out, err := client.Execute(&Command{
		Command:       "cat",
		InputArgs:     []string{"alpha", "beta"},
		InputInterval: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Execute with stdin: %v", err)
	}

	text := string(out)
	if !strings.Contains(text, "alpha") || !strings.Contains(text, "beta") {
		t.Fatalf("stdin not echoed: %q", text)
	}
	// 顺序必须与输入一致
	if strings.Index(text, "alpha") > strings.Index(text, "beta") {
		t.Fatalf("stdin order broken: %q", text)
	}
}

// liveStreamHandler 线程安全的流处理器
type liveStreamHandler struct {
	lock   sync.Mutex
	stdout []string
	stderr []string
	errs   []error
	done   error
	doneCh chan struct{}
}

func newLiveStreamHandler() *liveStreamHandler {
	return &liveStreamHandler{doneCh: make(chan struct{}, 1)}
}

func (own *liveStreamHandler) CallOnStdin(input string) string { return input }

func (own *liveStreamHandler) CallOnStdout(line string) {
	own.lock.Lock()
	defer own.lock.Unlock()
	own.stdout = append(own.stdout, line)
}

func (own *liveStreamHandler) CallOnStderr(line string) {
	own.lock.Lock()
	defer own.lock.Unlock()
	own.stderr = append(own.stderr, line)
}

func (own *liveStreamHandler) CallOnError(err error) {
	own.lock.Lock()
	defer own.lock.Unlock()
	own.errs = append(own.errs, err)
}

func (own *liveStreamHandler) CallOnDone(err error) {
	own.lock.Lock()
	own.done = err
	own.lock.Unlock()
	select {
	case own.doneCh <- struct{}{}:
	default:
	}
}

// snapshot 返回输出快照
func (own *liveStreamHandler) snapshot() (stdout, stderr []string, errs []error, done error) {
	own.lock.Lock()
	defer own.lock.Unlock()

	return append([]string(nil), own.stdout...),
		append([]string(nil), own.stderr...),
		append([]error(nil), own.errs...),
		own.done
}

// TestLive_ExecuteStream 验证流式输出逐行回调与错误分支
func TestLive_ExecuteStream(t *testing.T) {
	client := liveClient(t)

	handler := newLiveStreamHandler()
	client.ExecuteStream(&StreamCommand{Command: "printf 'l1\\nl2\\nl3\\n'"}, handler)

	stdout, _, errs, done := handler.snapshot()
	if len(errs) > 0 {
		t.Fatalf("unexpected stream errors: %v", errs)
	}
	if done != nil {
		t.Fatalf("done = %v, want nil", done)
	}

	var got []string
	for _, line := range stdout {
		if line != "" {
			got = append(got, line)
		}
	}
	if len(got) != 3 || got[0] != "l1" || got[2] != "l3" {
		t.Fatalf("stdout lines = %v, want [l1 l2 l3]", got)
	}

	// 空命令必须走 CallOnError
	bad := newLiveStreamHandler()
	client.ExecuteStream(&StreamCommand{}, bad)
	if _, _, errs, _ := bad.snapshot(); len(errs) != 1 || !errors.Is(errs[0], ErrEmptyCommand) {
		t.Fatalf("errs = %v, want ErrEmptyCommand", errs)
	}

	// 超时：不回调 done，但应能返回
	slow := newLiveStreamHandler()
	start := time.Now()
	client.ExecuteStream(&StreamCommand{Command: "sleep 10", Timeout: 2 * time.Second}, slow)
	if elapsed := time.Since(start); elapsed > 8*time.Second {
		t.Fatalf("stream timeout took too long: %s", elapsed)
	}
}

// TestLive_CloseSemantics 验证关闭后的调用返回 ErrClosed 且 Close 幂等
func TestLive_CloseSemantics(t *testing.T) {
	target := loadLiveTarget(t)
	client, err := NewClient(target.host, target.user, target.pass, WithPort(target.port), WithTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	if client.IsClosed() {
		t.Fatal("new client should not be closed")
	}
	if client.GetClient() == nil {
		t.Fatal("GetClient should not be nil before close")
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if !client.IsClosed() {
		t.Fatal("IsClosed should be true after close")
	}
	// 可重复关闭
	if err := client.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}

	if _, err := client.RunCommand("true"); !errors.Is(err, ErrClosed) {
		t.Fatalf("RunCommand after close err = %v, want ErrClosed", err)
	}
	if _, err := client.Execute(&Command{Command: "true"}); !errors.Is(err, ErrClosed) {
		t.Fatalf("Execute after close err = %v, want ErrClosed", err)
	}

	handler := newLiveStreamHandler()
	client.ExecuteStream(&StreamCommand{Command: "true"}, handler)
	if _, _, errs, _ := handler.snapshot(); len(errs) == 0 || !errors.Is(errs[0], ErrClosed) {
		t.Fatalf("stream after close errs = %v, want ErrClosed", errs)
	}
}

// TestLive_InvalidCredentials 验证错误凭据与不可达端口返回错误而非 panic
func TestLive_InvalidCredentials(t *testing.T) {
	target := loadLiveTarget(t)

	// 错误密码必须失败。
	// 注意：部分老设备（DES crypt / PAM 仅比对前 8 字符）会接受与正确密码
	// 前缀相同的密码，因此这里使用完全不同的密码，避免把设备侧认证策略差异
	// 误判为客户端缺陷。
	if _, err := NewClient(target.host, target.user, "gokit-definitely-wrong-password",
		WithPort(target.port), WithTimeout(10*time.Second), WithReconnect(0),
	); err == nil {
		t.Fatal("wrong password should fail")
	}

	// 关闭端口：用 1 号端口（几乎不可能有 SSH 服务）
	if _, err := NewClient(target.host, target.user, target.pass,
		WithPort(1), WithTimeout(2*time.Second), WithReconnect(0),
	); err == nil {
		t.Fatal("unreachable port should fail")
	}

	// 选项错误在拨号前返回
	if _, err := NewClient(target.host, target.user, target.pass,
		WithKnownHosts("no-such-known-hosts-file"), WithReconnect(0),
	); err == nil {
		t.Fatal("invalid known_hosts should fail")
	}
	if _, err := NewClient(target.host, target.user, target.pass,
		WithHostKeyCallback(nil), WithReconnect(0),
	); err == nil {
		t.Fatal("nil host key callback should fail")
	}
}

// TestLive_SetReconnect 验证重连参数可变且不影响存活会话
func TestLive_SetReconnect(t *testing.T) {
	client := liveClient(t)

	client.SetReconnect(0)
	if out, err := client.RunCommand("echo still-alive"); err != nil || !strings.Contains(string(out), "still-alive") {
		t.Fatalf("command after SetReconnect: %q, %v", out, err)
	}

	client.SetReconnect(3, 100*time.Millisecond)
	if out, err := client.RunCommand("echo ok"); err != nil || !strings.Contains(string(out), "ok") {
		t.Fatalf("command after SetReconnect with interval: %q, %v", out, err)
	}
}
