package version

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.version: "

var (
	// ErrEmptyVersion 表示版本号为空，无法参与比较或区间判定
	ErrEmptyVersion = errors.New(_ERROR_PREFIX + "version is empty")
	// ErrInvalidVersionExpr 表示版本区间表达式不合法（录入规范或语法错误）
	ErrInvalidVersionExpr = errors.New(_ERROR_PREFIX + "invalid version expression")
	// ErrInvalidVersion 表示版本号写法不合法（空段、通配符位置错误等）
	ErrInvalidVersion = errors.New(_ERROR_PREFIX + "invalid version")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
