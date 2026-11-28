package codec

import (
	"io"
	"testing"
)

func TestEOFCodec_EncodeDecode(t *testing.T) {
	codec := NewEOFCodec()
	payload := []byte("one shot message")

	client, server := newFakeConn()

	go func() {
		// 有 CloseWrite 行为的写入需要特殊处理，这里直接写入后关闭
		client.writer.Write(payload)
		// 模拟 CloseWrite: 关闭 writer
		if pw, ok := client.writer.(*io.PipeWriter); ok {
			pw.Close()
		}
	}()

	data, err := codec.Decode(server)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if string(data) != "one shot message" {
		t.Errorf("expected 'one shot message', got '%s'", data)
	}
}
