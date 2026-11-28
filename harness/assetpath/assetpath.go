// Package assetpath 定位测试/示例数据目录，并把相对路径安全地映射到该目录内。
//
// go test 以包目录为工作目录，go run 以调用目录为工作目录，两者并不一致，
// 因此这里先按源码位置推断，再沿工作目录逐级向上查找 harness/testdata，
// 保证「go run ./harness/mesh_demo」与「go test ./...」都能命中同一份数据。
//
// Resolve 会拒绝越界路径（含 ".."），避免把数据目录之外的仓库文件暴露给
// 文件下载接口或示例服务。
package assetpath

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// _FALLBACK_DIR 为所有候选都不可用时的兜底相对路径。
	_FALLBACK_DIR = "harness/testdata"
	// _HARNESS_DIR 为沿工作目录向上查找时优先命中的子目录名。
	_HARNESS_DIR = "harness"
	// _TESTDATA_DIR 为数据目录名。
	_TESTDATA_DIR = "testdata"
)

// GetDataDir 返回测试/示例数据目录。
//
// 查找顺序：
//  1. 源码位置推断：<包目录>/../testdata、<包目录>/../../testdata；
//  2. 从工作目录开始逐级向上查找 harness/testdata、testdata；
//  3. 全部不可用时返回兜底相对路径 "harness/testdata"，由调用方在读取时报错。
//
// 返回值可能是相对路径（兜底时），调用方不应假设它一定存在。
func GetDataDir() string {
	if _, file, _, ok := runtime.Caller(0); ok {
		base := filepath.Dir(file)
		for _, candidate := range []string{
			filepath.Join(base, "..", _TESTDATA_DIR),       // harness/testdata
			filepath.Join(base, "..", "..", _TESTDATA_DIR), // 仓库根 testdata
		} {
			if isDir(candidate) {
				return filepath.Clean(candidate)
			}
		}
	}

	if wd, err := os.Getwd(); err == nil {
		for dir := wd; ; dir = filepath.Dir(dir) {
			for _, candidate := range []string{
				filepath.Join(dir, _HARNESS_DIR, _TESTDATA_DIR),
				filepath.Join(dir, _TESTDATA_DIR),
			} {
				if isDir(candidate) {
					return filepath.Clean(candidate)
				}
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}

	return _FALLBACK_DIR
}

// Resolve 把相对路径映射到数据目录内，并保证结果不会越界。
//
// 前导 "/" 会被忽略（兼容 HTTP 路径写法）；空路径返回 ErrEmptyPath，
// 越界路径（含 ".."、指向目录本身）返回 ErrOutsideRoot。
// 本函数只做路径计算，不检查文件是否存在；符号链接不在此处做逃逸校验，
// 需要严格防逃逸的调用方应自行对最终路径做 filepath.EvalSymlinks 后复核。
func Resolve(relative string) (string, error) {
	trimmed := strings.TrimSpace(relative)
	if trimmed == "" {
		return "", wrapError(ErrEmptyPath, "%q", relative)
	}

	root := GetDataDir()
	target := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(trimmed, "/")))

	inner, err := filepath.Rel(root, target)
	if err != nil || inner == "." || inner == ".." || strings.HasPrefix(inner, ".."+string(filepath.Separator)) {
		return "", wrapError(ErrOutsideRoot, "%q", relative)
	}
	return target, nil
}

// ResolveOr 在 Resolve 失败时回落到 fallback（同样相对数据目录）。
//
// fallback 也非法时返回空串，调用方应据此报错。典型用法：示例服务把
// 未命中的下载路径回落到 data.txt，测试则回落到固定的 fixture 文件。
func ResolveOr(relative, fallback string) string {
	if path, err := Resolve(relative); err == nil {
		return path
	}
	if path, err := Resolve(fallback); err == nil {
		return path
	}
	return ""
}

// isDir 判断路径是否存在且为目录。
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
