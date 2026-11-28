package stringx

import (
	"errors"
	"fmt"
)

// _ERROR_PREFIX 定义本包错误消息的统一前缀。
const _ERROR_PREFIX = "gokit.stringx: "

// ErrInvalidNumber 表示字符串无法解析为整数
var ErrInvalidNumber = errors.New(_ERROR_PREFIX + "failed to parse integer text")

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
