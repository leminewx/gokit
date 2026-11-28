package pool

type (
	// Option 用于配置 Pool
	Option func(*config)

	config struct {
		workers      int
		queueSize    int
		panicHandler PanicHandler
	}
)

// WithWorkers 设置工作 goroutine 数量（缺省 4，n <= 0 时回退为 4）
func WithWorkers(n int) Option {
	return func(c *config) { c.workers = n }
}

// WithQueueSize 设置任务队列缓冲大小（默认 0，即无缓冲）
func WithQueueSize(n int) Option {
	return func(c *config) { c.queueSize = n }
}

// WithPanicHandler 设置 panic 回调
func WithPanicHandler(handler PanicHandler) Option {
	return func(c *config) { c.panicHandler = handler }
}
