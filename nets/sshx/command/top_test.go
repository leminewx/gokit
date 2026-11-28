package command

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// 真实设备上的 top -b -n 1 输出（终端宽度 512）
const (
	// GNU procps-ng（192.0.2.2，Ubuntu 22.04）
	_TOP_GNU_SAMPLE = `top - 23:58:05 up 333 days,  5:55,  2 users,  load average: 62.03, 62.12, 62.03
Tasks: 1478 total,   1 running, 1458 sleeping,   0 stopped,  19 zombie
%Cpu(s):  0.3 us,  0.5 sy,  0.0 ni, 93.3 id,  5.8 wa,  0.0 hi,  0.1 si,  0.0 st
MiB Mem : 257357.2 total,   2450.7 free,  19852.6 used, 235053.9 buff/cache
MiB Swap:   2048.0 total,      0.3 free,   2047.7 used. 235941.3 avail Mem 

    PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
 687546 user01    20   0   58080  44196   2304 D  26.3   0.0  81:26.63 grep
1964008 user01    20   0   17648   4608   3072 R  15.8   0.0   0:00.06 top
   3696 root      20   0       0      0      0 S  10.5   0.0 359:47.59 cifsd
 653181 user01    20   0 8052700   2.1g  26272 S   5.3   0.8   3607:47 remote-dev-serv
2689220 root      20   0   23.5g 894452  94244 S   5.3   0.3   5341:33 Model: bce-embe
`

	// BusyBox 1.24（198.51.100.31，内核 3.10）：首行是 Mem:，没有 `top -` 头部
	_TOP_BUSYBOX_SAMPLE = `Mem: 109528K used, 197236K free, 3928K shrd, 8280K buff, 40148K cached
CPU:  9.0% usr 13.6% sys  0.0% nic 77.2% idle  0.0% io  0.0% irq  0.0% sirq
Load average: 0.22 0.36 0.43 2/165 1971
  PID  PPID USER     STAT   VSZ %VSZ CPU %CPU COMMAND
 4891     1 root     S    43388 14.1   0  4.5 /system/bin/qxwz
  546     1 root     S    36700 11.9   0  4.5 omCore
 1966  1965 root     R     1628  0.5   3  4.5 top -b -n 1
`

	// 无 PPID 的嵌入式 top（部分 ToyBox / 定制 BusyBox 版本）
	_TOP_EMBEDDED_SAMPLE = `Mem: 512M used, 128M free, 0K shrd, 32M buff, 64M cached
CPU:  1.0% usr  2.0% sys  0.0% nic 97.0% idle  0.0% io  0.0% irq  0.0% sirq
Load average: 1.00 0.50 0.25 1/50 999
  PID USER     STAT   VSZ %VSZ CPU %CPU COMMAND
  100 root     S     1000  1.0   0  2.0 app
`
)

func TestNewTopParser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		output string
		want   TopParser
	}{
		{"GNU procps", _TOP_GNU_SAMPLE, &standardTopParser{}},
		{"BusyBox（带 PPID）", _TOP_BUSYBOX_SAMPLE, &oldTopParser{}},
		{"BusyBox（无 PPID）", _TOP_EMBEDDED_SAMPLE, &embeddedTopParser{}},
		{"无法识别时兜底旧式", "nothing useful here\n", &oldTopParser{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotType := fmt.Sprintf("%T", NewTopParser([]byte(tc.output)))
			wantType := fmt.Sprintf("%T", tc.want)
			if gotType != wantType {
				t.Errorf("NewTopParser() = %s, want %s", gotType, wantType)
			}
		})
	}
}

func TestParseTopGNU(t *testing.T) {
	t.Parallel()

	stats, err := ParseTop([]byte(_TOP_GNU_SAMPLE))
	if err != nil {
		t.Fatalf("ParseTop(GNU) 错误 = %v", err)
	}

	if stats.LoadAverage != [3]float64{62.03, 62.12, 62.03} {
		t.Errorf("负载 = %v", stats.LoadAverage)
	}

	cpu := stats.CPU
	if cpu.User != 0.3 || cpu.System != 0.5 || cpu.Idle != 93.3 || cpu.IOWait != 5.8 || cpu.SoftIRQ != 0.1 {
		t.Errorf("CPU = %+v", cpu)
	}

	mem := stats.Memory
	if mem.TotalMiB != 257357.2 || mem.UsedMiB != 19852.6 || mem.BuffCacheMiB != 235053.9 {
		t.Errorf("内存 = %+v", mem)
	}
	// avail Mem 行没有显式单位，需沿用在内存总览行上取到的 MiB
	if mem.AvailMiB != 235941.3 {
		t.Errorf("可用内存 = %.1f MiB, want 235941.3", mem.AvailMiB)
	}
	if mem.UsedPercent < 7.71 || mem.UsedPercent > 7.72 {
		t.Errorf("内存使用率 = %.2f%%, want ≈7.71%%", mem.UsedPercent)
	}

	if len(stats.Processes) != 5 {
		t.Fatalf("进程数 = %d, want 5", len(stats.Processes))
	}

	first := stats.Processes[0]
	if first.PID != 687546 || first.User != "user01" || first.State != "D" {
		t.Errorf("首个进程 = %+v", first)
	}
	if first.CPUPercent != 26.3 || first.MEMPercent != 0 {
		t.Errorf("首个进程占用 = %.1f/%.1f", first.CPUPercent, first.MEMPercent)
	}
	if first.VSZBytes != 58080<<10 || first.RSSBytes != 44196<<10 {
		t.Errorf("首个进程内存 = %d/%d", first.VSZBytes, first.RSSBytes)
	}
	if first.Command != "grep" {
		t.Errorf("首个进程命令行 = %q", first.Command)
	}
}

func TestParseTopBusybox(t *testing.T) {
	t.Parallel()

	parser := NewTopParser([]byte(_TOP_BUSYBOX_SAMPLE))
	stats, err := parser.Parse(_TOP_BUSYBOX_SAMPLE)
	if err != nil {
		t.Fatalf("Parse() 错误 = %v", err)
	}

	if stats.LoadAverage != [3]float64{0.22, 0.36, 0.43} {
		t.Errorf("负载 = %v", stats.LoadAverage)
	}

	cpu := stats.CPU
	if cpu.User != 9.0 || cpu.System != 13.6 || cpu.Nice != 0 || cpu.Idle != 77.2 || cpu.IOWait != 0 {
		t.Errorf("CPU = %+v", cpu)
	}

	// 内存值单位是 K，需要换算为 MiB；total = used + free + buff + cached
	mem := stats.Memory
	wantTotal := float64(109528+197236+8280+40148) / 1024
	if diff := mem.TotalMiB - wantTotal; diff > 0.001 || diff < -0.001 {
		t.Errorf("内存总量 = %.3f MiB, want %.3f", mem.TotalMiB, wantTotal)
	}
	if diff := mem.BuffCacheMiB - float64(8280+40148)/1024; diff > 0.001 || diff < -0.001 {
		t.Errorf("buff/cache = %.3f MiB", mem.BuffCacheMiB)
	}
	if mem.UsedPercent < 30.83 || mem.UsedPercent > 30.85 {
		t.Errorf("内存使用率 = %.2f%%, want ≈30.84%%", mem.UsedPercent)
	}

	if len(stats.Processes) != 3 {
		t.Fatalf("进程数 = %d, want 3", len(stats.Processes))
	}

	// %VSZ 是百分比列，不能被当作容量；CPU 是核心编号，不能被当作使用率
	qxwz := stats.Processes[0]
	if qxwz.PID != 4891 || qxwz.PPID != 1 || qxwz.State != "S" {
		t.Errorf("qxwz = %+v", qxwz)
	}
	if qxwz.VSZBytes != 43388<<10 || qxwz.RSSBytes != 0 {
		t.Errorf("qxwz 内存 = %d/%d, want VSZ 43388KiB / RSS 0", qxwz.VSZBytes, qxwz.RSSBytes)
	}
	if qxwz.CPUPercent != 4.5 || qxwz.MEMPercent != 0 {
		t.Errorf("qxwz 占用 = %.1f/%.1f", qxwz.CPUPercent, qxwz.MEMPercent)
	}
	if qxwz.Command != "/system/bin/qxwz" {
		t.Errorf("qxwz 命令行 = %q", qxwz.Command)
	}
}

func TestParseTopEmbedded(t *testing.T) {
	t.Parallel()

	stats, err := ParseTop([]byte(_TOP_EMBEDDED_SAMPLE))
	if err != nil {
		t.Fatalf("ParseTop(嵌入式) 错误 = %v", err)
	}

	// 内存值单位是 M，需要换算为 MiB
	mem := stats.Memory
	if mem.TotalMiB != 736 || mem.UsedMiB != 512 || mem.FreeMiB != 128 {
		t.Errorf("内存 = %+v, want total 736 / used 512 / free 128 MiB", mem)
	}
	if diff := mem.BuffCacheMiB - 96; diff > 0.001 || diff < -0.001 {
		t.Errorf("buff/cache = %.3f MiB, want 96", mem.BuffCacheMiB)
	}

	if len(stats.Processes) != 1 {
		t.Fatalf("进程数 = %d, want 1", len(stats.Processes))
	}
	if process := stats.Processes[0]; process.PID != 100 || process.VSZBytes != 1000<<10 || process.Command != "app" {
		t.Errorf("进程 = %+v", process)
	}
}

func TestParseTopUnknown(t *testing.T) {
	t.Parallel()

	// 完全无法识别的输出应返回 ErrParse，而不是全零快照
	if _, err := ParseTop([]byte("nothing useful here\n")); !errors.Is(err, ErrParse) {
		t.Errorf("ParseTop(无法识别) 错误 = %v, want ErrParse", err)
	}
}

func TestParseTopNarrowTerminal(t *testing.T) {
	t.Parallel()

	// 终端过窄时 GNU top 会裁掉 %CPU/%MEM 甚至 COMMAND 列，此时不应产生错误的进程记录
	narrow := `top - 23:58:04 up 333 days,  5:55,  2 users,  load average: 62.03, 62.12, 62.03
%Cpu(s):  0.1 us,  0.4 sy,  0.0 ni, 93.3 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st
MiB Mem : 257357.2 total,   2506.9 free,  19852.6 used, 235053.9 buff/cache
    PID USER      PR  NI    VIRT    RES
 687546 user01    20   0   58080  44196
`
	stats, err := ParseTop([]byte(narrow))
	if err != nil {
		t.Fatalf("ParseTop(窄终端) 错误 = %v", err)
	}
	if len(stats.Processes) != 0 {
		t.Errorf("窄终端进程数 = %d, want 0（缺少 COMMAND 列）", len(stats.Processes))
	}
	if stats.CPU.Idle != 93.3 || stats.Memory.TotalMiB != 257357.2 {
		t.Errorf("窄终端仍应解析出 CPU 与内存：%+v / %+v", stats.CPU, stats.Memory)
	}
}

func TestParseTopIdleCPU(t *testing.T) {
	t.Parallel()

	// 整机完全空闲时 idle 为 100.0，与前面逗号之间没有空格，仍应解析成功
	sample := strings.Replace(_TOP_GNU_SAMPLE, " 93.3 id", "100.0 id", 1)
	stats, err := ParseTop([]byte(sample))
	if err != nil {
		t.Fatalf("ParseTop(全空闲) 错误 = %v", err)
	}
	if stats.CPU.Idle != 100 || stats.CPU.User != 0.3 {
		t.Errorf("CPU = %+v, want idle 100 / user 0.3", stats.CPU)
	}
}

func TestParseTopPerCore(t *testing.T) {
	t.Parallel()

	// `top -1` 只输出逐核行，此时按各核平均值近似整机
	perCore := `top - 10:00:00 up 1 day,  0:00,  1 user,  load average: 0.10, 0.20, 0.30
%Cpu0  :  0.0 us,  0.0 sy,  0.0 ni,100.0 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st
%Cpu1  : 20.0 us, 10.0 sy,  0.0 ni, 70.0 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st
MiB Mem :   1000.0 total,    500.0 free,    400.0 used,    100.0 buff/cache
MiB Swap:      0.0 total,      0.0 free,      0.0 used.    600.0 avail Mem
`
	stats, err := ParseTop([]byte(perCore))
	if err != nil {
		t.Fatalf("ParseTop(逐核) 错误 = %v", err)
	}

	if stats.CPU.User != 10 || stats.CPU.System != 5 || stats.CPU.Idle != 85 {
		t.Errorf("逐核平均 CPU = %+v, want user 10 / system 5 / idle 85", stats.CPU)
	}
	if stats.Memory.TotalMiB != 1000 {
		t.Errorf("内存 = %+v", stats.Memory)
	}
}

func TestParseTopLoadAverageCase(t *testing.T) {
	t.Parallel()

	// 部分 BusyBox / 定制 busybox 输出小写 load average，解析应忽略大小写
	sample := strings.Replace(_TOP_EMBEDDED_SAMPLE, "Load average:", "load average:", 1)
	stats, err := ParseTop([]byte(sample))
	if err != nil {
		t.Fatalf("ParseTop(小写负载) 错误 = %v", err)
	}
	if stats.LoadAverage != [3]float64{1.00, 0.50, 0.25} {
		t.Errorf("负载 = %v", stats.LoadAverage)
	}
}
