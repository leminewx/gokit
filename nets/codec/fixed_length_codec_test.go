package codec

import (
	"io"
	"testing"
)

func TestFixedLengthCodec_EncodeDecode(t *testing.T) {
	codec := NewFixedLengthCodec(8)
	payload := []byte("12345678")

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
	if len(data) != 8 || string(data) != "12345678" {
		t.Errorf("expected '12345678' (len=8), got '%s' (len=%d)", data, len(data))
	}
}

func TestFixedLengthCodec_EncodePad(t *testing.T) {
	codec := NewFixedLengthCodec(8)
	payload := []byte("abc")

	client, server := newFakeConn()
	go func() { codec.Encode(client, payload) }()

	data, _ := codec.Decode(server)
	if len(data) != 8 {
		t.Errorf("expected padded to 8 bytes, got %d", len(data))
	}
	if string(data[:3]) != "abc" {
		t.Errorf("expected 'abc' prefix, got '%s'", data[:3])
	}
}

func TestFixedLengthCodec_EncodeTruncate(t *testing.T) {
	codec := NewFixedLengthCodec(4)
	payload := []byte("12345678")

	client, server := newFakeConn()
	go func() { codec.Encode(client, payload) }()

	data, _ := codec.Decode(server)
	if len(data) != 4 {
		t.Errorf("expected truncated to 4 bytes, got %d", len(data))
	}
}

func TestFixedLengthCodec_DecodeEOF(t *testing.T) {
	codec := NewFixedLengthCodec(16)
	client, server := newFakeConn()
	client.writer.(*io.PipeWriter).Close()

	data, err := codec.Decode(server)
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data on EOF, got %v", data)
	}
}
