package timewheel

import (
	"errors"
	"fmt"
)

// _ERROR_PREFIX 定义本包错误消息的统一前缀。
const _ERROR_PREFIX = "gokit.scheduler.timewheel: "

var (
	// ErrInvalidTickDuration 表示滴答间隔非法（必须大于 0）
	ErrInvalidTickDuration = errors.New(_ERROR_PREFIX + "invalid tick duration")
	// ErrInvalidSlotCount 表示槽位数量非法（必须大于 0）
	ErrInvalidSlotCount = errors.New(_ERROR_PREFIX + "invalid slot count")
	// ErrTaskPanic 表示任务函数执行时发生 panic（已被恢复，不会终止进程）
	ErrTaskPanic = errors.New(_ERROR_PREFIX + "task panicked")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
