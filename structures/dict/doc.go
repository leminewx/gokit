// Package dict 提供字典接口与四种实现：UnsafeDict（无锁）、SyncDict（sync.Map）、
// SafeDict（读写锁）与 ShardSafeDict（分片加锁）。
//
// 接口覆盖增删查、批量判断、键值快照、过滤与遍历；遍历通过 iter.Seq2 返回迭代器，
// 调用方可在 for-range 中直接使用。并发语义由实现决定：仅 UnsafeDict 非并发安全。
//
// 所有实现都不做键值深拷贝，取值返回的引用需由调用方自行保证不可变。
package dict
