package codec

import (
	"io"
	"testing"
	"time"
)

func TestDelimiterCodec_EncodeDecode(t *testing.T) {
	codec := NewDelimiterCodec('\n')
	payload := []byte("hello world")

	client, server := newFakeConn()

	go func() {
		if err := codec.Encode(client, payload); err != nil {
			t.Errorf("encode error: %v", err)
		}
	}()

	data, err := codec.Decode(server)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", data)
	}
}

func TestDelimiterCodec_MultipleMessages(t *testing.T) {
	codec := NewDelimiterCodec('\n')

	client, server := newFakeConn()

	go func() {
		codec.Encode(client, []byte("msg1"))
		time.Sleep(10 * time.Millisecond)
		codec.Encode(client, []byte("msg2"))
		time.Sleep(10 * time.Millisecond)
		codec.Encode(client, []byte("msg3"))
	}()

	// 验证每条消息都正确解码
	for i, expected := range []string{"msg1", "msg2", "msg3"} {
		data, err := codec.Decode(server)
		if err != nil {
			t.Fatalf("decode msg%d error: %v", i+1, err)
		}
		if string(data) != expected {
			t.Errorf("msg%d: expected '%s', got '%s'", i+1, expected, data)
		}
	}
}

func TestDelimiterCodec_DecodeEOF(t *testing.T) {
	codec := NewDelimiterCodec('\n')
	client, server := newFakeConn()

	go func() {
		client.writer.Write([]byte("partial data without newline"))
		client.writer.(*io.PipeWriter).Close()
	}()

	data, err := codec.Decode(server)
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
	if string(data) != "partial data without newline" {
		t.Errorf("expected partial data, got '%s'", data)
	}
}
