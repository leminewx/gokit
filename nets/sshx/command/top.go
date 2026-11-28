package command

import (
	"regexp"
	"strconv"
	"strings"
)

type (
	// TopParser 接口定义
	TopParser interface {
		Parse(output string) (*TopStats, error)
	}

	// TopStats 统一输出结构体，包含负载、CPU、内存和进程列表
	TopStats struct {
		CPU         topCPUStats       `json:"cpu"`
		Memory      topMemoryStats    `json:"memory"`
		Processes   []topProcessStats `json:"processes"`
		LoadAverage [3]float64        `json:"load_average"`
	}

	topCPUStats struct {
		User    float64 `json:"user"`
		System  float64 `json:"system"`
		Nice    float64 `json:"nice"`
		Idle    float64 `json:"idle"`
		IOWait  float64 `json:"io_wait"`
		HardIRQ float64 `json:"hard_irq"`
		SoftIRQ float64 `json:"soft_irq"`
		Steal   float64 `json:"steal"`
	}

	topMemoryStats struct {
		TotalMiB     float64 `json:"total_mib"`
		FreeMiB      float64 `json:"free_mib"`
		UsedMiB      float64 `json:"used_mib"`
		BuffCacheMiB float64 `json:"buff_cache_mib"`
		AvailMiB     float64 `json:"avail_mib"`
		UsedPercent  float64 `json:"used_percent"`
	}

	// topProcessStats 统一使用 ProcessStats，保留包内别名以免调用点大改
	topProcessStats = ProcessStats
)

// NewTopParser 根据一次 top 输出识别实现类型，返回对应解析器
//   - GNU procps：包含 `%Cpu(s):` 或 `%Cpu0 :`
//   - 旧式 procps / BusyBox：包含 `CPU:` 与 `Mem:`，其中带 PPID 列的为旧式
//   - BusyBox top 没有 `top -` 头部行，不能依赖首行判断
func NewTopParser(output []byte) TopParser {
	res := string(output)

	switch {
	case strings.Contains(res, "%Cpu(s):"), strings.Contains(res, "%Cpu0"):
		return &standardTopParser{}
	case strings.Contains(res, "CPU:") && strings.Contains(res, "Mem:"):
		// 旧式进程表通常带 PPID 列，嵌入式 BusyBox 没有
		if strings.Contains(res, "PPID") {
			return &oldTopParser{}
		}

		return &embeddedTopParser{}
	default:
		// 兜底：尝试旧式解析器，至少能拿到负载与 CPU 行
		return &oldTopParser{}
	}
}

// ParseTop 解析 top 输出，自动识别 GNU / 旧式 / 嵌入式格式
func ParseTop(output []byte) (*TopStats, error) {
	return NewTopParser(output).Parse(string(output))
}

// 编译期断言：各版本 top 输出解析器均实现 TopParser 接口
var (
	_ TopParser = (*standardTopParser)(nil)
	_ TopParser = (*oldTopParser)(nil)
	_ TopParser = (*embeddedTopParser)(nil)
)

// *********************************************************************************************************************
// 标准 Linux top 解析器
// *********************************************************************************************************************

type standardTopParser struct{}

var (
	standardTopLoadAverageRegex = regexp.MustCompile(`load average:\s+([\d\.]+),\s+([\d\.]+),\s+([\d\.]+)`)
	// GNU top 3.3+ 会按数值大小自动选择 KiB/MiB/GiB 单位
	standardTopMemoryRegex          = regexp.MustCompile(`(KiB|MiB|GiB|TiB) Mem\s*:\s*([\d\.]+)\s+total,\s*([\d\.]+)\s+free,\s*([\d\.]+)\s+used,\s*([\d\.]+)\s+buff/cache`)
	standardTopMemoryAvailableRegex = regexp.MustCompile(`([\d\.]+)\s+(KiB|MiB|GiB|TiB)?\s*avail Mem`)
	// 数值与前面的逗号之间可能没有空格（如 `ni,100.0 id`），因此统一用 \s*
	standardTopCPURegex = regexp.MustCompile(`%Cpu\(s\):\s*([\d\.]+)\s+us,\s*([\d\.]+)\s+sy,\s*([\d\.]+)\s+ni,\s*([\d\.]+)\s+id,\s*([\d\.]+)\s+wa,\s*([\d\.]+)\s+hi,\s*([\d\.]+)\s+si,\s*([\d\.]+)\s+st`)
	// top -1 等模式只输出逐核行 `%Cpu0 : ...`，需要按各核平均值近似整机
	standardTopPerCoreCPURegex = regexp.MustCompile(`%Cpu\d+\s*:\s*([\d\.]+)\s+us,\s*([\d\.]+)\s+sy,\s*([\d\.]+)\s+ni,\s*([\d\.]+)\s+id,\s*([\d\.]+)\s+wa,\s*([\d\.]+)\s+hi,\s*([\d\.]+)\s+si,\s*([\d\.]+)\s+st`)
)

func (*standardTopParser) Parse(output string) (*TopStats, error) {
	stats := &TopStats{}
	lines := strings.Split(output, "\n")

	// 解析负载
	for _, line := range lines {
		if matches := standardTopLoadAverageRegex.FindStringSubmatch(line); matches != nil {
			stats.LoadAverage[0], _ = strconv.ParseFloat(matches[1], 64)
			stats.LoadAverage[1], _ = strconv.ParseFloat(matches[2], 64)
			stats.LoadAverage[2], _ = strconv.ParseFloat(matches[3], 64)
			break
		}
	}

	// 解析 CPU：优先整机行；`top -1` 这类逐核输出按各核平均值近似整机
	for _, line := range lines {
		if matches := standardTopCPURegex.FindStringSubmatch(line); matches != nil {
			stats.CPU = parseTopCPUFields(matches)
			break
		}
	}
	if stats.CPU == (topCPUStats{}) {
		var (
			sum   topCPUStats
			count int
		)
		for _, line := range lines {
			if matches := standardTopPerCoreCPURegex.FindStringSubmatch(line); matches != nil {
				sum = addTopCPUStats(sum, parseTopCPUFields(matches))
				count++
			}
		}
		if count > 0 {
			stats.CPU = scaleTopCPUStats(sum, 1/float64(count))
		}
	}

	// 解析内存（GNU top 3.3+ 会按数值大小自动选择单位）
	memUnit := "MiB"
	for _, line := range lines {
		if matches := standardTopMemoryRegex.FindStringSubmatch(line); matches != nil {
			memUnit = matches[1]
			stats.Memory.TotalMiB = toMiB(matches[2], memUnit)
			stats.Memory.FreeMiB = toMiB(matches[3], memUnit)
			stats.Memory.UsedMiB = toMiB(matches[4], memUnit)
			stats.Memory.BuffCacheMiB = toMiB(matches[5], memUnit)
			if stats.Memory.TotalMiB > 0 {
				stats.Memory.UsedPercent = (stats.Memory.UsedMiB / stats.Memory.TotalMiB) * 100
			}
			break
		}
	}
	for _, line := range lines {
		if matches := standardTopMemoryAvailableRegex.FindStringSubmatch(line); matches != nil {
			// `avail Mem` 行的单位可省略（沿用在内存总览行上取到的单位）
			unit := matches[2]
			if unit == "" {
				unit = memUnit
			}
			stats.Memory.AvailMiB = toMiB(matches[1], unit)
			break
		}
	}

	// 解析进程表
	stats.Processes = parseTopProcessTable(lines)
	return stats.validate()
}

// validate 确认至少解析出一个字段，避免把无法识别的输出当作全零快照
func (own *TopStats) validate() (*TopStats, error) {
	if own.CPU == (topCPUStats{}) && own.Memory == (topMemoryStats{}) &&
		own.LoadAverage == [3]float64{} && len(own.Processes) == 0 {
		return nil, wrapError(ErrParse, "top: no recognizable field")
	}

	return own, nil
}

// buildTopMemory 将 KiB 单位的内存值汇总为 topMemoryStats
func buildTopMemory(used, free, buff, cached float64) topMemoryStats {
	memory := topMemoryStats{
		UsedMiB:      used / 1024,
		FreeMiB:      free / 1024,
		BuffCacheMiB: (buff + cached) / 1024,
	}

	if total := used + free + buff + cached; total > 0 {
		memory.TotalMiB = total / 1024
		memory.UsedPercent = used / total * 100
	}

	return memory
}

// parseTopCPUFields 将 `us, sy, ni, id, wa, hi, si, st` 的 8 个捕获组转为 topCPUStats
func parseTopCPUFields(matches []string) topCPUStats {
	values := make([]float64, 8)
	for idx := range values {
		values[idx], _ = strconv.ParseFloat(matches[idx+1], 64)
	}

	return topCPUStats{
		User:    values[0],
		System:  values[1],
		Nice:    values[2],
		Idle:    values[3],
		IOWait:  values[4],
		HardIRQ: values[5],
		SoftIRQ: values[6],
		Steal:   values[7],
	}
}

// addTopCPUStats 逐字段相加，用于求逐核平均值
func addTopCPUStats(left, right topCPUStats) topCPUStats {
	return topCPUStats{
		User:    left.User + right.User,
		System:  left.System + right.System,
		Nice:    left.Nice + right.Nice,
		Idle:    left.Idle + right.Idle,
		IOWait:  left.IOWait + right.IOWait,
		HardIRQ: left.HardIRQ + right.HardIRQ,
		SoftIRQ: left.SoftIRQ + right.SoftIRQ,
		Steal:   left.Steal + right.Steal,
	}
}

// scaleTopCPUStats 按比例缩放各字段
func scaleTopCPUStats(stats topCPUStats, factor float64) topCPUStats {
	return topCPUStats{
		User:    stats.User * factor,
		System:  stats.System * factor,
		Nice:    stats.Nice * factor,
		Idle:    stats.Idle * factor,
		IOWait:  stats.IOWait * factor,
		HardIRQ: stats.HardIRQ * factor,
		SoftIRQ: stats.SoftIRQ * factor,
		Steal:   stats.Steal * factor,
	}
}

// toMiB 将不同单位的内存值统一换算为 MiB
func toMiB(value, unit string) float64 {
	val, _ := strconv.ParseFloat(value, 64)

	switch strings.ToLower(unit) {
	case "kib", "kb":
		return val / 1024
	case "gib", "gb":
		return val * 1024
	case "tib", "tb":
		return val * 1024 * 1024
	default: // MiB 或未标注单位
		return val
	}
}

// parseTopProcessTable 解析 top 进程表
//   - GNU、旧式 procps 与 BusyBox top 的表头列名基本一致，因此统一按列名定位：
//     PID、USER、%CPU、%MEM、VIRT/VSZ、RES/RSS、S/STAT、COMMAND
//   - 终端过窄时 GNU top 会裁掉 %CPU/%MEM 列，此时对应字段保持 0
//   - `%VSZ` 这类百分比列不参与解析（否则会把百分比当成容量）
func parseTopProcessTable(lines []string) []ProcessStats {
	var (
		columns   map[string]int
		processes []ProcessStats
		inTable   bool
	)

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			if inTable {
				break // 空行表示进程表结束
			}
			continue
		}

		if !inTable {
			if idx, ok := parsePSHeader(fields); ok {
				columns = idx
				inTable = true
			}
			continue
		}

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
			continue
		}

		processes = append(processes, ProcessStats{
			PID:        pid,
			PPID:       parseIntColumn(fields, columns, _PS_COL_PPID),
			User:       parseStringColumn(fields, columns, _PS_COL_USER),
			State:      parseStringColumn(fields, columns, _PS_COL_STATE),
			CPUPercent: parseFloatColumn(fields, columns, _PS_COL_CPU),
			MEMPercent: parseFloatColumn(fields, columns, _PS_COL_MEM),
			RSSBytes:   parseKiBColumn(fields, columns, _PS_COL_RSS),
			VSZBytes:   parseKiBColumn(fields, columns, _PS_COL_VSZ),
			Command:    strings.Join(fields[cmdIdx:], " "),
		})
	}

	return processes
}

// *********************************************************************************************************************
// 旧式 Linux top 解析器（含 PPID、STAT 列）
// *********************************************************************************************************************

type oldTopParser struct{}

var (
	oldTopLoadAverageRegex = regexp.MustCompile(`(?i)Load average:\s+([\d\.]+)\s+([\d\.]+)\s+([\d\.]+)`)
	// BusyBox 的内存值可能带 K 或 M 后缀
	oldTopMemoryRegex = regexp.MustCompile(`Mem:\s+(\d+)([KM]?) used,\s+(\d+)([KM]?) free,\s+(\d+)([KM]?) shrd,\s+(\d+)([KM]?) buff,\s+(\d+)([KM]?) cached`)
	oldTopCPURegex    = regexp.MustCompile(`CPU:\s+([\d\.]+)%\s+usr\s+([\d\.]+)%\s+sys\s+([\d\.]+)%\s+nic\s+([\d\.]+)%\s+idle\s+([\d\.]+)%\s+io\s+([\d\.]+)%\s+irq\s+([\d\.]+)%\s+sirq`)
)

func (*oldTopParser) Parse(output string) (*TopStats, error) {
	stats := &TopStats{}
	lines := strings.Split(output, "\n")

	// 负载
	for _, line := range lines {
		if matches := oldTopLoadAverageRegex.FindStringSubmatch(line); matches != nil {
			stats.LoadAverage[0], _ = strconv.ParseFloat(matches[1], 64)
			stats.LoadAverage[1], _ = strconv.ParseFloat(matches[2], 64)
			stats.LoadAverage[2], _ = strconv.ParseFloat(matches[3], 64)
			break
		}
	}

	// CPU: usr% sys% nic% idle% io% irq% sirq%
	for _, line := range lines {
		if matches := oldTopCPURegex.FindStringSubmatch(line); matches != nil {
			stats.CPU.User, _ = strconv.ParseFloat(matches[1], 64)
			stats.CPU.System, _ = strconv.ParseFloat(matches[2], 64)
			stats.CPU.Nice, _ = strconv.ParseFloat(matches[3], 64)
			stats.CPU.Idle, _ = strconv.ParseFloat(matches[4], 64)
			stats.CPU.IOWait, _ = strconv.ParseFloat(matches[5], 64)
			stats.CPU.HardIRQ, _ = strconv.ParseFloat(matches[6], 64)
			stats.CPU.SoftIRQ, _ = strconv.ParseFloat(matches[7], 64)
			break
		}
	}

	// 内存：Mem: used free shrd buff cached
	for _, line := range lines {
		if matches := oldTopMemoryRegex.FindStringSubmatch(line); matches != nil {
			used := parseMemValue(matches[1], matches[2])
			free := parseMemValue(matches[3], matches[4])
			buff := parseMemValue(matches[7], matches[8])
			cached := parseMemValue(matches[9], matches[10])
			stats.Memory = buildTopMemory(used, free, buff, cached)
			break
		}
	}

	// 进程表解析（旧式表头含 PID PPID USER STAT ... COMMAND）
	stats.Processes = parseTopProcessTable(lines)
	return stats.validate()
}

// *********************************************************************************************************************
// 嵌入式 BusyBox top 解析器（无 PPID，可能无 %CPU 列）
// *********************************************************************************************************************

var (
	embeddedTopLoadAverageRegex = regexp.MustCompile(`(?i)Load average:\s+([\d\.]+)\s+([\d\.]+)\s+([\d\.]+)`)
	embeddedTopMemoryRegex      = regexp.MustCompile(`Mem:\s+(\d+)([KM]?) used,\s+(\d+)([KM]?) free,\s+(\d+)([KM]?) shrd,\s+(\d+)([KM]?) buff,\s+(\d+)([KM]?) cached`)
	embeddedTopCPURegex         = regexp.MustCompile(`CPU:\s+([\d\.]+)%\s+usr\s+([\d\.]+)%\s+sys\s+([\d\.]+)%\s+nic\s+([\d\.]+)%\s+idle\s+([\d\.]+)%\s+io\s+([\d\.]+)%\s+irq\s+([\d\.]+)%\s+sirq`)
)

type embeddedTopParser struct{}

func (*embeddedTopParser) Parse(output string) (*TopStats, error) {
	stats := &TopStats{}
	lines := strings.Split(output, "\n")

	// 负载（与旧式相同）
	for _, line := range lines {
		if matches := embeddedTopLoadAverageRegex.FindStringSubmatch(line); matches != nil {
			stats.LoadAverage[0], _ = strconv.ParseFloat(matches[1], 64)
			stats.LoadAverage[1], _ = strconv.ParseFloat(matches[2], 64)
			stats.LoadAverage[2], _ = strconv.ParseFloat(matches[3], 64)
			break
		}
	}

	// CPU 行（与旧式相同）
	for _, line := range lines {
		if matches := embeddedTopCPURegex.FindStringSubmatch(line); matches != nil {
			stats.CPU.User, _ = strconv.ParseFloat(matches[1], 64)
			stats.CPU.System, _ = strconv.ParseFloat(matches[2], 64)
			stats.CPU.Nice, _ = strconv.ParseFloat(matches[3], 64)
			stats.CPU.Idle, _ = strconv.ParseFloat(matches[4], 64)
			stats.CPU.IOWait, _ = strconv.ParseFloat(matches[5], 64)
			stats.CPU.HardIRQ, _ = strconv.ParseFloat(matches[6], 64)
			stats.CPU.SoftIRQ, _ = strconv.ParseFloat(matches[7], 64)
			break
		}
	}

	// 内存（单位可能是 K 或 M）
	for _, line := range lines {
		if matches := embeddedTopMemoryRegex.FindStringSubmatch(line); matches != nil {
			used := parseMemValue(matches[1], matches[2])
			free := parseMemValue(matches[3], matches[4])
			buff := parseMemValue(matches[7], matches[8])
			cached := parseMemValue(matches[9], matches[10])
			stats.Memory = buildTopMemory(used, free, buff, cached)
			break
		}
	}

	// 嵌入式进程表：表头可能是 "  PID  PPID USER     STAT   VSZ %VSZ CPU %CPU COMMAND"
	// 或者更简化的 "  PID  USER     STAT   VSZ %VSZ CPU %CPU COMMAND"
	stats.Processes = parseTopProcessTable(lines)
	return stats.validate()
}

// parseMemValue 将带单位的内存值统一换算为 KiB
func parseMemValue(val, unit string) float64 {
	v, _ := strconv.ParseFloat(val, 64)
	switch strings.ToLower(unit) {
	case "k", "kb", "kib":
		return v
	case "m", "mb", "mib":
		return v * 1024
	case "g", "gb", "gib":
		return v * 1024 * 1024
	default:
		return v
	}
}
