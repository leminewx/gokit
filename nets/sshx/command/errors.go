package command

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.nets.sshx.command: "

var (
	// ErrParse 表示命令输出不符合预期格式，无法解析
	ErrParse = errors.New(_ERROR_PREFIX + "failed to parse output")
	// ErrInvalidArgument 表示传入参数非法
	ErrInvalidArgument = errors.New(_ERROR_PREFIX + "invalid argument")
	// ErrNoData 表示远端未返回任何可用分段，通常是命令全部执行失败
	ErrNoData = errors.New(_ERROR_PREFIX + "no data collected")
)

// wrapError 在保留哨兵错误的前提下附加位置信息，便于 errors.Is 判定与日志定位
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}

	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
