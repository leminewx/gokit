package mesh

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.nets.mesh: "

var (
	// ErrAddrRequired 表示创建客户端时未提供监听地址
	ErrAddrRequired = errors.New(_ERROR_PREFIX + "address is required")
	// ErrClientNotRunning 表示客户端已停止或已关闭
	ErrClientNotRunning = errors.New(_ERROR_PREFIX + "client is not running")
	// ErrClientClosed 表示客户端已被关闭
	ErrClientClosed = errors.New(_ERROR_PREFIX + "client already closed")
	// ErrNodeBuild 表示底层节点构建失败
	ErrNodeBuild = errors.New(_ERROR_PREFIX + "failed to build node")
	// ErrRequestTimeout 表示等待响应超时
	ErrRequestTimeout = errors.New(_ERROR_PREFIX + "request timed out")
	// ErrNoSubscriber 表示订阅时未提供有效回调
	ErrNoSubscriber = errors.New(_ERROR_PREFIX + "subscribe callbacks are empty")
	// ErrNilQueryHandler 表示注册的查询处理器为空
	ErrNilQueryHandler = errors.New(_ERROR_PREFIX + "query handler is nil")
	// ErrUnexpectedResponse 表示收到的响应类型与请求不匹配
	ErrUnexpectedResponse = errors.New(_ERROR_PREFIX + "unexpected response type")
	// ErrPeerNotFound 表示目标节点未建立连接
	ErrPeerNotFound = errors.New(_ERROR_PREFIX + "peer not found")
	// ErrPeerClosed 表示目标节点连接已关闭
	ErrPeerClosed = errors.New(_ERROR_PREFIX + "peer already closed")
	// ErrSendQueueFull 表示目标节点发送队列已满
	ErrSendQueueFull = errors.New(_ERROR_PREFIX + "send queue is full")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
