/*
Package filex 提供进程异常退出时也不会损坏目标文件的文件操作。

核心思路是「同目录临时文件 + 落盘 + 原子改名」：

  - 临时文件建在目标文件所在目录，保证改名不跨文件系统（跨文件系统时
    os.Rename 会退化为复制+删除，失去原子性）；
  - 写入完成后先 Chmod 到目标权限、再 Sync 落盘，最后才 Rename 覆盖，
    因此任何一步失败都不会破坏已存在的目标文件；
  - 失败路径统一删除临时文件，不会在目录里留下半成品。

典型场景：配置文件热更新、任务产物落盘、私钥/证书写入——这些文件被写坏
后进程重启即无法恢复，用 WriteFileAtomic 代替 os.WriteFile 可以避免。
*/
package filex

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// _DEFAULT_PERM 为调用方未显式指定权限时的兜底值（仅属主可读写）。
const _DEFAULT_PERM os.FileMode = 0o600

// WriteFileAtomic 原子地把 data 写入 path。
//
// perm 为目标文件权限（Windows 上只保留只读位语义）；传 0 时按 0o600 处理。
// 目标文件的父目录必须已存在，本函数不会自动创建目录。
// 写入失败时 path 上的旧文件保持不变（临时文件会被清理）。
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	if path == "" {
		return wrapError(ErrInvalidPath, "WriteFileAtomic")
	}
	if perm == 0 {
		perm = _DEFAULT_PERM
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return wrapError(ErrCreateTemp, "%q: %w", path, err)
	}
	tmpPath := tmp.Name()

	// committed 为 true 表示临时文件已改名为目标文件，此时不能再删除
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return wrapError(ErrWrite, "%q: %w", tmpPath, err)
	}

	if err = commitTemp(tmp, tmpPath, path, perm); err != nil {
		return err
	}

	committed = true
	return nil
}

// CopyFileAtomic 原子地把 src 复制为 dst，权限继承 src（无法获取时按 0o600）。
//
// 目标文件的父目录必须已存在；失败时 dst 上的旧文件保持不变。
func CopyFileAtomic(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return wrapError(ErrOpen, "%q: %w", src, err)
	}

	perm := info.Mode().Perm()
	if perm == 0 {
		perm = _DEFAULT_PERM
	}
	return CopyFileAtomicWithMode(src, dst, perm)
}

// CopyFileAtomicWithMode 原子地把 src 复制为 dst，并显式指定目标权限。
//
// perm 传 0 时按 0o600 处理。
func CopyFileAtomicWithMode(src, dst string, perm os.FileMode) error {
	if dst == "" {
		return wrapError(ErrInvalidPath, "CopyFileAtomicWithMode")
	}
	if perm == 0 {
		perm = _DEFAULT_PERM
	}

	source, err := os.Open(src)
	if err != nil {
		return wrapError(ErrOpen, "%q: %w", src, err)
	}
	defer func() { _ = source.Close() }()

	tmp, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+".tmp-*")
	if err != nil {
		return wrapError(ErrCreateTemp, "%q: %w", dst, err)
	}
	tmpPath := tmp.Name()

	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err = io.Copy(tmp, source); err != nil {
		_ = tmp.Close()
		return wrapError(ErrCopy, "%q -> %q: %w", src, dst, err)
	}

	if err = commitTemp(tmp, tmpPath, dst, perm); err != nil {
		return err
	}

	committed = true
	return nil
}

// commitTemp 把临时文件定稿为最终文件：先收紧权限、再落盘、最后原子改名。
//
// 顺序不能调整：
//   - 权限必须先于改名固定，否则目标文件会短暂以临时文件权限（0600）存在，
//     或反过来让敏感文件出现权限过宽的窗口；
//   - Sync 必须先于 Rename，否则掉电后可能留下一个长度为 0 的目标文件。
//
// 无论成败，调用方负责在失败时删除临时文件。
func commitTemp(tmp *os.File, tmpPath, dst string, perm os.FileMode) error {
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return wrapError(ErrChmod, "%q: %w", dst, err)
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return wrapError(ErrSync, "%q: %w", tmpPath, err)
	}

	if err := tmp.Close(); err != nil {
		return wrapError(ErrWrite, "%q: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		return wrapError(ErrRename, "%q: %w", dst, err)
	}

	// 目录项也要落盘，否则掉电后可能看不到刚改名的文件（Windows 不支持）
	syncDir(filepath.Dir(dst))
	return nil
}

// syncDir 尽力把目录项刷到磁盘；失败不影响主流程（部分文件系统不允许对目录 fsync）。
func syncDir(dir string) {
	if runtime.GOOS == "windows" {
		return
	}

	handle, err := os.Open(dir)
	if err != nil {
		return
	}
	defer func() { _ = handle.Close() }()

	_ = handle.Sync()
}
