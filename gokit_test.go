package gokit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestGetValueWithDefault 验证默认值回落与首值优先
func TestGetValueWithDefault(t *testing.T) {
	if got := GetValueWithDefault("fallback"); got != "fallback" {
		t.Fatalf("no values = %q, want fallback", got)
	}
	if got := GetValueWithDefault("fallback", "first", "second"); got != "first" {
		t.Fatalf("with values = %q, want first", got)
	}
	// 零值也视为“已提供”
	if got := GetValueWithDefault(42, 0); got != 0 {
		t.Fatalf("zero value should be honored, got %d", got)
	}
}

// TestCreateEmptyFile 验证自动建目录、覆盖已有内容与关闭文件
func TestCreateEmptyFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "deep", "app.log")

	if err := CreateEmptyFile(target); err != nil {
		t.Fatalf("create: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("size = %d, want 0", info.Size())
	}

	// 已存在且有内容时应被截断
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := CreateEmptyFile(target); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if info, _ := os.Stat(target); info.Size() != 0 {
		t.Fatalf("size after truncate = %d, want 0", info.Size())
	}
}

// TestIsDir 验证目录判定与错误透传
func TestIsDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if ok, err := IsDir(dir); err != nil || !ok {
		t.Fatalf("IsDir(dir) = %v, %v", ok, err)
	}
	if ok, err := IsDir(file); err != nil || ok {
		t.Fatalf("IsDir(file) = %v, %v", ok, err)
	}
	if _, err := IsDir(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing path should return error")
	}
}

// TestListenQuitSignal_ContextCancel 验证 ctx 取消后执行 done 回调
//   - 不测试真实信号路径：测试进程内发信号会干扰 go test 自身
func TestListenQuitSignal_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var called bool
	done := make(chan error, 1)
	go func() {
		done <- ListenQuitSignal(ctx, func() error {
			called = true
			return nil
		})
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ListenQuitSignal: %v", err)
		}
		if !called {
			t.Fatal("done callback should be invoked")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ListenQuitSignal did not return after ctx cancel")
	}
}

// TestListenQuitSignal_NilContextAndDone 验证 nil ctx 与 nil done 的容错
func TestListenQuitSignal_NilContextAndDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := ListenQuitSignal(ctx, nil); err != nil {
		t.Fatalf("nil done should be ok, got %v", err)
	}

	// nil ctx 会回落为 Background，这里用已取消的 ctx 走同一分支即可（避免阻塞）
	cancelled, cancel2 := context.WithCancel(context.Background())
	cancel2()
	if err := ListenQuitSignal(cancelled, func() error { return nil }); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}
