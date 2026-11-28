package command

import "testing"

func TestLoadAverageString(t *testing.T) {
	tests := []struct {
		name string
		in   LoadAverage
		want string
	}{
		{"零值", LoadAverage{}, "0.00 0.00 0.00"},
		{"常规", LoadAverage{One: 0.26, Five: 0.37, Fifteen: 0.44}, "0.26 0.37 0.44"},
		{"四舍五入", LoadAverage{One: 1.005, Five: 2.345, Fifteen: 3.999}, "1.00 2.35 4.00"},
		{"忽略任务数", LoadAverage{One: 1, Five: 2, Fifteen: 3, Running: 5, Total: 100}, "1.00 2.00 3.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCPUTimesGetTotal(t *testing.T) {
	tests := []struct {
		name string
		in   CPUTimes
		want uint64
	}{
		{"零值", CPUTimes{}, 0},
		{"单字段", CPUTimes{Idle: 10}, 10},
		{
			"全部字段",
			CPUTimes{User: 1, Nice: 2, System: 3, Idle: 4, IOWait: 5, IRQ: 6, SoftIRQ: 7, Steal: 8},
			36,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.GetTotal(); got != tt.want {
				t.Fatalf("GetTotal() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCPUUsageString(t *testing.T) {
	usage := CPUUsage{
		User:    10.1,
		System:  20.2,
		Nice:    0.3,
		IOWait:  1.4,
		IRQ:     0.5,
		SoftIRQ: 0.6,
		Steal:   0.7,
		Idle:    66.2,
		Used:    33.8,
	}

	want := "used 33.8% | user 10.1% system 20.2% nice 0.3% iowait 1.4% irq 0.5% sirq 0.6% steal 0.7% idle 66.2%"
	if got := usage.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestMemoryStatsString(t *testing.T) {
	tests := []struct {
		name string
		in   MemoryStats
		want string
	}{
		{"零值", MemoryStats{}, "used 0.00% | available 0 B / 0 B"},
		{
			"常规",
			MemoryStats{TotalBytes: 2 << 30, AvailableBytes: 512 << 20, UsedPercent: 75.5},
			"used 75.50% | available 512.00 MiB / 2.00 GiB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
