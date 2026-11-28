// Package bloom 提供泛型布隆过滤器。
//
// 容量、期望假阳性率与哈希函数通过 Config 配置，元素通过 toBytes 转为字节后散列；
// Concurrent 为 true 时启用并发读保护，位数组本身以原子方式更新。
//
// 布隆过滤器只回答「可能存在」或「确定不存在」，不支持删除；GetSize、GetHashCount 与
// GetMemoryUsage 可用于观测内存占用。
package bloom
