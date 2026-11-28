// Package timex 提供可注入的时间源抽象。
//
// 时间轮、延时队列等依赖"当前时间"与"周期嘀嗒"的组件通过 Clock 取时，
// 生产环境使用系统时钟（NewSystemClock），测试环境可注入 Manual 手动推进时间，
// 从而把依赖真实等待的用例改写为确定性用例。
package timex

import "time"

type (
	// Ticker 定义周期嘀嗒器
	Ticker interface {
		// GetChannel 返回嘀嗒事件的接收通道
		GetChannel() <-chan time.Time
		// Stop 停止嘀嗒器并释放资源
		Stop()
	}

	// Clock 定义时间源
	Clock interface {
		// GetNow 返回当前时刻
		GetNow() time.Time
		// NewTicker 新建一个周期为 interval 的嘀嗒器
		NewTicker(interval time.Duration) Ticker
	}
)

// NewSystemClock 返回系统时钟（生产环境默认使用）
func NewSystemClock() Clock {
	return systemClock{}
}

// 编译期断言：系统时钟与系统嘀嗒器实现了本包接口
var _ Clock = systemClock{}
var _ Ticker = (*systemTicker)(nil)

// systemClock 基于 time 包的时钟实现
type systemClock struct{}

// GetNow 返回系统当前时刻
func (systemClock) GetNow() time.Time {
	return time.Now()
}

// NewTicker 新建基于 time.Ticker 的嘀嗒器
func (systemClock) NewTicker(interval time.Duration) Ticker {
	return &systemTicker{ticker: time.NewTicker(interval)}
}

// systemTicker 基于 time.Ticker 的嘀嗒器实现
type systemTicker struct {
	ticker *time.Ticker
}

// GetChannel 返回嘀嗒事件的接收通道
func (own *systemTicker) GetChannel() <-chan time.Time {
	return own.ticker.C
}

// Stop 停止嘀嗒器
func (own *systemTicker) Stop() {
	own.ticker.Stop()
}
