package codec

import (
	"net"
)

// CustomCodec 自定义编解码器
type CustomCodec struct {
	// 解码函数
	DecodeFunc func(conn net.Conn) ([]byte, error)
	// 编码函数
	EncodeFunc func(conn net.Conn, data []byte) error
}

// NewCustomCodec 创建自定义编解码器，nil 函数字段将使用无操作实现
func NewCustomCodec(decodeFn func(net.Conn) ([]byte, error), encodeFn func(net.Conn, []byte) error) Codec {
	if decodeFn == nil {
		decodeFn = func(net.Conn) ([]byte, error) { return nil, nil }
	}
	if encodeFn == nil {
		encodeFn = func(net.Conn, []byte) error { return nil }
	}
	return &CustomCodec{DecodeFunc: decodeFn, EncodeFunc: encodeFn}
}

func (own *CustomCodec) Decode(conn net.Conn) ([]byte, error) {
	return own.DecodeFunc(conn)
}

func (own *CustomCodec) Encode(conn net.Conn, data []byte) error {
	return own.EncodeFunc(conn, data)
}
