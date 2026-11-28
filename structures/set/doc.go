// Package set 提供集合接口与三种实现：UnsafeSet（无锁）、SafeSet（读写锁）与
// SyncSet（sync.Map）。
//
// 接口覆盖增删、包含判断、集合运算（并 / 交 / 差 / 互补）、过滤、遍历与克隆；集合运算
// 生成新集合，不修改操作数。并发语义由实现决定：仅 UnsafeSet 非并发安全。
//
// 集合只存可比较的元素，不做深拷贝；GetMap / GetSlice 返回快照副本。
package set
