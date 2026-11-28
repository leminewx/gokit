package command

import (
	"errors"
	"testing"
)

// 真实设备样本
const (
	// procps-ng（192.0.2.2，ps aux）
	_PS_AUX_SAMPLE = `USER         PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
root           1  1.0  0.0 168896 21504 ?        Ss    2025 4882:44 /lib/systemd/systemd --system --deserialize 64 splash
root           2  0.0  0.0      0     0 ?        S     2025  10:23 [kthreadd]
`

	// procps-ng（203.0.113.33，ps -eo pid,ppid,user,pcpu,pmem,args）
	_PS_EO_FORMAT_SAMPLE = `    PID    PPID USER     %CPU %MEM COMMAND
      1       0 root      0.0  0.0 /sbin/init 2
      2       0 root      0.0  0.0 [kthreadd]
`

	// BusyBox 1.24（198.51.100.31，ps / ps aux 均为该格式）
	_PS_BUSYBOX_SAMPLE = `PID   USER     TIME   COMMAND
    1 root       0:00 init
    2 root       0:00 [kthreadd]
`

	// BusyBox 1.24（198.51.100.31，ps -o pid,ppid,user,stat,vsz,rss,args）
	_PS_BUSYBOX_O_SAMPLE = `  PID  PPID USER     STAT   VSZ   RSS COMMAND
    1     0 root     S     1628   600 init
 4891     1 root     S    43388 11892 /system/bin/qxwz
`
)

func TestParsePS(t *testing.T) {
	t.Parallel()

	t.Run("ps aux", func(t *testing.T) {
		t.Parallel()

		processes, err := ParsePS(_PS_AUX_SAMPLE)
		if err != nil {
			t.Fatalf("ParsePS(aux) 错误 = %v", err)
		}
		if len(processes) != 2 {
			t.Fatalf("ParsePS(aux) 条目数 = %d, want 2", len(processes))
		}

		init := processes[0]
		if init.PID != 1 || init.User != "root" || init.State != "Ss" {
			t.Errorf("init 进程 = %+v", init)
		}
		if init.CPUPercent != 1 || init.MEMPercent != 0 {
			t.Errorf("init 占用 = %.1f/%.1f, want 1.0/0.0", init.CPUPercent, init.MEMPercent)
		}
		if init.VSZBytes != 168896<<10 || init.RSSBytes != 21504<<10 {
			t.Errorf("init 内存 = %d/%d", init.VSZBytes, init.RSSBytes)
		}
		if init.Command != "/lib/systemd/systemd --system --deserialize 64 splash" {
			t.Errorf("init 命令行 = %q", init.Command)
		}

		// 内核线程的 CPU/内存列为 0
		if processes[1].Command != "[kthreadd]" || processes[1].VSZBytes != 0 {
			t.Errorf("kthreadd = %+v", processes[1])
		}
	})

	t.Run("ps -eo pid,ppid,user,pcpu,pmem,args", func(t *testing.T) {
		t.Parallel()

		processes, err := ParsePS(_PS_EO_FORMAT_SAMPLE)
		if err != nil {
			t.Fatalf("ParsePS(-eo) 错误 = %v", err)
		}
		if len(processes) != 2 {
			t.Fatalf("ParsePS(-eo) 条目数 = %d, want 2", len(processes))
		}
		if processes[0].PPID != 0 || processes[0].Command != "/sbin/init 2" {
			t.Errorf("init 进程 = %+v", processes[0])
		}
	})

	t.Run("BusyBox ps", func(t *testing.T) {
		t.Parallel()

		processes, err := ParsePS(_PS_BUSYBOX_SAMPLE)
		if err != nil {
			t.Fatalf("ParsePS(BusyBox) 错误 = %v", err)
		}
		if len(processes) != 2 {
			t.Fatalf("ParsePS(BusyBox) 条目数 = %d, want 2", len(processes))
		}

		init := processes[0]
		if init.PID != 1 || init.User != "root" || init.Command != "init" {
			t.Errorf("BusyBox init = %+v", init)
		}
		// 缺少 %CPU/%MEM 列时保持 0，而不是解析到 TIME 列
		if init.CPUPercent != 0 || init.MEMPercent != 0 {
			t.Errorf("BusyBox 占用 = %.1f/%.1f, want 0/0", init.CPUPercent, init.MEMPercent)
		}
	})

	t.Run("BusyBox ps -o", func(t *testing.T) {
		t.Parallel()

		processes, err := ParsePS(_PS_BUSYBOX_O_SAMPLE)
		if err != nil {
			t.Fatalf("ParsePS(BusyBox -o) 错误 = %v", err)
		}
		if len(processes) != 2 {
			t.Fatalf("ParsePS(BusyBox -o) 条目数 = %d, want 2", len(processes))
		}

		qxwz := processes[1]
		if qxwz.PID != 4891 || qxwz.PPID != 1 || qxwz.State != "S" {
			t.Errorf("qxwz 进程 = %+v", qxwz)
		}
		if qxwz.VSZBytes != 43388<<10 || qxwz.RSSBytes != 11892<<10 {
			t.Errorf("qxwz 内存 = %d/%d", qxwz.VSZBytes, qxwz.RSSBytes)
		}
		if qxwz.Command != "/system/bin/qxwz" {
			t.Errorf("qxwz 命令行 = %q", qxwz.Command)
		}
	})

	t.Run("表头缺失", func(t *testing.T) {
		t.Parallel()

		if _, err := ParsePS("some random text\n"); !errors.Is(err, ErrParse) {
			t.Errorf("ParsePS(无表头) 错误 = %v, want ErrParse", err)
		}
	})
}
