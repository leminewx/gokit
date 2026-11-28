package codec

import (
	"io"
	"net"
)

// EOFCodec EOF 编解码器（一次性发送和接收）
type EOFCodec struct{}

// NewEOFCodec 创建 EOF 编解码器
func NewEOFCodec() Codec {
	return &EOFCodec{}
}

func (own *EOFCodec) Decode(conn net.Conn) ([]byte, error) {
	// 注意：会阻塞直到EOF或错误
	data, err := io.ReadAll(conn)
	if err != nil {
		// io.ReadAll 不会返回 io.EOF，此分支为真正的读取错误
		return data, wrapReadError("eof", err)
	}

	return data, nil
}

func (own *EOFCodec) Encode(conn net.Conn, data []byte) error {
	if _, err := conn.Write(data); err != nil {
		return wrapWriteError("eof", err)
	}

	// 关闭写端以通知对端数据已发送完毕
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if err := tcpConn.CloseWrite(); err != nil {
			return wrapWriteError("eof", err)
		}
	}

	return nil
}
