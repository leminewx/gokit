// Package speedometer 提供基于环形时间桶的事件速率计。
//
// 速率按滑动窗口统计：窗口被拆分为若干桶，时间推进时自动淘汰过期桶，GetSpeed 返回
// 当前窗口内的事件数除以窗口秒数。Add 传入负数会减少计数。
//
// 实现内部加锁，可并发调用；时间源可注入，测试中使用 timex.Manual 驱动窗口滑动。
package speedometer
