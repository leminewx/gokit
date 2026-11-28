// Package sshx 提供 SSH 命令执行、流式执行与 PTY 终端。
//
// Client 支持可选的重连与超时配置：Execute 可并发调用，SetReconnect 等修改客户端状态
// 的方法不要与其他调用并发。流式执行通过 StreamHandler 回调交付 stdin/stdout/stderr
// 事件，回调实现需自行保证并发安全。
//
// 调试日志只输出命令长度等元信息，不记录口令与完整输入；会话与连接在 Close 后不可复用。
package sshx
