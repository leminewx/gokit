package filex

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGetModuleRoot 返回值必须是一个真实包含 go.mod 的目录。
func TestGetModuleRoot(t *testing.T) {
	t.Parallel()

	root, err := GetModuleRoot()
	if err != nil {
		t.Fatalf("GetModuleRoot 返回错误: %v", err)
	}

	info, err := os.Stat(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("go.mod 不存在于 %q: %v", root, err)
	}
	if info.IsDir() {
		t.Fatalf("%q 下的 go.mod 不是文件", root)
	}
}

// TestResolvePath 常规相对路径、前导斜杠、多余分隔符都应落在根目录内。
func TestResolvePath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, relative := range []string{"data.txt", "/data.txt", "./data.txt", " sub/data.txt "} {
		path, err := ResolvePath(root, relative)
		if err != nil {
			t.Errorf("ResolvePath(%q) 返回错误: %v", relative, err)
			continue
		}

		inner, err := filepath.Rel(root, path)
		if err != nil || inner == ".." || strings.HasPrefix(inner, ".."+string(filepath.Separator)) {
			t.Errorf("ResolvePath(%q) = %q 越出根目录 %q", relative, path, root)
		}
	}
}

// TestResolvePathRejects 空路径、空根与越界路径必须被拒绝，而不是返回根目录之外的路径。
func TestResolvePathRejects(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, relative := range []string{"", "   "} {
		if _, err := ResolvePath(root, relative); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("ResolvePath(%q) 错误 = %v, want ErrInvalidPath", relative, err)
		}
	}
	if _, err := ResolvePath("", "data.txt"); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("ResolvePath 空根错误 = %v, want ErrInvalidPath", err)
	}

	for _, relative := range []string{"..", "../gokit.go", "a/../../x", "."} {
		if _, err := ResolvePath(root, relative); !errors.Is(err, ErrOutsideRoot) {
			t.Errorf("ResolvePath(%q) 错误 = %v, want ErrOutsideRoot", relative, err)
		}
	}
}
