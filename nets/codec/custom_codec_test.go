package codec

import (
	"net"
	"testing"
)

func TestCustomCodec_EncodeDecode(t *testing.T) {
	codec := NewCustomCodec(
		func(conn net.Conn) ([]byte, error) {
			buf := make([]byte, 32)
			n, _ := conn.Read(buf)
			return buf[:n], nil
		},
		func(conn net.Conn, data []byte) error {
			_, err := conn.Write(data)
			return err
		},
	)

	payload := []byte("custom data")
	client, server := newFakeConn()

	go func() { codec.Encode(client, payload) }()

	data, err := codec.Decode(server)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if string(data) != "custom data" {
		t.Errorf("expected 'custom data', got '%s'", data)
	}
}

func TestCustomCodec_NilSafety(t *testing.T) {
	// nil 构造函数不应返回 nil
	codec := NewCustomCodec(nil, nil)
	if codec == nil {
		t.Fatal("NewCustomCodec with nil args should not return nil")
	}

	// nil 函数不应 panic
	client, server := newFakeConn()
	go func() { codec.Encode(client, []byte("test")) }()
	_, err := codec.Decode(server)
	if err != nil {
		t.Errorf("nil-safe decode should not error: %v", err)
	}
}
