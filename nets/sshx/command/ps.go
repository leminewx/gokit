package command

import (
	"strconv"
	"strings"
)

// ps 列名到字段索引的映射键
const (
	_PS_COL_PID     = "pid"
	_PS_COL_PPID    = "ppid"
	_PS_COL_USER    = "user"
	_PS_COL_CPU     = "cpu"
	_PS_COL_MEM     = "mem"
	_PS_COL_RSS     = "rss"
	_PS_COL_VSZ     = "vsz"
	_PS_COL_STATE   = "state"
	_PS_COL_COMMAND = "command"
)

// ParsePS 解析 ps 输出，按表头定位列，兼容以下常见形式：
//   - `ps aux`（procps）：USER PID %CPU %MEM VSZ RSS TTY STAT START TIME COMMAND
//   - `ps -eo pid,ppid,user,pcpu,pmem,args`：PID PPID USER %CPU %MEM COMMAND
//   - BusyBox / ToyBox：`ps`、`ps -o pid,ppid,user,rss,stat,args`（无 %CPU/%MEM 列，对应字段为 0）
//
// 表头缺失时返回 ErrParse；命令列之后的全部字段都会被拼入 Command（含空格参数）。
func ParsePS(output string) ([]ProcessStats, error) {
	var entries []ProcessStats
	columns := map[string]int{}
	found := false

	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)

		// 1. 定位表头：包含 PID 与命令列
		if !found {
			if idx, ok := parsePSHeader(fields); ok {
				columns = idx
				found = true
			}
			continue
		}

		// 2. 解析数据行
		cmdIdx, ok := columns[_PS_COL_COMMAND]
		if !ok || cmdIdx >= len(fields) {
			continue
		}

		pidIdx, ok := columns[_PS_COL_PID]
		if !ok || pidIdx >= len(fields) {
			continue
		}

		pid, err := strconv.Atoi(fields[pidIdx])
		if err != nil {
			continue // 表头重复出现或非数据行
		}

		proc := ProcessStats{PID: pid, Command: strings.Join(fields[cmdIdx:], " ")}
		proc.PPID = parseIntColumn(fields, columns, _PS_COL_PPID)
		proc.User = parseStringColumn(fields, columns, _PS_COL_USER)
		proc.State = parseStringColumn(fields, columns, _PS_COL_STATE)
		proc.CPUPercent = parseFloatColumn(fields, columns, _PS_COL_CPU)
		proc.MEMPercent = parseFloatColumn(fields, columns, _PS_COL_MEM)
		proc.RSSBytes = parseKiBColumn(fields, columns, _PS_COL_RSS)
		proc.VSZBytes = parseKiBColumn(fields, columns, _PS_COL_VSZ)

		entries = append(entries, proc)
	}

	if !found {
		return nil, wrapError(ErrParse, "ps: table header not found")
	}

	return entries, nil
}

// parsePSHeader 解析 ps 表头，返回列名到索引的映射
//   - 表头必须同时包含 PID 与 COMMAND/CMD/ARGS
func parsePSHeader(fields []string) (map[string]int, bool) {
	columns := make(map[string]int, len(fields))
	for idx, field := range fields {
		if key := normalizePSColumn(field); key != "" {
			columns[key] = idx
		}
	}

	_, hasPID := columns[_PS_COL_PID]
	_, hasCmd := columns[_PS_COL_COMMAND]
	return columns, hasPID && hasCmd
}

// normalizePSColumn 归一化 ps / top 列名
//   - 去掉 `+` 等修饰并统一小写
//   - `CMD` / `ARGS` 视为命令列，`S` 视为状态列
//   - 百分号只在 `%CPU` / `%MEM` 上有效：top 的 `%VSZ` 是占比而不是容量，
//     裸 `CPU` 列（BusyBox top）是核心编号而不是使用率，均不参与解析
func normalizePSColumn(field string) string {
	field = strings.ToLower(strings.TrimSuffix(field, "+"))

	if strings.HasPrefix(field, "%") {
		switch field {
		case "%cpu":
			return _PS_COL_CPU
		case "%mem":
			return _PS_COL_MEM
		default:
			return ""
		}
	}

	switch field {
	case "pid":
		return _PS_COL_PID
	case "ppid":
		return _PS_COL_PPID
	case "user", "uid", "ruser":
		return _PS_COL_USER
	case "rss", "res":
		return _PS_COL_RSS
	case "vsz", "virt":
		return _PS_COL_VSZ
	case "stat", "s":
		return _PS_COL_STATE
	case "command", "cmd", "args":
		return _PS_COL_COMMAND
	default:
		return ""
	}
}

// parseStringColumn 读取字符串列
func parseStringColumn(fields []string, columns map[string]int, key string) string {
	idx, ok := columns[key]
	if !ok || idx >= len(fields) {
		return ""
	}

	return fields[idx]
}

// parseIntColumn 读取整型列
func parseIntColumn(fields []string, columns map[string]int, key string) int {
	idx, ok := columns[key]
	if !ok || idx >= len(fields) {
		return 0
	}

	val, _ := strconv.Atoi(fields[idx])
	return val
}

// parseFloatColumn 读取浮点列
func parseFloatColumn(fields []string, columns map[string]int, key string) float64 {
	idx, ok := columns[key]
	if !ok || idx >= len(fields) {
		return 0
	}

	val, _ := strconv.ParseFloat(fields[idx], 64)
	return val
}

// parseKiBColumn 读取以 KiB 为单位的容量列并换算为字节
func parseKiBColumn(fields []string, columns map[string]int, key string) uint64 {
	idx, ok := columns[key]
	if !ok || idx >= len(fields) {
		return 0
	}

	val, err := strconv.ParseUint(fields[idx], 10, 64)
	if err != nil {
		return 0
	}

	return val << 10
}
