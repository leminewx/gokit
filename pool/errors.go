package pool

import "errors"

const _ERROR_PREFIX = "gokit.pool: "

var (
	// ErrClosed 表示池已完全关闭，不再接受任务
	ErrClosed = errors.New(_ERROR_PREFIX + "pool is closed")
	// ErrClosing 表示池正在关闭，任务未能入队
	ErrClosing = errors.New(_ERROR_PREFIX + "pool is closing")
)
