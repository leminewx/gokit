package codec

import (
	"net"
)

// Codec 编解码器接口
type Codec interface {
	// Decode 从网络连接中读取响应数据，并进行解码
	Decode(conn net.Conn) ([]byte, error)
	// Encode 对请求数据进行编码，并写入网络连接
	Encode(conn net.Conn, data []byte) error
}

// 编译期断言：全部内置编解码器实现 Codec 接口
var (
	_ Codec = (*CustomCodec)(nil)
	_ Codec = (*DelimiterCodec)(nil)
	_ Codec = (*EOFCodec)(nil)
	_ Codec = (*FixedLengthCodec)(nil)
	_ Codec = (*LengthFieldCodec)(nil)
	_ Codec = (*RawCodec)(nil)
)
