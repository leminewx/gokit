// Package tree 提供泛型 Trie 前缀树。
//
// 路径以分隔符切分，支持 `:` 占位与通配符提取路径参数：Insert 写入模式，Search 返回
// 匹配值、参数表与是否存在。分隔符与通配符可通过 SetDelim / SetWildcard 定制。
//
// 树本身不加锁，并发读写需由调用方同步；模式冲突按后写入覆盖，不做歧义检测。
package tree
