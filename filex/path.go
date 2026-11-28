package filex

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// GetModuleRoot 返回包含 go.mod 的模块根目录。
//
// 先以工作目录为起点逐级向上查找；工作目录不在任何模块内时，再以本包源码位置为起点
// 重试，因此测试（包目录为工作目录）与示例（调用目录为工作目录）都能命中同一模块根。
// 找不到 go.mod 时返回 ErrModuleRoot。
func GetModuleRoot() (string, error) {
	if wd, err := os.Getwd(); err == nil {
		if root, ok := findModuleRoot(wd); ok {
			return root, nil
		}
	}

	if _, file, _, ok := runtime.Caller(0); ok {
		if root, ok := findModuleRoot(filepath.Dir(file)); ok {
			return root, nil
		}
	}

	return "", wrapError(ErrModuleRoot, "go.mod not found from working directory or package source")
}

// ResolvePath 把 relative 映射到 root 内，并保证结果不会越出 root。
//
// 前导 "/" 会被忽略（兼容 HTTP 路径写法）；空路径或空 root 返回 ErrInvalidPath，
// 越界路径（含 ".."、指向 root 本身）返回 ErrOutsideRoot。本函数只做路径计算，
// 不检查文件是否存在；符号链接不在此处做逃逸校验，需要严格防逃逸的调用方应自行
// 对最终路径做 filepath.EvalSymlinks 后复核。
func ResolvePath(root, relative string) (string, error) {
	trimmed := strings.TrimSpace(relative)
	if trimmed == "" || root == "" {
		return "", wrapError(ErrInvalidPath, "root %q, relative %q", root, relative)
	}

	target := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(trimmed, "/")))
	inner, err := filepath.Rel(root, target)
	if err != nil || inner == "." || inner == ".." || strings.HasPrefix(inner, ".."+string(filepath.Separator)) {
		return "", wrapError(ErrOutsideRoot, "%q", relative)
	}
	return target, nil
}

// findModuleRoot 从 start 开始逐级向上查找 go.mod，返回其所在目录。
func findModuleRoot(start string) (string, bool) {
	for dir := start; ; dir = filepath.Dir(dir) {
		if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !info.IsDir() {
			return dir, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
	}
}
