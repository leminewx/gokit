// Package queue 提供基于最小堆的延迟队列。
//
// 任务按到期时间排序，AddTask 入队后可在到期前 CancelTask 撤销；队列内部对堆与任务表
// 分别加锁，AddTask / CancelTask / 消费可并发调用。Stop 后队列拒绝新任务并释放后台协程。
//
// 到期判断依赖注入的时钟（默认系统时钟），测试可注入 timex.Manual 避免真实等待；
// 任务回调在消费协程中执行，回调 panic 会被捕获并转为错误上报。
package queue
