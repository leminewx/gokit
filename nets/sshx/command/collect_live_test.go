//go:build integration

// 真实设备采集测试：通过环境变量提供目标，默认不参与构建与测试。
//
// 环境变量：
//   - GOKIT_SSH_HOST      目标主机（必填，缺省时跳过）
//   - GOKIT_SSH_USER      SSH 用户名
//   - GOKIT_SSH_PASSWORD  SSH 密码（亦兼容 GOKIT_SSH_PASS）
//   - GOKIT_SSH_PORT      端口，默认 22
//   - GOKIT_SSH_INTERVAL  采样间隔，默认 2s
//
// 执行方式（PowerShell）：
//
//	$env:GOKIT_SSH_HOST="192.0.2.2"
//	$env:GOKIT_SSH_USER="user01"
//	$env:GOKIT_SSH_PASSWORD="******"
//	go test -tags integration -run TestCollectLive -v ./nets/sshx/command

package command

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"gitee.com/leminewx/gokit/nets/sshx"
)

func TestCollectLive(t *testing.T) {
	host := os.Getenv("GOKIT_SSH_HOST")
	if host == "" {
		t.Skip("未设置 GOKIT_SSH_HOST，跳过真实设备采集测试")
	}

	user := os.Getenv("GOKIT_SSH_USER")
	password := os.Getenv("GOKIT_SSH_PASSWORD")
	if password == "" {
		password = os.Getenv("GOKIT_SSH_PASS") // 与 nets/sshx 的变量名保持一致
	}

	port := 22
	if value := os.Getenv("GOKIT_SSH_PORT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			t.Fatalf("GOKIT_SSH_PORT 非法：%v", err)
		}
		port = parsed
	}

	interval := 2 * time.Second
	if value := os.Getenv("GOKIT_SSH_INTERVAL"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			t.Fatalf("GOKIT_SSH_INTERVAL 非法：%v", err)
		}
		interval = parsed
	}

	client, err := sshx.NewClient(host, user, password,
		sshx.WithPort(port),
		sshx.WithTimeout(10*time.Second),
		sshx.WithReconnect(0),
	)
	if err != nil {
		t.Fatalf("连接 %s 失败：%v", host, err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	started := time.Now()
	snapshot, err := NewCollector(client, WithSampleInterval(interval)).Collect(ctx)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("采集 %s 失败：%v", host, err)
	}

	t.Logf("%s 采集耗时 %s\n%s", host, elapsed, snapshot.Format())

	// 各段数据都应有效：这里只校验必要条件，避免不同设备的挂载点差异导致误报
	if snapshot.Host == "" {
		t.Error("主机名为空")
	}
	if snapshot.Kernel == "" {
		t.Error("内核版本为空")
	}
	if snapshot.CPUCores <= 0 {
		t.Errorf("CPU 核心数 = %d", snapshot.CPUCores)
	}
	if snapshot.Memory.TotalBytes == 0 {
		t.Error("内存总量为 0")
	}
	if snapshot.Memory.UsedPercent < 0 || snapshot.Memory.UsedPercent > 100 {
		t.Errorf("内存使用率越界 = %.2f", snapshot.Memory.UsedPercent)
	}
	if used := snapshot.CPU.Used + snapshot.CPU.Idle + snapshot.CPU.IOWait; used < 99 || used > 101 {
		t.Errorf("CPU 使用率与空闲率之和 = %.2f, want ≈100", used)
	}
	if len(snapshot.CoreUsage) == 0 {
		t.Error("缺少逐核使用率")
	}
	if len(snapshot.Filesystems) == 0 {
		t.Error("缺少磁盘信息")
	}
	if len(snapshot.Networks) == 0 {
		t.Error("缺少网卡信息")
	}
	if snapshot.Uptime <= 0 {
		t.Error("运行时长应大于 0")
	}
}
