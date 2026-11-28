package command

import (
	"errors"
	"testing"
	"time"
)

// 以下样本均来自真实设备：
//   - 192.0.2.2：Ubuntu 22.04 / 内核 6.8 / 96 核
//   - 198.51.100.31：嵌入式 Linux 3.10（BusyBox 1.24）/ 2 核
//   - 203.0.113.33：Jetson Linux 5.15（Buildroot）/ 8 核

const (
	_PROC_UPTIME_SAMPLE       = "2176689.00 3789261.97\n"
	_LOAD_AVG_PROC_SAMPLE     = "0.26 0.37 0.44 1/160 1941\n"
	_LOAD_AVG_UPTIME_SAMPLE   = " 23:58:04 up 333 days,  5:55,  2 users,  load average: 62.03, 62.12, 62.03\n"
	_BUSYBOX_TOP_CPU_HEADER   = "Mem: 109528K used, 197236K free, 3928K shrd, 8280K buff, 40148K cached\n"
	_MEM_INFO_WITH_AVAIL      = "MemTotal:       31434800 kB\nMemFree:         5536960 kB\nMemAvailable:   24702632 kB\nBuffers:         1492008 kB\nCached:         17690960 kB\nSReclaimable:     123456 kB\nSwapTotal:       15716348 kB\nSwapFree:        15714736 kB\n"
	_MEM_INFO_WITHOUT_AVAIL   = "MemTotal:         306764 kB\nMemFree:          197540 kB\nBuffers:            8280 kB\nCached:            40148 kB\nSwapTotal:             0 kB\nSwapFree:              0 kB\n"
	_MEM_INFO_WITHOUT_TOTAL   = "MemFree:          197540 kB\n"
	_PROC_STAT_OLD_SAMPLE     = "cpu  13374001 0 40031100 372455903 4381 8593 1128759 0 0 0\ncpu0 8087784 0 30990122 172144945 881 6118 797102 0 0 0\ncpu3 5286056 0 9040327 200308792 3389 2475 331657 0 0 0\nintr 7254734992 0 0\nctxt 28469608749\nbtime 1793618399\n"
	_PROC_STAT_OLD_SAMPLE_TWO = "cpu  13374011 0 40031110 372455923 4381 8593 1128759 0 0 0\ncpu0 8087794 0 30990132 172144965 881 6118 797102 0 0 0\ncpu3 5286056 0 9040327 200308792 3389 2475 331657 0 0 0\nintr 7254734992 0 0\nctxt 28469608749\nbtime 1793618399\n"
	_PROC_STAT_EMPTY_SAMPLE   = "intr 7254734992\nctxt 28469608749\n"
	_CPU_INFO_X86_SAMPLE      = "processor\t: 0\nmodel name\t: Intel(R) Xeon(R) Gold 6338 CPU @ 2.00GHz\n\nprocessor\t: 1\nmodel name\t: Intel(R) Xeon(R) Gold 6338 CPU @ 2.00GHz\n"
	_CPU_INFO_ARM_SAMPLE      = "processor\t: 0\nmodel name\t: ARMv8 Processor rev 1 (v8l)\n\nprocessor\t: 1\nmodel name\t: ARMv8 Processor rev 1 (v8l)\n"
	_NET_DEV_SAMPLE           = "Inter-|   Receive                                                |  Transmit\n face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed\n    lo: 82281849290 961041878    0    0    0     0          0         0 82281849290 961041878    0    0    0     0       0          0\n  eth0: 1269313076 25768682 5147 2593    0  5147          0         0 422742334 2184225    0    0    0     0       0          0\n"
)

func TestParseUptime(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		output string
		want   time.Duration
	}{
		{"proc 伪文件", _PROC_UPTIME_SAMPLE, 2176689 * time.Second},
		{"uptime 命令（天）", " 23:58:04 up 333 days,  5:55,  2 users,  load average: 62.03, 62.12, 62.03", 333*24*time.Hour + 5*time.Hour + 55*time.Minute},
		{"uptime 命令（小时）", " 09:01:50 up  4:38,  1 user,  load average: 0.1, 0.2, 0.3", 4*time.Hour + 38*time.Minute},
		{"uptime 命令（分钟）", " 09:01:50 up 5 min,  1 user,  load average: 0.1, 0.2, 0.3", 5 * time.Minute},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseUptime(tc.output)
			if err != nil {
				t.Fatalf("ParseUptime() 错误 = %v", err)
			}
			if got != tc.want {
				t.Errorf("ParseUptime() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseUptimeInvalid(t *testing.T) {
	t.Parallel()

	for _, output := range []string{"", "up", "not-an-uptime"} {
		if _, err := ParseUptime(output); !errors.Is(err, ErrParse) {
			t.Errorf("ParseUptime(%q) 错误 = %v, want ErrParse", output, err)
		}
	}
}

func TestParseLoadAverage(t *testing.T) {
	t.Parallel()

	// /proc/loadavg：额外解析可运行任务数与任务总数
	procLoad, err := ParseLoadAverage(_LOAD_AVG_PROC_SAMPLE)
	if err != nil {
		t.Fatalf("ParseLoadAverage(/proc/loadavg) 错误 = %v", err)
	}
	if procLoad.One != 0.26 || procLoad.Five != 0.37 || procLoad.Fifteen != 0.44 {
		t.Errorf("ParseLoadAverage() = %v, want 0.26/0.37/0.44", procLoad)
	}
	if procLoad.Running != 1 || procLoad.Total != 160 {
		t.Errorf("ParseLoadAverage() 任务数 = %d/%d, want 1/160", procLoad.Running, procLoad.Total)
	}

	// uptime / top 头部：逗号分隔，不含任务数
	cmdLoad, err := ParseLoadAverage(_LOAD_AVG_UPTIME_SAMPLE)
	if err != nil {
		t.Fatalf("ParseLoadAverage(uptime) 错误 = %v", err)
	}
	if cmdLoad.One != 62.03 || cmdLoad.Five != 62.12 || cmdLoad.Fifteen != 62.03 {
		t.Errorf("ParseLoadAverage(uptime) = %v", cmdLoad)
	}
	if cmdLoad.Running != 0 || cmdLoad.Total != 0 {
		t.Errorf("ParseLoadAverage(uptime) 任务数 = %d/%d, want 0/0", cmdLoad.Running, cmdLoad.Total)
	}

	if _, err := ParseLoadAverage("nothing here"); !errors.Is(err, ErrParse) {
		t.Errorf("ParseLoadAverage(非法输入) 错误 = %v, want ErrParse", err)
	}
}

func TestParseProcStatAndUsage(t *testing.T) {
	t.Parallel()

	// 嵌入式设备的 CPU 编号可能不连续（本机只有 cpu0 与 cpu3）
	prev, err := ParseProcStat(_PROC_STAT_OLD_SAMPLE)
	if err != nil {
		t.Fatalf("ParseProcStat(prev) 错误 = %v", err)
	}
	if len(prev.Cores) != 2 {
		t.Fatalf("ParseProcStat(prev) 核心数 = %d, want 2", len(prev.Cores))
	}
	if _, ok := prev.Cores["cpu3"]; !ok {
		t.Errorf("ParseProcStat(prev) 缺少 cpu3 核心：%v", prev.Cores)
	}

	cur, err := ParseProcStat(_PROC_STAT_OLD_SAMPLE_TWO)
	if err != nil {
		t.Fatalf("ParseProcStat(cur) 错误 = %v", err)
	}

	set, err := CalculateCPUUsageSet(prev, cur)
	if err != nil {
		t.Fatalf("CalculateCPUUsageSet() 错误 = %v", err)
	}

	// 总时间片增量为 40：user +10、system +10、idle +20
	if set.Total.User != 25 || set.Total.System != 25 || set.Total.Idle != 50 || set.Total.Used != 50 {
		t.Errorf("总体使用率 = %+v, want user 25 / system 25 / idle 50 / used 50", set.Total)
	}

	// cpu0 有增量，cpu3 在采样间隔内无变化因此被跳过
	if len(set.Cores) != 1 {
		t.Fatalf("逐核使用率数量 = %d, want 1（cpu3 无变化应被跳过）", len(set.Cores))
	}
	if usage := set.Cores["cpu0"]; usage.Used != 50 {
		t.Errorf("cpu0 使用率 = %+v, want used 50", usage)
	}

	if _, err := ParseProcStat(_PROC_STAT_EMPTY_SAMPLE); !errors.Is(err, ErrParse) {
		t.Errorf("ParseProcStat(无 cpu 行) 错误 = %v, want ErrParse", err)
	}

	if _, err := CalculateCPUUsageSet(nil, nil); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("CalculateCPUUsageSet(nil) 错误 = %v, want ErrInvalidArgument", err)
	}

	if _, err := CalculateCPUUsage(prev.Total, prev.Total); !errors.Is(err, ErrParse) {
		t.Errorf("CalculateCPUUsage(无增量) 错误 = %v, want ErrParse", err)
	}
}

func TestParseMemInfo(t *testing.T) {
	t.Parallel()

	// 含 MemAvailable：直接使用内核给出的可用内存
	withAvail, err := ParseMemInfo(_MEM_INFO_WITH_AVAIL)
	if err != nil {
		t.Fatalf("ParseMemInfo(含 MemAvailable) 错误 = %v", err)
	}
	if withAvail.Memory.TotalBytes != 31434800<<10 {
		t.Errorf("MemTotal = %d, want %d", withAvail.Memory.TotalBytes, 31434800<<10)
	}
	if withAvail.Memory.AvailableBytes != 24702632<<10 {
		t.Errorf("MemAvailable = %d, want %d", withAvail.Memory.AvailableBytes, 24702632<<10)
	}
	if withAvail.Memory.AvailableEstimated {
		t.Error("含 MemAvailable 时不应标记为推算值")
	}
	if withAvail.Memory.CachedBytes != (17690960+123456)<<10 {
		t.Errorf("CachedBytes = %d, want Cached + SReclaimable", withAvail.Memory.CachedBytes)
	}
	if withAvail.Swap.TotalBytes != 15716348<<10 || withAvail.Swap.FreeBytes != 15714736<<10 {
		t.Errorf("Swap = %+v, want total %d free %d", withAvail.Swap, 15716348<<10, 15714736<<10)
	}
	if withAvail.Swap.UsedBytes != (15716348-15714736)<<10 {
		t.Errorf("Swap.UsedBytes = %d, want %d", withAvail.Swap.UsedBytes, (15716348-15714736)<<10)
	}

	// 内核 3.10 没有 MemAvailable：按 Free + Buffers + Cached 推算
	withoutAvail, err := ParseMemInfo(_MEM_INFO_WITHOUT_AVAIL)
	if err != nil {
		t.Fatalf("ParseMemInfo(无 MemAvailable) 错误 = %v", err)
	}
	wantAvail := uint64(197540+8280+40148) << 10
	if withoutAvail.Memory.AvailableBytes != wantAvail {
		t.Errorf("推算 AvailableBytes = %d, want %d", withoutAvail.Memory.AvailableBytes, wantAvail)
	}
	if !withoutAvail.Memory.AvailableEstimated {
		t.Error("无 MemAvailable 时应标记为推算值")
	}
	if withoutAvail.Memory.UsedBytes != (306764-245968)<<10 {
		t.Errorf("UsedBytes = %d, want %d", withoutAvail.Memory.UsedBytes, (306764-245968)<<10)
	}
	if withoutAvail.Memory.UsedPercent < 19.81 || withoutAvail.Memory.UsedPercent > 19.83 {
		t.Errorf("UsedPercent = %.2f, want ≈19.82", withoutAvail.Memory.UsedPercent)
	}
	if withoutAvail.Swap.TotalBytes != 0 || withoutAvail.Swap.UsedPercent != 0 {
		t.Errorf("无交换分区时 Swap = %+v, want 零值", withoutAvail.Swap)
	}

	if _, err := ParseMemInfo(_MEM_INFO_WITHOUT_TOTAL); !errors.Is(err, ErrParse) {
		t.Errorf("ParseMemInfo(缺 MemTotal) 错误 = %v, want ErrParse", err)
	}
}

func TestParseCPUInfo(t *testing.T) {
	t.Parallel()

	x86, err := ParseCPUInfo(_CPU_INFO_X86_SAMPLE)
	if err != nil {
		t.Fatalf("ParseCPUInfo(x86) 错误 = %v", err)
	}
	if x86.Processors != 2 {
		t.Errorf("x86 核心数 = %d, want 2", x86.Processors)
	}
	if x86.Model != "Intel(R) Xeon(R) Gold 6338 CPU @ 2.00GHz" {
		t.Errorf("x86 型号 = %q", x86.Model)
	}

	arm, err := ParseCPUInfo(_CPU_INFO_ARM_SAMPLE)
	if err != nil {
		t.Fatalf("ParseCPUInfo(ARM) 错误 = %v", err)
	}
	if arm.Processors != 2 || arm.Model != "ARMv8 Processor rev 1 (v8l)" {
		t.Errorf("ARM 信息 = %+v", arm)
	}

	if _, err := ParseCPUInfo(""); !errors.Is(err, ErrParse) {
		t.Errorf("ParseCPUInfo(空输入) 错误 = %v, want ErrParse", err)
	}
}

func TestParseNetDev(t *testing.T) {
	t.Parallel()

	entries, err := ParseNetDev(_NET_DEV_SAMPLE)
	if err != nil {
		t.Fatalf("ParseNetDev() 错误 = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("ParseNetDev() 条目数 = %d, want 2", len(entries))
	}

	eth0 := entries[1]
	if eth0.Name != "eth0" {
		t.Errorf("网卡名 = %q, want eth0", eth0.Name)
	}
	if eth0.RxBytes != 1269313076 || eth0.RxPackets != 25768682 {
		t.Errorf("eth0 接收 = %d 字节/%d 包", eth0.RxBytes, eth0.RxPackets)
	}
	if eth0.RxErrors != 5147 || eth0.RxDropped != 2593 {
		t.Errorf("eth0 接收错误 = %d/%d, want 5147/2593", eth0.RxErrors, eth0.RxDropped)
	}
	if eth0.TxBytes != 422742334 || eth0.TxPackets != 2184225 {
		t.Errorf("eth0 发送 = %d 字节/%d 包", eth0.TxBytes, eth0.TxPackets)
	}

	if _, err := ParseNetDev("Inter-| Receive\n face |bytes\n"); !errors.Is(err, ErrParse) {
		t.Errorf("ParseNetDev(仅表头) 错误 = %v, want ErrParse", err)
	}
}
