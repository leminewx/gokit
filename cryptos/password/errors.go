package password

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.cryptos.password: "

var (
	// ErrPasswordMismatch 表示密码与摘要不匹配
	ErrPasswordMismatch = errors.New(_ERROR_PREFIX + "password mismatch")
	// ErrInvalidCost 表示 bcrypt 代价因子越界（[4, 31]）
	ErrInvalidCost = errors.New(_ERROR_PREFIX + "bcrypt cost must be between 4 and 31")
	// ErrInvalidHash 表示摘要不是合法的 bcrypt 格式（未哈希、被截断等）
	ErrInvalidHash = errors.New(_ERROR_PREFIX + "hash is not a valid bcrypt digest")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
