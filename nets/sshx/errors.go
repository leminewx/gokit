package sshx

import (
	"errors"
	"fmt"
)

// _ERROR_PREFIX 定义本包错误消息的统一前缀。
const _ERROR_PREFIX = "gokit.nets.sshx: "

var (
	// ErrClosed 表示 SSH 客户端或连接已关闭
	ErrClosed = errors.New(_ERROR_PREFIX + "connection is closed")
	// ErrEmptyCommand 表示待执行的命令为空
	ErrEmptyCommand = errors.New(_ERROR_PREFIX + "command is empty")
	// ErrConnect 表示建立 SSH 连接失败
	ErrConnect = errors.New(_ERROR_PREFIX + "failed to connect to SSH server")
	// ErrNoConnection 表示没有可用的连接
	ErrNoConnection = errors.New(_ERROR_PREFIX + "no available connection")
	// ErrSession 表示创建或启动会话失败
	ErrSession = errors.New(_ERROR_PREFIX + "failed to prepare SSH session")
	// ErrReconnect 表示重连失败
	ErrReconnect = errors.New(_ERROR_PREFIX + "failed to reconnect")
	// ErrClose 表示关闭客户端失败
	ErrClose = errors.New(_ERROR_PREFIX + "failed to close SSH client")
	// ErrWrite 表示写入标准输入失败
	ErrWrite = errors.New(_ERROR_PREFIX + "failed to write to stdin")
	// ErrRead 表示读取命令输出失败
	ErrRead = errors.New(_ERROR_PREFIX + "failed to read from pipe")
	// ErrCommandFailed 表示命令以非零状态退出
	ErrCommandFailed = errors.New(_ERROR_PREFIX + "command failed")
	// ErrUnexpectedOutput 表示命令成功退出但产生了 stderr 输出
	ErrUnexpectedOutput = errors.New(_ERROR_PREFIX + "command succeeded but stderr produced output")
	// ErrPanic 表示会话执行过程中发生 panic（已被恢复）
	ErrPanic = errors.New(_ERROR_PREFIX + "session panicked")
	// ErrInvalidOption 表示构造客户端时传入的选项非法
	ErrInvalidOption = errors.New(_ERROR_PREFIX + "invalid client option")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
