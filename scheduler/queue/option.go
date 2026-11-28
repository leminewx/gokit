package queue

import (
	"log"

	"gitee.com/leminewx/gokit/timex"
)

// ErrorHandler 定义任务回调 panic 的处理函数
//   - taskID：发生 panic 的任务 ID
//   - err：包装了 ErrTaskPanic 的错误，附带 panic 值与调用栈
type ErrorHandler func(taskID int64, err error)

// Option 定义延时队列的构造选项
type Option func(*DelayQueue)

// WithClock 指定延时队列使用的时间源，缺省使用系统时钟
func WithClock(source timex.Clock) Option {
	return func(own *DelayQueue) {
		if source != nil {
			own.clock = source
		}
	}
}

// WithErrorHandler 指定任务回调 panic 的处理函数，缺省写入标准日志
func WithErrorHandler(handler ErrorHandler) Option {
	return func(own *DelayQueue) {
		if handler != nil {
			own.handleError = handler
		}
	}
}

// WithChanSize 指定任务投递通道的缓冲大小，缺省 1024
func WithChanSize(size int) Option {
	return func(own *DelayQueue) {
		if size > 0 {
			own.taskChan = make(chan *task, size)
		}
	}
}

// logErrorHandler 缺省的任务 panic 处理：写入标准日志
func logErrorHandler(taskID int64, err error) {
	log.Printf("[ERROR] gokit.scheduler.queue: task %d: %v", taskID, err)
}
