package speedometer

import (
	"sync"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

// Speedometer 计速器接口
type Speedometer interface {
	// Add 增加事件计数，可添加任意正整数
	//   - 传入负数将减少计数
	Add(int)
	// GetSpeed 获取当前速率（事件数/秒）
	GetSpeed() float64
}

// defaultSpeedometer 滑动窗口计速器
//   - 将时间窗口划分为多个桶，每个桶独立计数，窗口滑动时自动淘汰过期桶
type defaultSpeedometer struct {
	windowSize   time.Duration // 期望的时间窗口长度（仅用于创建时）
	actualWindow time.Duration // 实际的时间窗口长度 = bucketSize * bucketNumber
	bucketSize   time.Duration // 每个桶的时间长度
	buckets      []int         // 环形桶数组
	bucketNumber int           // 桶的数量
	currentTotal int           // 当前窗口内的事件总数
	currentIndex int           // 当前写入的桶索引
	lastTime     time.Time     // 上次更新的时间戳

	clock timex.Clock // 时间源，便于测试注入

	lock sync.Mutex
}

// 编译期断言：滑窗计速器实现 Speedometer 接口
var _ Speedometer = (*defaultSpeedometer)(nil)

// New 新建计速器
//   - windowSize: 时间窗口长度，必须 > 0，否则使用默认值 2 秒
//   - 桶的数量缺省 7（建议取值 5~10），可通过 WithBucketNumber 指定
//   - 时间源缺省为系统时钟，可通过 WithClock 注入
func New(windowSize time.Duration, options ...Option) Speedometer {
	size := 2 * time.Second
	if windowSize > 0 {
		size = windowSize
	}

	own := &defaultSpeedometer{
		windowSize:   size,
		bucketNumber: _DEFAULT_BUCKET_NUMBER,
		clock:        timex.NewSystemClock(),
	}

	for _, option := range options {
		option(own)
	}

	// 确保 bucketSize >= 1ns
	// 实际窗口 = bucketSize * 桶数量，与期望窗口接近但不一定完全相等
	own.bucketSize = size / time.Duration(own.bucketNumber)
	if own.bucketSize == 0 {
		own.bucketSize = 1
	}
	own.actualWindow = own.bucketSize * time.Duration(own.bucketNumber)
	own.buckets = make([]int, own.bucketNumber)
	own.lastTime = own.clock.GetNow()

	return own
}

// Add 增加事件计数，可添加任意正整数
//   - 传入负数将减少计数，但一般不推荐
func (own *defaultSpeedometer) Add(n int) {
	own.lock.Lock()
	now := own.clock.GetNow()
	own.update(now)
	own.buckets[own.currentIndex] += n
	own.currentTotal += n
	own.lock.Unlock()
}

// GetSpeed 获取当前速率（事件数/秒）
func (own *defaultSpeedometer) GetSpeed() float64 {
	own.lock.Lock()
	now := own.clock.GetNow()
	own.update(now)
	speed := float64(own.currentTotal) / own.actualWindow.Seconds() // 使用实际窗口长度计算，更精确
	own.lock.Unlock()
	return speed
}

// update 根据当前时间滑动窗口，淘汰过期桶
//   - 需在持有锁时调用
func (own *defaultSpeedometer) update(now time.Time) {
	elapsed := now.Sub(own.lastTime)

	// 处理时间回拨：直接将 lastTime 设为当前时间，并清空所有数据（安全降级）
	if elapsed < 0 {
		own.resetAll(now)
		return
	}
	if elapsed == 0 {
		return
	}

	moveSteps := int(elapsed / own.bucketSize)
	if moveSteps == 0 {
		return
	}

	// 如果需要移动的步数超过桶数量，直接清空整个窗口
	if moveSteps >= own.bucketNumber {
		own.resetAll(now)
		return
	}

	// 逐步移动，淘汰过期桶
	for i := 0; i < moveSteps; i++ {
		own.currentIndex = (own.currentIndex + 1) % own.bucketNumber
		own.currentTotal -= own.buckets[own.currentIndex]
		own.buckets[own.currentIndex] = 0
	}

	// 更新 lastTime，只对齐到完整的桶边界，余数留待下次处理
	own.lastTime = own.lastTime.Add(time.Duration(moveSteps) * own.bucketSize)
}

// resetAll 清空所有桶，重置状态，并将 lastTime 设为当前时间
func (own *defaultSpeedometer) resetAll(now time.Time) {
	for i := range own.buckets {
		own.buckets[i] = 0
	}

	own.currentTotal = 0
	own.currentIndex = 0
	own.lastTime = now
}
