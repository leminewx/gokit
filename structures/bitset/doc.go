// Package bitset 提供位集合与并、交、差、补运算。
//
// 位集合按需扩容，Set / Clear 返回自身以支持链式调用；Union / Intersect / Subtract
// 生成新集合，不修改操作数。越界下标会被自动扩容或忽略，具体语义以各方法注释为准。
//
// 本包不保证并发安全，同一实例的并发读写需由调用方串行化。
package bitset
