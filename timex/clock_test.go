package timex

import (
	"testing"
	"time"
)

// 编译期断言：系统时钟实现 Clock 接口
var _ Clock = NewSystemClock()

// TestSystemNow 验证系统时钟返回的当前时刻与真实时间接近
func TestSystemNow(t *testing.T) {
	t.Parallel()

	source := NewSystemClock()
	if got := time.Since(source.GetNow()); got > time.Second || got < -time.Second {
		t.Fatalf("NewSystemClock().GetNow() 与当前时间偏差过大: %v", got)
	}
}

// TestSystemTicker 验证系统时钟的嘀嗒器能正常触发并停止
func TestSystemTicker(t *testing.T) {
	t.Parallel()

	ticker := NewSystemClock().NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	select {
	case <-ticker.GetChannel():
	case <-time.After(2 * time.Second):
		t.Fatal("系统嘀嗒器未在超时时间内触发")
	}
}
