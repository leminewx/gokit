package serial

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

var testConfig = Config{
	Name:        "COM6",
	Baud:        115200,
	DataBits:    8,
	StopBits:    1,
	Parity:      'N',
	ReadTimeout: 2 * time.Second, // 默认读超时（当未设置 deadline 时使用）
	FlowControl: FLOW_CONTROL_NONE,
}

// TestConnection 测试串口连接的基本读写和超时行为
func TestConnection(t *testing.T) {
	conn, err := NewConnection(testConfig)
	if err != nil {
		t.Skipf("无法打开 COM6，请确认端口存在且未被占用: %v", err)
	}
	defer func() { _ = conn.Close() }()

	t.Logf("成功打开串口 %s", testConfig.Name)

	// 测试读超时（无数据时应该超时）
	t.Run("ReadTimeout", func(t *testing.T) {
		buf := make([]byte, 128)
		// 设置读 deadline 为 500ms 后
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		if err == nil {
			t.Errorf("期望超时错误，但读取到 %d 字节: %q", n, buf[:n])
			return
		} else if !errors.Is(err, ErrTimeout) {
			t.Errorf("期望 ErrTimeout，得到: %v", err)
		}

		// 检查是否为 net.Error 且 Timeout() 为 true
		if netErr, ok := err.(net.Error); ok {
			if !netErr.Timeout() {
				t.Errorf("错误应标记为 Timeout，但 Timeout() 返回 false")
			}
		} else {
			t.Errorf("错误未实现 net.Error 接口")
		}

		t.Logf("读超时测试通过: %v", err)
	})

	// 测试写操作（发送数据）和读响应
	t.Run("WriteAndRead", func(t *testing.T) {
		// 清除之前的 deadline
		conn.SetDeadline(time.Time{})

		// 发送 AT 指令（适用于调制解调器或环回设备）
		cmd := "AT\r\n"
		n, err := conn.Write([]byte(cmd))
		if err != nil {
			t.Fatalf("写入失败: %v", err)
		}
		t.Logf("已写入 %d 字节: %q", n, cmd)

		// 设置读 deadline 为 3 秒，等待响应
		buf := make([]byte, 256)
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, err = conn.Read(buf)
		if err != nil {
			if errors.Is(err, ErrTimeout) {
				t.Fatal("读取超时，可能设备无响应或未连接")
			}
			t.Fatalf("读取失败: %v", err)
		}
		t.Logf("收到响应 %d 字节: %q", n, buf[:n])
	})

	// 测试并发读写（可选，验证互斥锁是否正确）
	t.Run("Concurrent", func(t *testing.T) {
		// 简单测试：启动两个 goroutine 同时读写，但这里仅演示，不深入
		// 实际可扩展为更复杂的并发测试
		var wg sync.WaitGroup
		wg.Go(func() {
			for i := 0; i < 10; i++ {
				_, _ = conn.Write([]byte("test"))
				data := make([]byte, 128)
				_, _ = conn.Read(data)
				t.Logf("goroutine1: 已读到数据: %v", data)
			}
		})

		wg.Go(func() {
			for i := 0; i < 10; i++ {
				_, _ = conn.Write([]byte("test"))
				data := make([]byte, 128)
				_, _ = conn.Read(data)
				t.Logf("goroutine2: 已读到数据: %v", data)
			}
		})

		wg.Wait()
	})

	// 测试关闭后的行为
	t.Run("AfterClose", func(t *testing.T) {
		// 关闭连接
		_ = conn.Close()
		if _, err := conn.Write([]byte("test")); err == nil {
			t.Error("关闭后写入应返回错误")
		}
	})
}
