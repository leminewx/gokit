// Package ini 提供 INI 与 JSON 配置的加载、类型化访问、序列化与保存。
//
// 支持从文件、io.Reader 与字符串载入，键名按节（section）组织；类型化取值在转换失败时
// 返回 ErrConvertFailed，键不存在返回 ErrKeyNotFound。Unmarshal 把配置映射到结构体字段，
// Marshal / Save 负责写回，写入走同目录临时文件再原子改名的路径。
//
// 本包不缓存文件内容，也不监听配置变更；需要热更新时由调用方重新 Load。
package ini
