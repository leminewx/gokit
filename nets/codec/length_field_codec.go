package codec

import (
	"encoding/binary"
	"io"
	"math"
	"net"
)

// _DEFAULT_MAX_MESSAGE_SIZE 未显式指定上限时的解码上限（64 MiB）。
// 长度字段由对端提供，完全不限制时只靠一个长度字段就能触发巨额 make 分配。
const _DEFAULT_MAX_MESSAGE_SIZE = 64 << 20

// LengthFieldCodec 长度字段编解码器
//
// 帧格式: [长度字段(1/2/4/8字节)] + [数据负载]
//   - 长度字段表示数据负载的字节数（不包含长度字段自身）。
//
// 支持三种长度字段编码方式:
//   - binary.BigEndian    (默认)
//   - binary.LittleEndian
//   - uvarint 变长编码
type LengthFieldCodec struct {
	// 长度字段的字节数: 1, 2, 4, 或 8。设为 0 表示使用 uvarint 变长编码
	lengthFieldSize int
	// 字节序，仅 lengthFieldSize > 0 时有效
	byteOrder binary.ByteOrder
	// 最大消息大小（字节），<=0 时由构造函数置为默认上限
	maxMessageSize int
}

// NewLengthFieldCodec 创建长度字段编解码器
//   - lengthFieldSize: 长度字段字节数 (1/2/4/8)
//   - byteOrder: 字节序 (binary.BigEndian, binary.LittleEndian)
//   - maxMessageSize: 最大消息体大小，<=0 时使用默认上限 64 MiB（如需不限制请传 math.MaxInt）
func NewLengthFieldCodec(lengthFieldSize int, byteOrder binary.ByteOrder, maxMessageSize int) Codec {
	if byteOrder == nil {
		byteOrder = binary.BigEndian
	}
	if lengthFieldSize < 0 {
		lengthFieldSize = 0
	}
	if maxMessageSize <= 0 {
		maxMessageSize = _DEFAULT_MAX_MESSAGE_SIZE
	}

	return &LengthFieldCodec{
		lengthFieldSize: lengthFieldSize,
		byteOrder:       byteOrder,
		maxMessageSize:  maxMessageSize,
	}
}

// NewUvarintLengthCodec 创建变长长度字段编解码器（uvarint 编码）
//   - maxMessageSize: 最大消息体大小，<=0 时使用默认上限 64 MiB（如需不限制请传 math.MaxInt）
func NewUvarintLengthCodec(maxMessageSize int) Codec {
	if maxMessageSize <= 0 {
		maxMessageSize = _DEFAULT_MAX_MESSAGE_SIZE
	}

	return &LengthFieldCodec{
		lengthFieldSize: 0, // 0 表示 uvarint
		maxMessageSize:  maxMessageSize,
	}
}

func (own *LengthFieldCodec) Decode(conn net.Conn) ([]byte, error) {
	var payloadLen int
	var err error

	if own.lengthFieldSize == 0 {
		// uvarint 变长编码
		payloadLen, err = own.readUvarint(conn)
	} else {
		// 固定长度字段
		payloadLen, err = own.readFixedLength(conn)
	}
	if err != nil {
		return nil, err
	}

	if payloadLen < 0 {
		return nil, wrapError(ErrInvalidLength, "negative payload length %d", payloadLen)
	}
	if own.maxMessageSize > 0 && payloadLen > own.maxMessageSize {
		return nil, wrapError(ErrMessageTooLarge, "payload length %d exceeds limit %d", payloadLen, own.maxMessageSize)
	}

	// 无负载
	if payloadLen == 0 {
		return nil, nil
	}

	data := make([]byte, payloadLen)
	n, err := io.ReadFull(conn, data)
	if err != nil {
		if err == io.EOF {
			return data[:n], io.EOF
		}

		if err == io.ErrUnexpectedEOF {
			return data[:n], io.EOF
		}

		return nil, wrapReadError("length-field", err)
	}

	return data, nil
}

func (own *LengthFieldCodec) Encode(conn net.Conn, data []byte) error {
	payloadLen := len(data)

	// 定长长度字段必须先确认能装下负载长度：否则 byte/uint16 截断会写出
	// 与真实负载不符的长度，对端按截断值解析会造成静默错帧。
	if own.lengthFieldSize > 0 {
		capacity := maxLengthForFieldSize(own.lengthFieldSize)
		if capacity == 0 {
			return wrapError(ErrUnsupportedLengthField, "%d", own.lengthFieldSize)
		}
		if uint64(payloadLen) > capacity {
			return wrapError(ErrMessageTooLarge, "payload length %d exceeds %d-byte length field capacity %d",
				payloadLen, own.lengthFieldSize, capacity)
		}
	}

	var buf []byte
	if own.lengthFieldSize == 0 {
		// uvarint 变长编码
		var header [binary.MaxVarintLen64]byte
		n := binary.PutUvarint(header[:], uint64(payloadLen))
		buf = make([]byte, n+payloadLen)
		copy(buf, header[:n])
		copy(buf[n:], data)
	} else {
		buf = make([]byte, own.lengthFieldSize+payloadLen)
		switch own.lengthFieldSize {
		case 1:
			buf[0] = byte(payloadLen)
		case 2:
			own.byteOrder.PutUint16(buf, uint16(payloadLen))
		case 4:
			own.byteOrder.PutUint32(buf, uint32(payloadLen))
		case 8:
			own.byteOrder.PutUint64(buf, uint64(payloadLen))
		default:
			return wrapError(ErrUnsupportedLengthField, "%d", own.lengthFieldSize)
		}
		copy(buf[own.lengthFieldSize:], data)
	}

	if _, err := conn.Write(buf); err != nil {
		return wrapWriteError("length-field", err)
	}

	return nil
}

func (own *LengthFieldCodec) readFixedLength(conn net.Conn) (int, error) {
	var header [8]byte
	if _, err := io.ReadFull(conn, header[:own.lengthFieldSize]); err != nil {
		if err == io.EOF {
			return 0, err
		}

		return 0, wrapReadError("length-field", err)
	}

	var length uint64
	switch own.lengthFieldSize {
	case 1:
		length = uint64(header[0])
	case 2:
		length = uint64(own.byteOrder.Uint16(header[:2]))
	case 4:
		length = uint64(own.byteOrder.Uint32(header[:4]))
	case 8:
		length = own.byteOrder.Uint64(header[:8])
	default:
		return 0, wrapError(ErrUnsupportedLengthField, "%d", own.lengthFieldSize)
	}

	// 32 位平台上 int 只有 32 位，直接转换会把 0x1_0000_0000 截成 0（被当作空帧，
	// 进而与后续数据错位），因此先做平台可表示性校验。
	if length > math.MaxInt {
		return 0, wrapError(ErrInvalidLength, "length %d exceeds platform int", length)
	}

	return int(length), nil
}

// maxLengthForFieldSize 返回定长长度字段能表示的最大负载长度；不支持的宽度返回 0。
func maxLengthForFieldSize(size int) uint64 {
	switch size {
	case 1:
		return math.MaxUint8
	case 2:
		return math.MaxUint16
	case 4:
		return math.MaxUint32
	case 8:
		return math.MaxUint64
	default:
		return 0
	}
}

func (own *LengthFieldCodec) readUvarint(conn net.Conn) (int, error) {
	// 逐字节读取 uvarint（net.Conn 不支持 io.ByteReader）
	var buf [1]byte
	var x uint64
	var s uint
	for i := 0; i < binary.MaxVarintLen64; i++ {
		if _, err := io.ReadFull(conn, buf[:]); err != nil {
			return 0, wrapReadError("length-field", err)
		}

		b := buf[0]
		if b < 0x80 {
			if i == binary.MaxVarintLen64-1 && b > 1 {
				return 0, wrapError(ErrInvalidVarint, "uvarint overflow")
			}
			return int(x | uint64(b)<<s), nil
		}

		x |= uint64(b&0x7f) << s
		s += 7
	}

	return 0, wrapError(ErrInvalidVarint, "uvarint value is too long")
}
