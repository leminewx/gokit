package command

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// uptime 命令：... up 25 days,  4:38,  load average: 0.22, 0.36, 0.43
	uptimeCommandRegex = regexp.MustCompile(`up\s+(?:(\d+)\s+days?,\s*)?(?:(\d+):(\d+)|(\d+)\s+min)`)
	// uptime/top 头部：load average: 0.22, 0.36, 0.43（分隔符可能是逗号或空格）
	loadAverageCommandRegex = regexp.MustCompile(`load averages?:\s+([\d.]+)[,\s]+([\d.]+)[,\s]+([\d.]+)`)
	// /proc/loadavg：0.26 0.37 0.44 1/160 1941
	loadAverageProcRegex = regexp.MustCompile(`^\s*([\d.]+)\s+([\d.]+)\s+([\d.]+)\s+(\d+)/(\d+)`)
	// /proc/loadavg 缺少运行任务段时的退化形式
	loadAveragePlainRegex = regexp.MustCompile(`^\s*([\d.]+)\s+([\d.]+)\s+([\d.]+)\s*$`)
)

// ParseUptime 解析系统运行时长，同时兼容两种输入：
//   - /proc/uptime：`2176689.00 3789261.97`（第一列为运行秒数）
//   - uptime 命令：` 23:58:04 up 333 days,  5:55,  2 users, ...`
func ParseUptime(output string) (time.Duration, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return 0, wrapError(ErrParse, "uptime: empty output")
	}

	// 1. /proc/uptime：以浮点秒数开头
	if fields := strings.Fields(output); len(fields) >= 1 {
		if seconds, ok := parseFloat(fields[0]); ok && !strings.Contains(fields[0], ":") {
			if len(fields) == 1 || isFloat(fields[1]) {
				return time.Duration(seconds * float64(time.Second)), nil
			}
		}
	}

	// 2. uptime 命令：up [N days,] H:MM
	matches := uptimeCommandRegex.FindStringSubmatch(output)
	if matches == nil {
		return 0, wrapError(ErrParse, "uptime: no `up` field in %q", firstLine(output))
	}

	var days, hours, minutes int
	if matches[1] != "" {
		days, _ = strconv.Atoi(matches[1])
	}

	switch {
	case matches[2] != "": // H:MM
		hours, _ = strconv.Atoi(matches[2])
		minutes, _ = strconv.Atoi(matches[3])
	case matches[4] != "": // N min
		minutes, _ = strconv.Atoi(matches[4])
	}

	return time.Duration(days)*24*time.Hour + time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute, nil
}

// ParseLoadAverage 解析系统负载均值，同时兼容两种输入：
//   - /proc/loadavg：`0.26 0.37 0.44 1/160 1941`（第 4 段为 可运行数/任务总数）
//   - uptime / top 头部：`load average: 0.22, 0.36, 0.43`
func ParseLoadAverage(output string) (LoadAverage, error) {
	// 1. 带 "load average" 前缀的形式（uptime、GNU top 头部）
	if matches := loadAverageCommandRegex.FindStringSubmatch(output); matches != nil {
		load := LoadAverage{}
		load.One, _ = strconv.ParseFloat(matches[1], 64)
		load.Five, _ = strconv.ParseFloat(matches[2], 64)
		load.Fifteen, _ = strconv.ParseFloat(matches[3], 64)
		return load, nil
	}

	// 2. /proc/loadavg 原样输出
	for _, line := range strings.Split(output, "\n") {
		if matches := loadAverageProcRegex.FindStringSubmatch(line); matches != nil {
			load := LoadAverage{}
			load.One, _ = strconv.ParseFloat(matches[1], 64)
			load.Five, _ = strconv.ParseFloat(matches[2], 64)
			load.Fifteen, _ = strconv.ParseFloat(matches[3], 64)
			load.Running, _ = strconv.Atoi(matches[4])
			load.Total, _ = strconv.Atoi(matches[5])
			return load, nil
		}

		if matches := loadAveragePlainRegex.FindStringSubmatch(line); matches != nil {
			load := LoadAverage{}
			load.One, _ = strconv.ParseFloat(matches[1], 64)
			load.Five, _ = strconv.ParseFloat(matches[2], 64)
			load.Fifteen, _ = strconv.ParseFloat(matches[3], 64)
			return load, nil
		}
	}

	return LoadAverage{}, wrapError(ErrParse, "load average: no matching line in %q", firstLine(output))
}

// ProcStat 是 /proc/stat 中 CPU 累计时间片的解析结果
//   - Cores 以 CPU 编号为键（如 "cpu0"、"cpu3"）：嵌入式设备的编号可能不连续
type ProcStat struct {
	Total CPUTimes            `json:"total"`
	Cores map[string]CPUTimes `json:"cores"`
}

// ParseProcStat 解析 /proc/stat 的 CPU 行（`cpu` 与 `cpuN`），忽略 intr/ctxt 等其它行
func ParseProcStat(output string) (*ProcStat, error) {
	stat := &ProcStat{Cores: make(map[string]CPUTimes)}
	found := false

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.HasPrefix(fields[0], "cpu") {
			continue
		}

		times, err := parseCPUTimes(fields[1:])
		if err != nil {
			return nil, wrapError(ErrParse, "proc stat %s: %v", fields[0], err)
		}

		if fields[0] == "cpu" {
			stat.Total = times
			found = true
			continue
		}

		// 过滤 cpufreq 之类的行：cpu 后缀必须是数字
		if _, err := strconv.Atoi(strings.TrimPrefix(fields[0], "cpu")); err != nil {
			continue
		}

		stat.Cores[fields[0]] = times
	}

	if !found {
		return nil, wrapError(ErrParse, "proc stat: no aggregate `cpu` line")
	}

	return stat, nil
}

// CalculateCPUUsage 由两次 /proc/stat 采样计算总体或单核使用率
//   - 返回值中的百分比取值 0~100，Used = 100 - Idle - IOWait
//   - 两次采样之间没有任何时间片变化时返回 ErrParse
func CalculateCPUUsage(prev, cur CPUTimes) (CPUUsage, error) {
	prevTotal, curTotal := prev.GetTotal(), cur.GetTotal()
	if curTotal <= prevTotal {
		return CPUUsage{}, wrapError(ErrParse, "cpu: no time elapsed between samples")
	}

	total := curTotal - prevTotal
	usage := CPUUsage{
		User:    deltaPercent(prev.User, cur.User, total),
		System:  deltaPercent(prev.System, cur.System, total),
		Nice:    deltaPercent(prev.Nice, cur.Nice, total),
		IOWait:  deltaPercent(prev.IOWait, cur.IOWait, total),
		IRQ:     deltaPercent(prev.IRQ, cur.IRQ, total),
		SoftIRQ: deltaPercent(prev.SoftIRQ, cur.SoftIRQ, total),
		Steal:   deltaPercent(prev.Steal, cur.Steal, total),
		Idle:    deltaPercent(prev.Idle, cur.Idle, total),
	}

	usage.Used = roundPercent(100 - usage.Idle - usage.IOWait)
	if usage.Used < 0 {
		usage.Used = 0
	}

	return usage, nil
}

// CPUUsageSet 是总体与各核心使用率的集合
type CPUUsageSet struct {
	Total CPUUsage            `json:"total"`
	Cores map[string]CPUUsage `json:"cores,omitempty"`
}

// CalculateCPUUsageSet 由两次 ProcStat 采样计算总体与各核心使用率
//   - 仅有单次采样、或某个核心在两次采样中不存在时跳过该核心
func CalculateCPUUsageSet(prev, cur *ProcStat) (*CPUUsageSet, error) {
	if prev == nil || cur == nil {
		return nil, wrapError(ErrInvalidArgument, "cpu: both samples are required")
	}

	total, err := CalculateCPUUsage(prev.Total, cur.Total)
	if err != nil {
		return nil, err
	}

	set := &CPUUsageSet{Total: total}
	for name, curTimes := range cur.Cores {
		prevTimes, ok := prev.Cores[name]
		if !ok {
			continue
		}

		usage, err := CalculateCPUUsage(prevTimes, curTimes)
		if err != nil {
			continue // 该核心在采样间隔内无变化，忽略
		}

		if set.Cores == nil {
			set.Cores = make(map[string]CPUUsage)
		}
		set.Cores[name] = usage
	}

	return set, nil
}

// ParseMemInfo 解析 /proc/meminfo
//   - 内核 3.14 之前没有 MemAvailable，此时按 Free + Buffers + Cached 推算并置
//     AvailableEstimated，此时 Used 等价于 `free` 的 "-/+ buffers/cache" 行
func ParseMemInfo(output string) (*MemInfo, error) {
	values := make(map[string]uint64)
	for _, line := range strings.Split(output, "\n") {
		name, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)

		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}

		val, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}

		unit := ""
		if len(fields) > 1 {
			unit = strings.ToLower(fields[1])
		}
		switch unit {
		case "kb", "kib":
			val *= 1 << 10
		case "mb", "mib":
			val *= 1 << 20
		case "gb", "gib":
			val *= 1 << 30
		case "": // 无单位时按字节处理
		}

		values[name] = val
	}

	total, ok := values["MemTotal"]
	if !ok {
		return nil, wrapError(ErrParse, "meminfo: MemTotal not found")
	}

	mem := MemoryStats{
		TotalBytes:   total,
		FreeBytes:    values["MemFree"],
		BuffersBytes: values["Buffers"],
		CachedBytes:  values["Cached"] + values["SReclaimable"],
	}

	if avail, ok := values["MemAvailable"]; ok {
		mem.AvailableBytes = avail
	} else {
		mem.AvailableBytes = mem.FreeBytes + mem.BuffersBytes + mem.CachedBytes
		mem.AvailableEstimated = true
	}
	if mem.AvailableBytes > total {
		mem.AvailableBytes = total
	}

	mem.UsedBytes = total - mem.AvailableBytes
	if total > 0 {
		mem.UsedPercent = roundPercent(float64(mem.UsedBytes) / float64(total) * 100)
	}

	swap := SwapStats{
		TotalBytes: values["SwapTotal"],
		FreeBytes:  values["SwapFree"],
	}
	if swap.TotalBytes > 0 {
		swap.UsedBytes = swap.TotalBytes - min(swap.FreeBytes, swap.TotalBytes)
		swap.UsedPercent = roundPercent(float64(swap.UsedBytes) / float64(swap.TotalBytes) * 100)
	}

	return &MemInfo{Memory: mem, Swap: swap}, nil
}

// CPUInfo 是 /proc/cpuinfo 的关键字段
type CPUInfo struct {
	Processors int    `json:"processors"`
	Model      string `json:"model,omitempty"`
}

// ParseCPUInfo 解析 /proc/cpuinfo，统计 processor 条目数并尽力提取型号
//   - x86 取 "model name"，ARM64 取 "model name"，ARMv7 取 "Processor" / "Hardware"
func ParseCPUInfo(output string) (CPUInfo, error) {
	var (
		info     CPUInfo
		modelKey = map[string]bool{"model name": true, "model": true, "processor": true, "hardware": true, "cpu model": true}
	)

	for _, line := range strings.Split(output, "\n") {
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.TrimSpace(value)

		// processor 的值为序号（0、1、2 ...），据此统计核心数
		if name == "processor" {
			if _, err := strconv.Atoi(value); err == nil {
				info.Processors++
				continue
			}
		}

		if info.Model == "" && modelKey[name] && value != "" {
			info.Model = value
		}
	}

	if info.Processors == 0 {
		return info, wrapError(ErrParse, "cpuinfo: no processor entry found")
	}

	return info, nil
}

// ParseNetDev 解析 /proc/net/dev
//   - 接收与发送各 8 列，仅保留字节、包、错误、丢包四项
func ParseNetDev(output string) ([]NetworkStats, error) {
	var (
		entries []NetworkStats
		lines   = strings.Split(output, "\n")
	)

	for _, line := range lines {
		name, rest, ok := strings.Cut(line, ":")
		if !ok || strings.Contains(line, "|") {
			continue // 跳过两行表头
		}

		fields := strings.Fields(rest)
		if len(fields) < 16 {
			continue
		}

		values := make([]uint64, 16)
		valid := true
		for i := 0; i < 16; i++ {
			val, err := strconv.ParseUint(fields[i], 10, 64)
			if err != nil {
				valid = false
				break
			}
			values[i] = val
		}
		if !valid {
			continue
		}

		entries = append(entries, NetworkStats{
			Name:      strings.TrimSpace(name),
			RxBytes:   values[0],
			RxPackets: values[1],
			RxErrors:  values[2],
			RxDropped: values[3],
			TxBytes:   values[8],
			TxPackets: values[9],
			TxErrors:  values[10],
			TxDropped: values[11],
		})
	}

	if len(entries) == 0 {
		return nil, wrapError(ErrParse, "net dev: no interface entry found")
	}

	return entries, nil
}

// parseCPUTimes 解析 /proc/stat CPU 行中 `cpu` 之后的时间片
//   - 依次为 user nice system idle iowait irq softirq steal [guest guest_nice]
//   - 老内核可能少于 8 列，缺失项按 0 处理
func parseCPUTimes(fields []string) (CPUTimes, error) {
	if len(fields) < 4 {
		return CPUTimes{}, wrapError(ErrParse, "expect at least 4 fields, got %d", len(fields))
	}

	values := make([]uint64, 8)
	for i := range values {
		if i >= len(fields) {
			break
		}

		val, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return CPUTimes{}, wrapError(ErrParse, "invalid field %q", fields[i])
		}
		values[i] = val
	}

	return CPUTimes{
		User:    values[0],
		Nice:    values[1],
		System:  values[2],
		Idle:    values[3],
		IOWait:  values[4],
		IRQ:     values[5],
		SoftIRQ: values[6],
		Steal:   values[7],
	}, nil
}

// deltaPercent 计算时间片增量占比
func deltaPercent(prev, cur, total uint64) float64 {
	if cur <= prev {
		return 0
	}

	delta := cur - prev
	if delta > total {
		delta = total
	}

	return roundPercent(float64(delta) / float64(total) * 100)
}

// roundPercent 保留两位小数，避免浮点误差导致的 99.99999 之类的展示
func roundPercent(v float64) float64 {
	return math.Round(v*100) / 100
}

// parseFloat 解析浮点数并返回是否成功
func parseFloat(s string) (float64, bool) {
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

// isFloat 判断字符串是否为浮点数
func isFloat(s string) bool {
	_, ok := parseFloat(s)
	return ok
}

// firstLine 返回首行内容，用于错误信息中展示原始输出片段
func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}

	return strings.TrimSpace(s)
}
