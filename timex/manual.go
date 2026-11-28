package timex

import (
	"sync"
	"time"
)

// _MAX_TICKS_PER_ADVANCE 单次推进单个嘀嗒器允许补发的最大嘀嗒数
//   - 防止以极小间隔一次性推进很长时间时产生海量滴答并撑爆内存
const _MAX_TICKS_PER_ADVANCE = 4096

// Manual 是可由调用方手动推进的测试时钟
//   - 时间不会自行流逝，只有 Advance / AdvanceTo 才会前进
//   - 并发安全，可在被测代码的 goroutine 与测试 goroutine 之间共享
type Manual struct {
	lock    sync.Mutex
	now     time.Time
	tickers []*manualTicker
}

// 编译期断言：手动时钟与手动嘀嗒器实现了本包接口
var _ Clock = (*Manual)(nil)

// NewManual 新建一个从 start 开始的手动时钟
func NewManual(start time.Time) *Manual {
	return &Manual{now: start}
}

// NewManualAt 新建一个从零值（UTC 纪元）开始的手动时钟
func NewManualAt() *Manual {
	return NewManual(time.Unix(0, 0).UTC())
}

// GetNow 返回手动时钟的当前时刻
func (own *Manual) GetNow() time.Time {
	own.lock.Lock()
	defer own.lock.Unlock()

	return own.now
}

// NewTicker 新建一个由手动时钟驱动的嘀嗒器
//   - interval <= 0 时按 1ns 处理，避免除零与死循环
func (own *Manual) NewTicker(interval time.Duration) Ticker {
	if interval <= 0 {
		interval = time.Nanosecond
	}

	own.lock.Lock()
	defer own.lock.Unlock()

	ticker := &manualTicker{
		clock:    own,
		interval: interval,
		next:     own.now.Add(interval),
		events:   make(chan time.Time, _MAX_TICKS_PER_ADVANCE),
	}
	own.tickers = append(own.tickers, ticker)

	return ticker
}

// Advance 把时钟向前推进 duration，并补发期间产生的嘀嗒
func (own *Manual) Advance(duration time.Duration) {
	if duration <= 0 {
		return
	}

	own.lock.Lock()
	own.now = own.now.Add(duration)
	now := own.now
	// 在持锁期间取出快照，避免补发过程与 NewTicker 并发修改切片
	tickers := make([]*manualTicker, len(own.tickers))
	copy(tickers, own.tickers)
	own.lock.Unlock()

	for _, ticker := range tickers {
		ticker.fire(now)
	}
}

// AdvanceTo 把时钟推进到 target，并补发期间产生的嘀嗒
//   - target 不晚于当前时刻时不做任何处理
func (own *Manual) AdvanceTo(target time.Time) {
	own.lock.Lock()
	var duration time.Duration
	if target.After(own.now) {
		duration = target.Sub(own.now)
	}
	own.lock.Unlock()

	own.Advance(duration)
}

// removeTicker 从活动列表中移除已停止的嘀嗒器
func (own *Manual) removeTicker(target *manualTicker) {
	own.lock.Lock()
	defer own.lock.Unlock()

	for i, ticker := range own.tickers {
		if ticker == target {
			own.tickers = append(own.tickers[:i], own.tickers[i+1:]...)
			return
		}
	}
}

// manualTicker 由 Manual 驱动的嘀嗒器
type manualTicker struct {
	clock    *Manual
	interval time.Duration
	events   chan time.Time

	lock   sync.Mutex
	next   time.Time // 下一次应触发嘀嗒的时刻
	closed bool
}

// 编译期断言：手动嘀嗒器实现 Ticker 接口
var _ Ticker = (*manualTicker)(nil)

// GetChannel 返回嘀嗒事件的接收通道
func (own *manualTicker) GetChannel() <-chan time.Time {
	return own.events
}

// Stop 停止嘀嗒器；重复调用安全
func (own *manualTicker) Stop() {
	own.lock.Lock()
	if own.closed {
		own.lock.Unlock()
		return
	}
	own.closed = true
	own.lock.Unlock()

	own.clock.removeTicker(own)
}

// fire 补发 now 之前所有应触发的嘀嗒
func (own *manualTicker) fire(now time.Time) {
	own.lock.Lock()
	defer own.lock.Unlock()

	if own.closed {
		return
	}

	for ticks := 0; own.next.Before(now) || own.next.Equal(now); ticks++ {
		if ticks >= _MAX_TICKS_PER_ADVANCE {
			// 超出补发上限时直接跳到未来，宁可丢嘀嗒也不无限循环
			own.next = now.Add(own.interval)
			return
		}

		select {
		case own.events <- own.next:
		default: // 接收方未及时消费时丢弃该嘀嗒，语义与 time.Ticker 一致
		}

		own.next = own.next.Add(own.interval)
	}
}
