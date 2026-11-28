//go:build integration

// 设备依赖测试：需要真实串口回环设备才能运行，默认不参与构建与测试。
//
// 执行前需设置环境变量 GOKIT_SERIAL_PORT（例如 COM6 或 /dev/ttyUSB0），
// 未设置时测试会被跳过。
//
// 执行方式：go test -tags integration ./nets/serial

package serial

import (
	"os"
	"testing"
	"time"
)

func TestPortLoopback(t *testing.T) {
	name := os.Getenv("GOKIT_SERIAL_PORT")
	if name == "" {
		t.Skip("未配置 GOKIT_SERIAL_PORT，跳过串口集成测试")
	}

	port, err := Open(Config{
		Name:        name,
		Baud:        115200,
		DataBits:    8,
		StopBits:    1,
		Parity:      'N',
		ReadTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("打开串口 %s 失败: %v", name, err)
	}
	defer func() { _ = port.Close() }()

	if _, err := port.Write([]byte("gokit-serial\r\n")); err != nil {
		t.Fatalf("写入串口失败: %v", err)
	}

	buf := make([]byte, 128)
	n, err := port.Read(buf)
	if err != nil {
		t.Fatalf("读取串口失败: %v", err)
	}
	t.Logf("收到回环数据: %q", buf[:n])
}
