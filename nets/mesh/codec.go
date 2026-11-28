package mesh

import (
	"encoding/json"
	"io"
)

// ******************************************************************************************************************
// Codec：消息编解码器
//
// 每个连接独立创建编解码器实例，编解码器必须满足以下约束：
//   - 编码器与解码器必须成对使用（同一 Codec 返回的两个函数互为逆操作）
//   - 必须自带消息边界（如 JSON 的换行分隔、gob 的长度前缀），保证流式读写不会粘包
//   - 必须无状态，仅依赖调用时传入的 io.Writer / io.Reader
//
// 缺省使用 NewJSONCodec（换行分隔的 JSON 对象流），适合调试与跨语言互通；
// 追求吞吐与带宽时可注入自定义实现（如 msgpack、protobuf、gob）。
// ******************************************************************************************************************

// Codec 定义消息编解码器
type Codec interface {
	// GetEncoder 为指定写入流创建编码函数
	GetEncoder(writer io.Writer) func(msg Message) error
	// GetDecoder 为指定读取流创建解码函数
	GetDecoder(reader io.Reader) func(msg *Message) error
}

// NewJSONCodec 基于 JSON 的编解码器（缺省）
func NewJSONCodec() Codec {
	return jsonCodec{}
}

// jsonCodec JSON 编解码器
type jsonCodec struct{}

// GetEncoder 创建 JSON 编码函数
//   - json.Encoder 以换行分隔每条消息，天然具备消息边界
func (jsonCodec) GetEncoder(writer io.Writer) func(msg Message) error {
	encoder := json.NewEncoder(writer)

	return func(msg Message) error {
		return encoder.Encode(msg)
	}
}

// GetDecoder 创建 JSON 解码函数
func (jsonCodec) GetDecoder(reader io.Reader) func(msg *Message) error {
	decoder := json.NewDecoder(reader)

	return func(msg *Message) error {
		return decoder.Decode(msg)
	}
}
