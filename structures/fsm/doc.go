// Package fsm 提供泛型有限状态机。
//
// 状态以可比较类型表示，转移由 AddTransition / UpsertTransition 注册，Trigger 触发状态
// 迁移并在进入、离开与事件回调中执行注入的函数。非法状态或转移返回对应哨兵错误，
// 回调 panic 会被捕获并转为 Err*Callback。
//
// 状态机本身不加锁，同一实例的 Trigger 必须串行调用；Export / Import 用于持久化快照。
package fsm
