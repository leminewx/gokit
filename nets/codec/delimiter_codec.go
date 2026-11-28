package codec

import (
	"bufio"
	"io"
	"net"
)

// DelimiterCodec 分隔符编解码器
type DelimiterCodec struct {
	delim  byte
	reader *bufio.Reader
}

// NewDelimiterCodec 创建分隔符编解码器
// 首次 Decode 时会从 conn 创建 buffered reader 并复用
func NewDelimiterCodec(delim byte) Codec {
	return &DelimiterCodec{delim: delim}
}

func (own *DelimiterCodec) Decode(conn net.Conn) ([]byte, error) {
	if own.reader == nil {
		own.reader = bufio.NewReader(conn)
	}

	// 包含分隔符
	data, err := own.reader.ReadBytes(own.delim)
	if err != nil {
		// 读取结束或意外中断：已读到的数据仍然有效（含分隔符则去掉）
		stripped := stripDelimiter(data, own.delim)
		if err == io.EOF {
			return stripped, io.EOF
		}

		// 读取错误
		return nil, wrapReadError("delimiter", err)
	}

	return stripDelimiter(data, own.delim), nil
}

func stripDelimiter(data []byte, delim byte) []byte {
	if len(data) > 0 && data[len(data)-1] == delim {
		return data[:len(data)-1]
	}
	return data
}

func (own *DelimiterCodec) Encode(conn net.Conn, data []byte) error {
	// 确保消息以分隔符结尾
	if len(data) == 0 || data[len(data)-1] != own.delim {
		data = append(data, own.delim)
	}

	if _, err := conn.Write(data); err != nil {
		return wrapWriteError("delimiter", err)
	}

	return nil
}
