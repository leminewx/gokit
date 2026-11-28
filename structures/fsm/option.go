package fsm

// 选项配置
type Option[T any] func(*FSM[T])

// WithEventErrorPolicy 单独设置事件回调错误策略（默认 true）
func WithEventErrorPolicy[T any](stop bool) Option[T] {
	return func(f *FSM[T]) { f.stopOnEventError = stop }
}

// WithLeaveErrorPolicy 单独设置离开回调错误策略（默认 true）
func WithLeaveErrorPolicy[T any](stop bool) Option[T] {
	return func(f *FSM[T]) { f.stopOnLeaveError = stop }
}

// WithEnterErrorPolicy 单独设置进入回调错误策略（默认 true）
func WithEnterErrorPolicy[T any](stop bool) Option[T] {
	return func(f *FSM[T]) { f.stopOnEnterError = stop }
}

// WithStopOnError 统一设置所有回调出错时是否中止转移（默认 true）
func WithStopOnError[T any](stop bool) Option[T] {
	return func(f *FSM[T]) {
		f.stopOnEventError = stop
		f.stopOnLeaveError = stop
		f.stopOnEnterError = stop
	}
}
