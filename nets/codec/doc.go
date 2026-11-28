// Package codec 提供基于 net.Conn 的消息分帧编解码：原始字节流、定长、分隔符、EOF、
// 长度前缀、uvarint 长度与自定义实现。
//
// 编解码器只负责「一次读取/写入一条完整消息」，不处理重连与并发；长度前缀类实现会校验
// 上限与字段宽度，超限返回 ErrMessageTooLarge 或 ErrUnsupportedLengthField。
//
// 自定义解码可传入函数实现，便于对接私有协议。
package codec
