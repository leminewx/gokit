package codec

import (
	"io"
	"testing"
)

func TestRawCodec_EncodeDecode(t *testing.T) {
	codec := NewRawCodec(1024)
	payload := []byte("hello world")

	client, server := newFakeConn()

	// encode in goroutine
	go func() {
		if err := codec.Encode(client, payload); err != nil {
			t.Errorf("encode error: %v", err)
		}
	}()

	// decode
	data, err := codec.Decode(server)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", data)
	}
}

func TestRawCodec_DecodeEOF(t *testing.T) {
	codec := NewRawCodec(64)
	client, server := newFakeConn()
	client.writer.(*io.PipeWriter).Close()

	_, err := codec.Decode(server)
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}
