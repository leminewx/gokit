package fsm

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.structures.fsm: "

var (
	// ErrStateNotFound 表示状态未在状态集中注册
	ErrStateNotFound = errors.New(_ERROR_PREFIX + "state not found")
	// ErrTransitionNotFound 表示未找到匹配的转移规则
	ErrTransitionNotFound = errors.New(_ERROR_PREFIX + "transition not found")
	// ErrTransitionExists 表示同一 (from, event) 的转移规则已存在
	ErrTransitionExists = errors.New(_ERROR_PREFIX + "transition already exists")
	// ErrEventCallback 表示事件回调执行失败
	ErrEventCallback = errors.New(_ERROR_PREFIX + "failed to run event callback")
	// ErrLeaveCallback 表示离开回调执行失败
	ErrLeaveCallback = errors.New(_ERROR_PREFIX + "failed to run leave callback")
	// ErrEnterCallback 表示进入回调执行失败
	ErrEnterCallback = errors.New(_ERROR_PREFIX + "failed to run enter callback")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}

// wrapStateError 构造状态未注册的错误
func wrapStateError(state State) error {
	return wrapError(ErrStateNotFound, "%q", state)
}
