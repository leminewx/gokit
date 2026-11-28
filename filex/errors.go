package filex

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.filex: "

var (
	// ErrInvalidPath 表示目标路径为空，无法确定落盘位置
	ErrInvalidPath = errors.New(_ERROR_PREFIX + "target path is empty")
	// ErrCreateTemp 表示在目标目录创建临时文件失败（目录不存在或无写权限）
	ErrCreateTemp = errors.New(_ERROR_PREFIX + "failed to create temp file")
	// ErrOpen 表示打开源文件失败
	ErrOpen = errors.New(_ERROR_PREFIX + "failed to open source file")
	// ErrCopy 表示复制数据失败
	ErrCopy = errors.New(_ERROR_PREFIX + "failed to copy data")
	// ErrWrite 表示写入数据失败
	ErrWrite = errors.New(_ERROR_PREFIX + "failed to write data")
	// ErrChmod 表示设置文件权限失败
	ErrChmod = errors.New(_ERROR_PREFIX + "failed to chmod target file")
	// ErrSync 表示把数据刷到磁盘失败
	ErrSync = errors.New(_ERROR_PREFIX + "failed to sync target file")
	// ErrRename 表示原子改名失败（目标被目录占用或跨文件系统）
	ErrRename = errors.New(_ERROR_PREFIX + "failed to rename temp file")
	// ErrOutsideRoot 表示相对路径越出了给定根目录（含 ".." 或指向根目录本身）
	ErrOutsideRoot = errors.New(_ERROR_PREFIX + "relative path escapes root directory")
	// ErrModuleRoot 表示从工作目录与源码位置向上均未找到 go.mod
	ErrModuleRoot = errors.New(_ERROR_PREFIX + "go.mod not found")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
