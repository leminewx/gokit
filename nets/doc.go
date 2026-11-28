// Package nets 提供网络层通用工具：连接错误判定、CIDR 网段枚举与本机出口 IP 优选。
//
// 连接错误判定基于错误链与底层 syscall 错误，用于区分「对端关闭」「超时」与普通业务
// 错误；IP 枚举对超大网段有数量上限，超过时返回 ErrTooManyAddresses。
//
// 具体的 TCP 客户端、编解码、串口、SSH 与 SMTP 能力见对应子包。
package nets
