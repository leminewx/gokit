// Package timewheel 实现单级时间轮调度器。
//
// 时间轮由固定数量的槽位组成，每个槽位对应一个滴答间隔；事件循环按滴答推进指针，
// 到期槽位中的任务被投递到独立 goroutine 执行，因此任务回调不会阻塞时间轮转动。
//
// 典型用法：
//
//	wheel := timewheel.NewSingleWheel(100*time.Millisecond, 60)
//	wheel.Start()
//	defer wheel.Stop()
//
//	id := wheel.AddTask(time.Now().Add(time.Second), func() { /* do something */ }, 0)
//	wheel.CancelTask(id, nil)
package timewheel
