package assetpath

import (
	"errors"
	"fmt"
)

// _ERROR_PREFIX 定义本包错误消息的统一前缀。
const _ERROR_PREFIX = "gokit.harness.assetpath: "

var (
	// ErrEmptyPath 表示传入的相对路径为空，无法定位文件
	ErrEmptyPath = errors.New(_ERROR_PREFIX + "relative path is empty")
	// ErrOutsideRoot 表示相对路径越出了数据目录（含 ".." 或指向目录本身）
	ErrOutsideRoot = errors.New(_ERROR_PREFIX + "relative path escapes data directory")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
