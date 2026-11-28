package filex

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// listDir 返回目录下的全部文件名（用于断言没有残留临时文件）。
func listDir(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取目录 %q 失败: %v", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// TestWriteFileAtomic 首次写入后内容与权限正确，且不留下临时文件。
func TestWriteFileAtomic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	data := []byte("key = value\n")

	if err := WriteFileAtomic(path, data, 0o600); err != nil {
		t.Fatalf("WriteFileAtomic 失败: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取目标文件失败: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("内容 = %q, want %q", got, data)
	}

	if names := listDir(t, dir); len(names) != 1 || names[0] != "config.ini" {
		t.Errorf("目录残留多余文件: %v", names)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat 目标文件失败: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("权限 = %o, want 600", perm)
		}
	}
}

// TestWriteFileAtomicOverwritesTruncates 覆盖写必须截断旧内容，并更新权限。
func TestWriteFileAtomicOverwritesTruncates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.bin")
	stale := bytes.Repeat([]byte("STALE"), 4096)
	if err := os.WriteFile(path, stale, 0o644); err != nil {
		t.Fatalf("准备旧文件失败: %v", err)
	}

	fresh := []byte("ok")
	if err := WriteFileAtomic(path, fresh, 0o640); err != nil {
		t.Fatalf("WriteFileAtomic 失败: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取目标文件失败: %v", err)
	}
	if !bytes.Equal(got, fresh) {
		t.Fatalf("旧内容未被截断: len=%d, want %d", len(got), len(fresh))
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat 目标文件失败: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o640 {
			t.Errorf("权限 = %o, want 640", perm)
		}
	}
}

// TestWriteFileAtomicInvalidArgs 空路径与不存在的目录都要给出可判定错误。
func TestWriteFileAtomicInvalidArgs(t *testing.T) {
	t.Parallel()

	if err := WriteFileAtomic("", []byte("x"), 0o600); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("空路径错误 = %v, want ErrInvalidPath", err)
	}

	missing := filepath.Join(t.TempDir(), "no-such-dir", "file.txt")
	if err := WriteFileAtomic(missing, []byte("x"), 0o600); !errors.Is(err, ErrCreateTemp) {
		t.Errorf("目录不存在错误 = %v, want ErrCreateTemp", err)
	}
}

// TestWriteFileAtomicRenameFailureCleansTemp 改名失败时不留临时文件。
//
// 目标位置被同名目录占用时 os.Rename 必定失败，用它模拟定稿阶段的失败。
func TestWriteFileAtomicRenameFailureCleansTemp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "occupied")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("创建占位目录失败: %v", err)
	}

	err := WriteFileAtomic(target, []byte("x"), 0o600)
	if !errors.Is(err, ErrRename) {
		t.Fatalf("错误 = %v, want ErrRename", err)
	}

	if names := listDir(t, dir); len(names) != 1 || names[0] != "occupied" {
		t.Errorf("失败后残留临时文件: %v", names)
	}
}

// TestCopyFileAtomic 复制内容、继承源文件权限，并覆盖已有目标文件。
func TestCopyFileAtomic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	data := bytes.Repeat([]byte("payload"), 128)
	if err := os.WriteFile(src, data, 0o640); err != nil {
		t.Fatalf("准备源文件失败: %v", err)
	}
	if err := os.WriteFile(dst, []byte("stale"), 0o600); err != nil {
		t.Fatalf("准备目标文件失败: %v", err)
	}

	if err := CopyFileAtomic(src, dst); err != nil {
		t.Fatalf("CopyFileAtomic 失败: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("读取目标文件失败: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("复制内容不一致: len=%d, want %d", len(got), len(data))
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(dst)
		if err != nil {
			t.Fatalf("stat 目标文件失败: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o640 {
			t.Errorf("权限 = %o, want 640", perm)
		}
	}
}

// TestCopyFileAtomicWithMode 显式权限优先于源文件权限。
func TestCopyFileAtomicWithMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o666); err != nil {
		t.Fatalf("准备源文件失败: %v", err)
	}

	if err := CopyFileAtomicWithMode(src, dst, 0o600); err != nil {
		t.Fatalf("CopyFileAtomicWithMode 失败: %v", err)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(dst)
		if err != nil {
			t.Fatalf("stat 目标文件失败: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("权限 = %o, want 600", perm)
		}
	}

	if err := CopyFileAtomicWithMode(src, "", 0o600); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("空目标路径错误 = %v, want ErrInvalidPath", err)
	}
}

// TestCopyFileAtomicMissingSource 源文件不存在时返回可判定错误且不创建目标文件。
func TestCopyFileAtomicMissingSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "missing.bin")
	dst := filepath.Join(dir, "dst.bin")

	if err := CopyFileAtomic(src, dst); !errors.Is(err, ErrOpen) {
		t.Fatalf("错误 = %v, want ErrOpen", err)
	}
	if _, err := os.Stat(dst); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("失败时不应创建目标文件, stat err = %v", err)
	}
	if names := listDir(t, dir); len(names) != 0 {
		t.Errorf("目录残留文件: %v", names)
	}
}
