// Package serial 提供跨平台串口访问，连接类型实现 net.Conn 的读写与截止时间语义。
//
// 串口通过 Config 描述端口、波特率、数据位、停止位与校验方式；读写支持超时，
// 超时错误可用 errors.Is(err, ErrTimeout) 判定。平台不支持串口时返回 ErrUnsupported。
//
// 本包不做粘包处理，字节流分帧请配合 nets/codec 使用；连接不做并发保护，
// 同一连接上的并发读写需由调用方串行化。
package serial
