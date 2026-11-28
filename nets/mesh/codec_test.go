package mesh

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

// TestJSONCodec_RoundTrip 验证缺省编解码器的往返一致性
func TestJSONCodec_RoundTrip(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	encode := NewJSONCodec().GetEncoder(&buffer)

	messages := []Message{
		{SN: 1, Type: MESSAGE_TYPE_PUB, SenderID: "node-1", Path: "/a", Value: "hello"},
		{SN: 2, Type: MESSAGE_TYPE_PUT, SenderID: "node-2", Path: "/b", Value: 42.0},
		{SN: 3, Type: MESSAGE_TYPE_GOSSIP_HEARTBEAT, SenderID: "node-3"},
	}
	for _, msg := range messages {
		if err := encode(msg); err != nil {
			t.Fatalf("编码失败: %v", err)
		}
	}

	decode := NewJSONCodec().GetDecoder(&buffer)
	for index, want := range messages {
		var got Message
		if err := decode(&got); err != nil {
			t.Fatalf("第 %d 条消息解码失败: %v", index, err)
		}

		if got.SN != want.SN || got.Type != want.Type || got.SenderID != want.SenderID || got.Path != want.Path {
			t.Fatalf("第 %d 条消息不一致: %+v, 期望 %+v", index, got, want)
		}
	}

	// 流已耗尽
	var extra Message
	if err := decode(&extra); !errors.Is(err, io.EOF) {
		t.Fatalf("流耗尽后应返回 EOF, 实际 %v", err)
	}
}

// TestTCPTransport_CustomCodec 验证自定义编解码器可被注入并完成真实通信
func TestTCPTransport_CustomCodec(t *testing.T) {
	config1 := NewDefaultConfig()
	WithCodec(lengthPrefixCodec{})(config1)

	config2 := NewDefaultConfig()
	WithCodec(lengthPrefixCodec{})(config2)

	transport1 := NewTCPTransport("127.0.0.1:0", 10, config1)
	transport2 := NewTCPTransport("127.0.0.1:0", 10, config2)
	defer transport1.Close()
	defer transport2.Close()

	if err := transport1.Start(); err != nil {
		t.Fatalf("transport1 Start: %v", err)
	}
	if err := transport2.Start(); err != nil {
		t.Fatalf("transport2 Start: %v", err)
	}

	addr2 := getTransportAddr(transport2)
	if addr2 == "" {
		t.Fatal("无法获取 transport2 的监听地址")
	}

	var (
		received   Message
		receivedMu sync.Mutex
		done       = make(chan struct{})
	)
	transport2.AddHandler(&testMessageHandler{
		handleFunc: func(msg Message, fromAddr string) {
			receivedMu.Lock()
			received = msg
			receivedMu.Unlock()
			close(done)
		},
	})

	transport1.AddPeer(addr2)

	deadline := time.After(3 * time.Second)
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()

	// 连接建立是异步的，轮询重试发送
	for {
		err := transport1.SendTo(addr2, Message{
			SN:       1,
			Type:     MESSAGE_TYPE_PUB,
			SenderID: "codec-node-1",
			Path:     "/codec/test",
			Value:    "length-prefixed",
		})
		if err == nil {
			break
		}

		select {
		case <-deadline:
			t.Fatalf("发送失败: %v", err)
		case <-tick.C:
		}
	}

	select {
	case <-done:
		receivedMu.Lock()
		defer receivedMu.Unlock()

		if received.Path != "/codec/test" || received.Value != "length-prefixed" {
			t.Fatalf("消息内容错误: %+v", received)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("等待自定义编解码器消息超时")
	}
}

// lengthPrefixCodec 演示自定义编解码器：4 字节长度前缀 + JSON 载荷
//   - 长度前缀自带消息边界，避免流式读写粘包
type lengthPrefixCodec struct{}

func (lengthPrefixCodec) GetEncoder(writer io.Writer) func(Message) error {
	return func(msg Message) error {
		payload, err := json.Marshal(msg)
		if err != nil {
			return err
		}

		var header [4]byte
		binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
		if _, err := writer.Write(header[:]); err != nil {
			return err
		}

		_, err = writer.Write(payload)

		return err
	}
}

func (lengthPrefixCodec) GetDecoder(reader io.Reader) func(*Message) error {
	return func(msg *Message) error {
		var header [4]byte
		if _, err := io.ReadFull(reader, header[:]); err != nil {
			return err
		}

		payload := make([]byte, binary.BigEndian.Uint32(header[:]))
		if _, err := io.ReadFull(reader, payload); err != nil {
			return err
		}

		return json.Unmarshal(payload, msg)
	}
}
