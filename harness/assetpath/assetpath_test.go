package assetpath

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGetDataDir 数据目录必须真实存在（gokit 仓库自带 harness/testdata）。
func TestGetDataDir(t *testing.T) {
	t.Parallel()

	dir := GetDataDir()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("GetDataDir = %q, stat 失败: %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("GetDataDir = %q, 不是目录", dir)
	}
}

// TestResolve 常规相对路径、前导斜杠、多余分隔符都能映射到数据目录内。
func TestResolve(t *testing.T) {
	t.Parallel()

	root := GetDataDir()
	cases := []string{"test.ini", "/test.ini", "./test.ini", " sub/test.ini "}
	for _, relative := range cases {
		path, err := Resolve(relative)
		if err != nil {
			t.Errorf("Resolve(%q) 返回错误: %v", relative, err)
			continue
		}
		if !filepath.IsAbs(path) && !strings.HasPrefix(path, root) {
			t.Errorf("Resolve(%q) = %q, 不在数据目录 %q 内", relative, path, root)
		}

		inner, err := filepath.Rel(root, path)
		if err != nil || inner == ".." || strings.HasPrefix(inner, ".."+string(filepath.Separator)) {
			t.Errorf("Resolve(%q) = %q 越出数据目录 %q", relative, path, root)
		}
	}
}

// TestResolveRejects 空路径与越界路径必须被拒绝，而不是返回目录外的路径。
func TestResolveRejects(t *testing.T) {
	t.Parallel()

	if _, err := Resolve(""); !errors.Is(err, ErrEmptyPath) {
		t.Errorf("Resolve(\"\") 错误 = %v, want ErrEmptyPath", err)
	}
	if _, err := Resolve("   "); !errors.Is(err, ErrEmptyPath) {
		t.Errorf("Resolve(空白) 错误 = %v, want ErrEmptyPath", err)
	}

	for _, relative := range []string{"..", "../gokit.go", "a/../../x", "."} {
		if _, err := Resolve(relative); !errors.Is(err, ErrOutsideRoot) {
			t.Errorf("Resolve(%q) 错误 = %v, want ErrOutsideRoot", relative, err)
		}
	}
}

// TestResolveOr 非法路径回落到 fallback，fallback 也非法时返回空串。
func TestResolveOr(t *testing.T) {
	t.Parallel()

	root := GetDataDir()
	if got := ResolveOr("../escape", "test.ini"); got != filepath.Join(root, "test.ini") {
		t.Errorf("ResolveOr 回落结果 = %q, want %q", got, filepath.Join(root, "test.ini"))
	}
	if got := ResolveOr("test.ini", "missing.txt"); got != filepath.Join(root, "test.ini") {
		t.Errorf("ResolveOr 命中结果 = %q, want %q", got, filepath.Join(root, "test.ini"))
	}
	if got := ResolveOr("../escape", "../escape"); got != "" {
		t.Errorf("ResolveOr 双非法应返回空串, got %q", got)
	}
}
