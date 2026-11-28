// Package assetpath 定位测试/示例数据目录，并把相对路径安全地映射到该目录内。
//
// 数据目录定位与越界路径映射属于通用文件能力，已迁移到 gitee.com/leminewx/gokit/filex：
// 用 GetModuleRoot 定位模块根，用 ResolvePath 把相对路径限制在根目录内。本包保留为
// 兼容层，GetDataDir / Resolve / ResolveOr 的签名与错误哨兵不变，内部委托 filex 实现。
//
// Deprecated: 新代码请直接使用 filex.GetModuleRoot 与 filex.ResolvePath，本包将在
// 下一个 MAJOR 版本移除。
package assetpath

import (
	"errors"
	"os"
	"path/filepath"

	"gitee.com/leminewx/gokit/filex"
)

const (
	// _FALLBACK_DIR 为所有候选都不可用时的兜底相对路径。
	_FALLBACK_DIR = "harness/testdata"
	// _HARNESS_DIR 为沿目录向上查找时优先命中的子目录名。
	_HARNESS_DIR = "harness"
	// _TESTDATA_DIR 为数据目录名。
	_TESTDATA_DIR = "testdata"
)

// GetDataDir 返回测试/示例数据目录。
//
// 查找顺序：
//  1. 从工作目录开始逐级向上查找 harness/testdata、testdata；
//  2. 以模块根（filex.GetModuleRoot）为起点重复上述查找；
//  3. 全部不可用时返回兜底相对路径 "harness/testdata"，由调用方在读取时报错。
//
// 返回值可能是相对路径（兜底时），调用方不应假设它一定存在。
//
// Deprecated: 使用 filex.GetModuleRoot 定位模块根，再与 filex.ResolvePath 组合。
func GetDataDir() string {
	if wd, err := os.Getwd(); err == nil {
		if dir, ok := findDataDir(wd); ok {
			return dir
		}
	}

	if root, err := filex.GetModuleRoot(); err == nil {
		if dir, ok := findDataDir(root); ok {
			return dir
		}
	}

	return _FALLBACK_DIR
}

// Resolve 把相对路径映射到数据目录内，并保证结果不会越界。
//
// 前导 "/" 会被忽略（兼容 HTTP 路径写法）；空路径返回 ErrEmptyPath，
// 越界路径（含 ".."、指向目录本身）返回 ErrOutsideRoot。本函数只做路径计算，
// 不检查文件是否存在；符号链接不在此处做逃逸校验，需要严格防逃逸的调用方应自行
// 对最终路径做 filepath.EvalSymlinks 后复核。
//
// Deprecated: 使用 filex.ResolvePath(root, relative)。
func Resolve(relative string) (string, error) {
	path, err := filex.ResolvePath(GetDataDir(), relative)

	switch {
	case errors.Is(err, filex.ErrInvalidPath):
		return "", wrapError(ErrEmptyPath, "%q", relative)
	case errors.Is(err, filex.ErrOutsideRoot):
		return "", wrapError(ErrOutsideRoot, "%q", relative)
	}

	return path, err
}

// ResolveOr 在 Resolve 失败时回落到 fallback（同样相对数据目录）。
//
// fallback 也非法时返回空串，调用方应据此报错。典型用法：示例服务把
// 未命中的下载路径回落到 data.txt，测试则回落到固定的 fixture 文件。
//
// Deprecated: 使用 filex.ResolvePath 自行回落。
func ResolveOr(relative, fallback string) string {
	if path, err := Resolve(relative); err == nil {
		return path
	}
	if path, err := Resolve(fallback); err == nil {
		return path
	}
	return ""
}

// findDataDir 从 start 开始逐级向上查找 harness/testdata 或 testdata。
func findDataDir(start string) (string, bool) {
	for dir := start; ; dir = filepath.Dir(dir) {
		for _, candidate := range []string{
			filepath.Join(dir, _HARNESS_DIR, _TESTDATA_DIR),
			filepath.Join(dir, _TESTDATA_DIR),
		} {
			if isDir(candidate) {
				return filepath.Clean(candidate), true
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
	}
}

// isDir 判断路径是否存在且为目录。
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
