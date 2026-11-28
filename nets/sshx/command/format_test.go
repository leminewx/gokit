package command

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHumanBytes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value uint64
		want  string
	}{
		{"零值", 0, "0 B"},
		{"不足 1 KiB", 1023, "1023 B"},
		{"恰好 1 KiB", 1 << 10, "1.00 KiB"},
		{"1.5 KiB", 1536, "1.50 KiB"},
		{"1 GiB", 1 << 30, "1.00 GiB"},
		{"超过 int64 表示范围时截断", ^uint64(0), humanBytes(1 << 62)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := humanBytes(tc.value); got != tc.want {
				t.Errorf("humanBytes(%d) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   time.Duration
		want string
	}{
		{"零值", 0, "0s"},
		{"负数", -time.Second, "0s"},
		{"仅秒", 45 * time.Second, "45s"},
		{"分秒", 38*time.Minute + 5*time.Second, "38m5s"},
		{"时分秒", 4*time.Hour + 38*time.Minute + 5*time.Second, "4h38m5s"},
		{"天", 25*24*time.Hour + 4*time.Hour + 38*time.Minute, "25d4h38m"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := formatDuration(tc.in); got != tc.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSortedCoreNames(t *testing.T) {
	t.Parallel()

	cores := map[string]CPUUsage{
		"cpu10": {},
		"cpu2":  {},
		"cpu0":  {},
		"cpu1":  {},
		"node":  {},
	}

	want := []string{"cpu0", "cpu1", "cpu2", "cpu10", "node"}
	got := sortedCoreNames(cores)
	if len(got) != len(want) {
		t.Fatalf("sortedCoreNames() = %v, want %v", got, want)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("sortedCoreNames() = %v, want %v（核心应按编号排序）", got, want)
		}
	}
}

func TestValueStringers(t *testing.T) {
	t.Parallel()

	if got := (LoadAverage{One: 0.26, Five: 0.37, Fifteen: 0.44}).String(); got != "0.26 0.37 0.44" {
		t.Errorf("LoadAverage.String() = %q", got)
	}

	usage := CPUUsage{Used: 12.5, User: 8, System: 4.5, Idle: 87.5}
	if got := usage.String(); !strings.Contains(got, "used 12.5%") || !strings.Contains(got, "idle 87.5%") {
		t.Errorf("CPUUsage.String() = %q", got)
	}

	memory := MemoryStats{TotalBytes: 8 << 30, AvailableBytes: 4 << 30, UsedPercent: 50}
	if got := memory.String(); got != "used 50.00% | available 4.00 GiB / 8.00 GiB" {
		t.Errorf("MemoryStats.String() = %q", got)
	}
}

// buildSnapshot 构造一份覆盖全部字段的快照，供格式化测试复用
func buildSnapshot() *Snapshot {
	return &Snapshot{
		Host:           "host-a",
		Kernel:         "Linux 5.15.148-tegra aarch64",
		CollectedAt:    time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC),
		SampleInterval: time.Second,
		Uptime:         25*24*time.Hour + 4*time.Hour + 38*time.Minute,
		CPUCores:       8,
		Load:           LoadAverage{One: 0.5, Five: 0.4, Fifteen: 0.3, Running: 2, Total: 200},
		CPU:            CPUUsage{Used: 12.5, User: 8, System: 4.5, Idle: 87.5},
		Memory: MemoryStats{
			TotalBytes:         8 << 30,
			AvailableBytes:     4 << 30,
			UsedBytes:          4 << 30,
			BuffersBytes:       1 << 20,
			CachedBytes:        2 << 20,
			UsedPercent:        50,
			AvailableEstimated: true,
		},
		Swap: SwapStats{TotalBytes: 2 << 30, UsedBytes: 1 << 20, UsedPercent: 0.5},
		Filesystems: []FilesystemStats{
			{MountedOn: "/var", Filesystem: "/dev/sda2", TotalBytes: 100 << 30, UsedBytes: 50 << 30, FreeBytes: 50 << 30, UsedPercent: 50},
			{MountedOn: "/", Filesystem: "/dev/sda1", TotalBytes: 200 << 30, UsedBytes: 100 << 30, FreeBytes: 100 << 30, UsedPercent: 50},
		},
		Networks: []NetworkStats{
			{Name: "eth1", RxBytes: 2 << 20, TxBytes: 1 << 20},
			{Name: "eth0", RxBytes: 1 << 20, TxBytes: 1 << 20, RxDropped: 3, TxDropped: 1},
		},
		CoreUsage: map[string]CPUUsage{
			"cpu1": {Used: 20, User: 10, System: 10},
			"cpu0": {Used: 10, User: 5, System: 5},
		},
	}
}

func TestSnapshotFormat(t *testing.T) {
	t.Parallel()

	snapshot := buildSnapshot()
	output := snapshot.Format()

	for _, want := range []string{
		"Host:", "host-a",
		"Kernel:", "Linux 5.15.148-tegra aarch64",
		"Collected at:", "2026-11-27 10:00:00 (interval 1s)",
		"Uptime:", "25d4h38m",
		"Load average:", "0.50 0.40 0.30 (cores: 8, tasks: 2 running / 200 total)",
		"CPU:", "used 12.5%",
		"Memory:", "available(est) 4.00 GiB",
		"Swap:", "used 1.00 MiB / 2.00 GiB (0.5%)",
		"Filesystems (2):",
		"Networks (2):",
		"CPU per core (2):",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("Format() 缺少 %q：\n%s", want, output)
		}
	}

	// 磁盘按挂载点、网卡按名称排序，保证输出稳定
	if root, variable := strings.Index(output, "/dev/sda1"), strings.Index(output, "/dev/sda2"); root < 0 || variable < 0 || root > variable {
		t.Errorf("文件系统未按挂载点排序：root=%d var=%d\n%s", root, variable, output)
	}
	if eth0, eth1 := strings.Index(output, "eth0"), strings.Index(output, "eth1"); eth0 < 0 || eth1 < 0 || eth0 > eth1 {
		t.Errorf("网卡未按名称排序：eth0=%d eth1=%d\n%s", eth0, eth1, output)
	}
	if cpu0, cpu1 := strings.Index(output, "cpu0"), strings.Index(output, "cpu1"); cpu0 < 0 || cpu1 < 0 || cpu0 > cpu1 {
		t.Errorf("核心未按编号排序：cpu0=%d cpu1=%d\n%s", cpu0, cpu1, output)
	}
}

func TestSnapshotFormatCoreOverview(t *testing.T) {
	t.Parallel()

	cores := make(map[string]CPUUsage, 20)
	for idx := 0; idx < 20; idx++ {
		cores["cpu"+strconv.Itoa(idx)] = CPUUsage{Used: float64(idx + 1)}
	}

	snapshot := &Snapshot{CoreUsage: cores}
	output := snapshot.Format()

	// 超过 16 核时只输出概览：min 1.0 / avg 10.5 / max 20.0
	want := "CPU per core: 20 cores, used min 1.0% / avg 10.5% / max 20.0%"
	if !strings.Contains(output, want) {
		t.Errorf("Format() 概览 = %q, want 包含 %q", output, want)
	}
	if strings.Contains(output, "CORE") {
		t.Errorf("超过 16 核时不应输出逐核明细：\n%s", output)
	}
}

func TestSnapshotWriteTo(t *testing.T) {
	t.Parallel()

	var nilSnapshot *Snapshot
	if output := nilSnapshot.Format(); output != "" {
		t.Errorf("nil Snapshot.Format() = %q, want 空", output)
	}
	if written, err := nilSnapshot.WriteTo(&bytes.Buffer{}); written != 0 || err != nil {
		t.Errorf("nil Snapshot.WriteTo() = %d, %v", written, err)
	}

	snapshot := buildSnapshot()
	expected := snapshot.Format()

	var buffer bytes.Buffer
	written, err := snapshot.WriteTo(&buffer)
	if err != nil {
		t.Fatalf("WriteTo() 错误 = %v", err)
	}
	if written != int64(len(expected)) || buffer.String() != expected {
		t.Errorf("WriteTo() = %d 字节，与 Format() 的 %d 字节不一致", written, len(expected))
	}
}
