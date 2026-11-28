package command

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitee.com/leminewx/gokit/stringx"
)

// _MAX_CORE_DETAIL 为逐核明细的输出上限
//   - 超过该数量时（如 96 核服务器）只输出 min/avg/max 概览，避免刷屏
//   - 完整逐核数据仍在 Snapshot.CoreUsage 中
const _MAX_CORE_DETAIL = 16

// humanBytes 将字节数转换为人类可读格式（复用 stringx.HumanizeBytes）
func humanBytes(v uint64) string {
	if v > uint64(1<<62) { // 避免超出 int64 表示范围
		v = 1 << 62
	}

	return stringx.HumanizeBytes(int64(v))
}

// Format 返回人类可读的整机概览（逐行文本，便于直接日志输出）
//   - 各段按固定顺序输出：主机、内核、运行时长、负载、CPU、内存、交换区、磁盘、网卡
//   - 磁盘按挂载点、网卡按名称排序，保证同一快照输出的顺序稳定
func (own *Snapshot) Format() string {
	if own == nil {
		return ""
	}

	var builder strings.Builder
	write := func(label, format string, args ...any) {
		fmt.Fprintf(&builder, "%-14s", label)
		fmt.Fprintf(&builder, format, args...)
		builder.WriteByte('\n')
	}

	if own.Host != "" {
		write("Host:", "%s", own.Host)
	}
	if own.Kernel != "" {
		write("Kernel:", "%s", own.Kernel)
	}
	if !own.CollectedAt.IsZero() {
		write("Collected at:", "%s (interval %s)", own.CollectedAt.Format("2006-01-02 15:04:05"), own.SampleInterval)
	}
	if own.Uptime > 0 {
		write("Uptime:", "%s", formatDuration(own.Uptime))
	}

	if own.Load.One != 0 || own.Load.Five != 0 || own.Load.Fifteen != 0 {
		tasks := ""
		if own.Load.Total > 0 {
			tasks = fmt.Sprintf(", tasks: %d running / %d total", own.Load.Running, own.Load.Total)
		}
		write("Load average:", "%s (cores: %d%s)", own.Load, own.CPUCores, tasks)
	}

	if own.CPU.Used != 0 || own.CPU.Idle != 0 {
		write("CPU:", "%s", own.CPU)
	}

	if own.Memory.TotalBytes > 0 {
		available := ""
		if own.Memory.AvailableEstimated {
			available = fmt.Sprintf(" | available(est) %s", humanBytes(own.Memory.AvailableBytes))
		} else {
			available = fmt.Sprintf(" | available %s", humanBytes(own.Memory.AvailableBytes))
		}

		write("Memory:", "used %s / %s (%.1f%%)%s | buffers %s | cached %s",
			humanBytes(own.Memory.UsedBytes), humanBytes(own.Memory.TotalBytes),
			own.Memory.UsedPercent, available,
			humanBytes(own.Memory.BuffersBytes), humanBytes(own.Memory.CachedBytes))
	}

	write("Swap:", "used %s / %s (%.1f%%)",
		humanBytes(own.Swap.UsedBytes), humanBytes(own.Swap.TotalBytes), own.Swap.UsedPercent)

	if len(own.Filesystems) > 0 {
		own.writeFilesystems(&builder)
	}

	if len(own.Networks) > 0 {
		own.writeNetworks(&builder)
	}

	if len(own.CoreUsage) > 0 {
		own.writeCoreUsage(&builder)
	}

	return builder.String()
}

// WriteTo 将概览写入 writer，便于直接输出到日志或 HTTP 响应
func (own *Snapshot) WriteTo(w io.Writer) (int64, error) {
	if own == nil {
		return 0, nil
	}

	written, err := io.WriteString(w, own.Format())
	return int64(written), err
}

// writeFilesystems 输出磁盘使用情况（按挂载点排序）
func (own *Snapshot) writeFilesystems(builder *strings.Builder) {
	filesystems := make([]FilesystemStats, len(own.Filesystems))
	copy(filesystems, own.Filesystems)
	sort.Slice(filesystems, func(i, j int) bool {
		return filesystems[i].MountedOn < filesystems[j].MountedOn
	})

	fmt.Fprintf(builder, "Filesystems (%d):\n", len(filesystems))
	fmt.Fprintf(builder, "  %-28s %7s %11s %11s %11s  %s\n", "MOUNT", "USAGE", "USED", "AVAIL", "TOTAL", "SOURCE")
	for _, fs := range filesystems {
		fmt.Fprintf(builder, "  %-28s %6.1f%% %11s %11s %11s  %s\n",
			fs.MountedOn, fs.UsedPercent,
			humanBytes(fs.UsedBytes), humanBytes(fs.FreeBytes), humanBytes(fs.TotalBytes),
			fs.Filesystem)
	}
}

// writeNetworks 输出网卡流量与丢包（按名称排序）
func (own *Snapshot) writeNetworks(builder *strings.Builder) {
	networks := make([]NetworkStats, len(own.Networks))
	copy(networks, own.Networks)
	sort.Slice(networks, func(i, j int) bool {
		return networks[i].Name < networks[j].Name
	})

	fmt.Fprintf(builder, "Networks (%d):\n", len(networks))
	fmt.Fprintf(builder, "  %-16s %11s %11s %9s %9s\n", "NAME", "RX", "TX", "RX-DROP", "TX-DROP")
	for _, nic := range networks {
		fmt.Fprintf(builder, "  %-16s %11s %11s %9d %9d\n",
			nic.Name, humanBytes(nic.RxBytes), humanBytes(nic.TxBytes), nic.RxDropped, nic.TxDropped)
	}
}

// writeCoreUsage 输出各核心使用率（按核心编号排序）
//   - 核心数超过 _MAX_CORE_DETAIL 时只输出概览
func (own *Snapshot) writeCoreUsage(builder *strings.Builder) {
	names := sortedCoreNames(own.CoreUsage)

	if len(names) > _MAX_CORE_DETAIL {
		var sum, minUsed, maxUsed float64
		for idx, name := range names {
			used := own.CoreUsage[name].Used
			if idx == 0 || used < minUsed {
				minUsed = used
			}
			if idx == 0 || used > maxUsed {
				maxUsed = used
			}
			sum += used
		}

		fmt.Fprintf(builder, "CPU per core: %d cores, used min %.1f%% / avg %.1f%% / max %.1f%%\n",
			len(names), minUsed, sum/float64(len(names)), maxUsed)
		return
	}

	fmt.Fprintf(builder, "CPU per core (%d):\n", len(names))
	fmt.Fprintf(builder, "  %-12s %8s %8s %8s %8s\n", "CORE", "USED", "USER", "SYSTEM", "IOWAIT")
	for _, name := range names {
		usage := own.CoreUsage[name]
		fmt.Fprintf(builder, "  %-12s %7.1f%% %7.1f%% %7.1f%% %7.1f%%\n",
			name, usage.Used, usage.User, usage.System, usage.IOWait)
	}
}

// sortedCoreNames 按核心编号排序，避免 cpu10 排在 cpu2 之前
func sortedCoreNames(cores map[string]CPUUsage) []string {
	names := make([]string, 0, len(cores))
	for name := range cores {
		names = append(names, name)
	}

	sort.Slice(names, func(i, j int) bool {
		left, leftOK := coreIndex(names[i])
		right, rightOK := coreIndex(names[j])

		switch {
		case leftOK && rightOK:
			return left < right
		case leftOK != rightOK:
			return leftOK
		default:
			return names[i] < names[j]
		}
	})

	return names
}

// coreIndex 提取 "cpu12" 中的编号
func coreIndex(name string) (int, bool) {
	idx, err := strconv.Atoi(strings.TrimPrefix(name, "cpu"))
	return idx, err == nil
}

// formatDuration 将时长格式化为 d/h/m/s 组合
//   - 超过 1 天：`25d4h38m`
//   - 超过 1 小时：`4h38m5s`
//   - 其余：`38m5s`
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}

	total := int64(d / time.Second)
	days := total / 86400
	hours := total % 86400 / 3600
	minutes := total % 3600 / 60
	seconds := total % 60

	var builder strings.Builder
	if days > 0 {
		fmt.Fprintf(&builder, "%dd%dh%dm", days, hours, minutes)
		return builder.String()
	}

	if hours > 0 {
		fmt.Fprintf(&builder, "%dh%dm%ds", hours, minutes, seconds)
		return builder.String()
	}

	if minutes > 0 {
		fmt.Fprintf(&builder, "%dm%ds", minutes, seconds)
		return builder.String()
	}

	fmt.Fprintf(&builder, "%ds", seconds)
	return builder.String()
}
