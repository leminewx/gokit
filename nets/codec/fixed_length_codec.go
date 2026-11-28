package codec

import (
	"io"
	"net"
)

// FixedLengthCodec 固定长度编解码器
type FixedLengthCodec struct {
	size int
}

// NewFixedLengthCodec 创建固定长度编解码器
func NewFixedLengthCodec(size int) Codec {
	return &FixedLengthCodec{size: size}
}

func (own *FixedLengthCodec) Decode(conn net.Conn) ([]byte, error) {
	if own.size <= 0 {
		return nil, wrapError(ErrInvalidLength, "invalid fixed-length size %d", own.size)
	}

	data := make([]byte, own.size)

	// 阻塞读满
	n, err := io.ReadFull(conn, data)
	if err != nil {
		// 完全无数据可读
		if err == io.EOF {
			return nil, io.EOF
		}

		// 读取了部分数据但连接中断
		if err == io.ErrUnexpectedEOF {
			return data[:n], io.EOF
		}

		// 读取错误
		return nil, wrapReadError("fixed", err)
	}

	return data, nil
}

func (own *FixedLengthCodec) Encode(conn net.Conn, data []byte) error {
	if own.size <= 0 {
		return wrapError(ErrInvalidLength, "invalid fixed-length size %d", own.size)
	}

	if len(data) != own.size {
		if len(data) < own.size {
			// 充填
			padded := make([]byte, own.size)
			copy(padded, data)
			data = padded
		} else {
			// 截断
			data = data[:own.size]
		}
	}

	if _, err := conn.Write(data); err != nil {
		return wrapWriteError("fixed", err)
	}

	return nil
}
