package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestLengthFieldCodec_FixedBE_EncodeDecode(t *testing.T) {
	tests := []struct {
		name      string
		fieldSize int
		payload   string
	}{
		{"1byte", 1, "hello"},
		{"2bytes", 2, "hello world"},
		{"4bytes", 4, strings.Repeat("x", 256)},
		{"8bytes", 8, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := NewLengthFieldCodec(tt.fieldSize, binary.BigEndian, 0)
			payload := []byte(tt.payload)

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
			if string(data) != tt.payload {
				t.Errorf("expected '%s', got '%s'", tt.payload, data)
			}
		})
	}
}

func TestLengthFieldCodec_LittleEndian(t *testing.T) {
	codec := NewLengthFieldCodec(2, binary.LittleEndian, 0)
	payload := []byte("LE test")

	client, server := newFakeConn()
	go func() { codec.Encode(client, payload) }()

	data, err := codec.Decode(server)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if string(data) != "LE test" {
		t.Errorf("expected 'LE test', got '%s'", data)
	}
}

func TestLengthFieldCodec_Uvarint(t *testing.T) {
	codec := NewUvarintLengthCodec(0)
	payload := []byte("uvarint message")

	client, server := newFakeConn()
	go func() { codec.Encode(client, payload) }()

	data, err := codec.Decode(server)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if string(data) != "uvarint message" {
		t.Errorf("expected 'uvarint message', got '%s'", data)
	}
}

func TestLengthFieldCodec_EmptyPayload(t *testing.T) {
	codec := NewLengthFieldCodec(2, binary.BigEndian, 0)

	client, server := newFakeConn()
	go func() { codec.Encode(client, nil) }()

	data, err := codec.Decode(server)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil for empty payload, got %v", data)
	}
}

func TestLengthFieldCodec_MaxMessageSize(t *testing.T) {
	codec := NewLengthFieldCodec(2, binary.BigEndian, 10)
	payload := []byte(strings.Repeat("x", 20))

	client, server := newFakeConn()
	go func() { codec.Encode(client, payload) }()

	_, err := codec.Decode(server)
	if err == nil {
		t.Fatal("expected error for oversized message")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("expected 'too large' error, got: %v", err)
	}
}

// TestLengthFieldCodec_EncodeRejectsOverflow 验证定长长度字段无法表示负载长度时报错，而不是静默截断
func TestLengthFieldCodec_EncodeRejectsOverflow(t *testing.T) {
	tests := []struct {
		name      string
		fieldSize int
		payload   int
	}{
		{"1byte-256", 1, 256},
		{"2bytes-65536", 2, 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := NewLengthFieldCodec(tt.fieldSize, binary.BigEndian, 0)
			client, _ := newFakeConn()

			// 不启动对端读取：若实现仍然尝试写入，会因 io.Pipe 阻塞而暴露问题
			err := codec.Encode(client, make([]byte, tt.payload))
			if !errors.Is(err, ErrMessageTooLarge) {
				t.Fatalf("expected ErrMessageTooLarge, got %v", err)
			}
		})
	}
}

// TestLengthFieldCodec_EncodeBoundary 验证长度字段容量边界上的负载可以正常编解码
func TestLengthFieldCodec_EncodeBoundary(t *testing.T) {
	codec := NewLengthFieldCodec(1, binary.BigEndian, 0)
	payload := []byte(strings.Repeat("x", 255))

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
	if len(data) != len(payload) {
		t.Fatalf("payload length = %d, want %d", len(data), len(payload))
	}
}

// TestLengthFieldCodec_DecodeDefaultLimit 验证未显式指定上限时，超大长度直接在分配前被拒绝
func TestLengthFieldCodec_DecodeDefaultLimit(t *testing.T) {
	t.Run("fixed", func(t *testing.T) {
		codec := NewLengthFieldCodec(4, binary.BigEndian, 0)
		client, server := newFakeConn()

		go func() {
			var header [4]byte
			binary.BigEndian.PutUint32(header[:], 1<<30) // 1 GiB，远超默认 64 MiB
			_, _ = client.Write(header[:])
		}()

		_, err := codec.Decode(server)
		if !errors.Is(err, ErrMessageTooLarge) {
			t.Fatalf("expected ErrMessageTooLarge, got %v", err)
		}
	})

	t.Run("uvarint", func(t *testing.T) {
		codec := NewUvarintLengthCodec(0)
		client, server := newFakeConn()

		go func() {
			var header [binary.MaxVarintLen64]byte
			n := binary.PutUvarint(header[:], 1<<30)
			_, _ = client.Write(header[:n])
		}()

		_, err := codec.Decode(server)
		if !errors.Is(err, ErrMessageTooLarge) {
			t.Fatalf("expected ErrMessageTooLarge, got %v", err)
		}
	})
}

func TestLengthFieldCodec_MultipleMessages(t *testing.T) {
	codec := NewLengthFieldCodec(2, binary.BigEndian, 0)

	client, server := newFakeConn()

	go func() {
		for i := 0; i < 5; i++ {
			msg := []byte(fmt.Sprintf("msg-%d", i))
			codec.Encode(client, msg)
			time.Sleep(5 * time.Millisecond)
		}
	}()

	for i := 0; i < 5; i++ {
		expected := fmt.Sprintf("msg-%d", i)
		data, err := codec.Decode(server)
		if err != nil {
			t.Fatalf("decode msg%d error: %v", i, err)
		}
		if string(data) != expected {
			t.Errorf("msg%d: expected '%s', got '%s'", i, expected, data)
		}
	}
}
