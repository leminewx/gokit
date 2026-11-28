package codec

import (
	"io"
	"net"
)

// RawCodec 原始编解码器（不做任何编解码处理）
type RawCodec struct {
	readSize int
}

func NewRawCodec(readSize int) Codec {
	return &RawCodec{readSize: readSize}
}

func (own *RawCodec) Decode(conn net.Conn) ([]byte, error) {
	if own.readSize <= 0 {
		return nil, wrapError(ErrInvalidLength, "invalid raw read size %d", own.readSize)
	}

	data := make([]byte, own.readSize)
	n, err := conn.Read(data)
	if err != nil {
		// 读取结束
		if err == io.EOF {
			return data[:n], io.EOF
		}

		// 读取错误
		return nil, wrapReadError("raw", err)
	}

	return data[:n], nil
}

func (own *RawCodec) Encode(conn net.Conn, data []byte) error {
	if _, err := conn.Write(data); err != nil {
		return wrapWriteError("raw", err)
	}

	return nil
}
