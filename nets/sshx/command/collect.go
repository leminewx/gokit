package command

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"gitee.com/leminewx/gokit/nets/sshx"
)

const (
	// 两次 /proc/stat 采样之间的默认间隔
	_DEFAULT_SAMPLE_INTERVAL = time.Second
	// 单次采集的默认超时时间（含采样间隔）
	_DEFAULT_COLLECT_TIMEOUT = 20 * time.Second
	// 分段标记前缀：远端脚本用它把多段输出分隔开
	_SECTION_PREFIX = "###gokit:"
)

// Collector 通过 SSH 采集 Linux 设备的通用性能指标
//   - 单次 Collect 只建立一个会话，脚本内分段输出，避免多次往返
//   - CPU 使用率需要两次 /proc/stat 采样求差，因此采集耗时不小于采样间隔
type Collector struct {
	client   *sshx.Client
	interval time.Duration
	timeout  time.Duration
}

// Option 采集器选项
type Option func(*Collector)

// WithSampleInterval 设置两次 CPU 采样的间隔（默认 1 秒）
//   - 脚本内按整秒 sleep：非整秒向上取整，不足 1 秒抬升为 1 秒
//   - BusyBox 的 sleep 不支持小数秒
func WithSampleInterval(interval time.Duration) Option {
	return func(own *Collector) {
		if interval > 0 {
			own.interval = interval
		}
	}
}

// WithCollectTimeout 设置单次采集的超时时间（默认 20 秒）
func WithCollectTimeout(timeout time.Duration) Option {
	return func(own *Collector) {
		if timeout > 0 {
			own.timeout = timeout
		}
	}
}

// NewCollector 新建采集器
func NewCollector(client *sshx.Client, opts ...Option) *Collector {
	collector := &Collector{
		client:   client,
		interval: _DEFAULT_SAMPLE_INTERVAL,
		timeout:  _DEFAULT_COLLECT_TIMEOUT,
	}

	for _, opt := range opts {
		opt(collector)
	}

	return collector
}

// Collect 采集一次整机性能快照
//   - 主机名、内核、运行时长、负载、CPU、内存、交换区、磁盘、网卡均尽力采集，
//     单段解析失败只影响该段（并汇总到返回错误中，仅在没有任何有效数据时返回）
//   - ctx 需要并发控制时由调用方传入，nil 表示使用默认超时
func (own *Collector) Collect(ctx context.Context) (*Snapshot, error) {
	if own == nil || own.client == nil {
		return nil, wrapError(ErrInvalidArgument, "collector: client is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	started := time.Now()

	// 采集耗时取决于采样间隔，由采集器在 ctx 上兜底超时；
	// 调用方传入的 deadline 更短时以调用方为准
	if own.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, own.timeout)
		defer cancel()
	}

	// 采集脚本内部已重定向大部分错误输出，这里再兜底过滤 stderr 与非零退出码，
	// 避免个别设备缺少某个文件时整次采集失败
	output, err := own.client.ExecuteContext(ctx, &sshx.Command{
		Command:        own.script(),
		ExitCodeFilter: func(int) bool { return true },
		StderrFilter:   func(error) bool { return true },
	})

	sections := splitSections(string(output))
	if len(sections) == 0 {
		if err != nil {
			return nil, wrapError(ErrNoData, "collect: no section returned: %v", err)
		}

		return nil, wrapError(ErrNoData, "collect: no section marker in output")
	}

	snapshot := &Snapshot{
		CollectedAt:    started,
		SampleInterval: time.Duration(own.sampleSeconds()) * time.Second,
		Host:           firstLine(sections["hostname"]),
		Kernel:         firstLine(sections["kernel"]),
	}

	var parseErrs []error

	if value := sections["cpuinfo"]; value != "" {
		snapshot.CPUCores = parseCPUCores(value)
	}

	if d, err := ParseUptime(sections["uptime"]); err == nil {
		snapshot.Uptime = d
	} else if sections["uptime"] != "" {
		parseErrs = append(parseErrs, err)
	}

	if load, err := ParseLoadAverage(sections["loadavg"]); err == nil {
		snapshot.Load = load
	} else if sections["loadavg"] != "" {
		parseErrs = append(parseErrs, err)
	}

	prev, errPrev := ParseProcStat(sections["stat1"])
	cur, errCur := ParseProcStat(sections["stat2"])
	switch {
	case errPrev == nil && errCur == nil:
		if set, err := CalculateCPUUsageSet(prev, cur); err == nil {
			snapshot.CPU = set.Total
			snapshot.CoreUsage = set.Cores
		} else {
			parseErrs = append(parseErrs, err)
		}

		if snapshot.CPUCores == 0 {
			snapshot.CPUCores = len(cur.Cores)
		}
	default:
		if errCur != nil {
			parseErrs = append(parseErrs, errCur)
		} else if errPrev != nil {
			parseErrs = append(parseErrs, errPrev)
		}

		if errCur == nil && snapshot.CPUCores == 0 {
			snapshot.CPUCores = len(cur.Cores)
		}
	}

	if info, err := ParseMemInfo(sections["meminfo"]); err == nil {
		snapshot.Memory = info.Memory
		snapshot.Swap = info.Swap
	} else if sections["meminfo"] != "" {
		parseErrs = append(parseErrs, err)
	}

	if networks, err := ParseNetDev(sections["netdev"]); err == nil {
		snapshot.Networks = networks
	} else if sections["netdev"] != "" {
		parseErrs = append(parseErrs, err)
	}

	if filesystems, err := ParseDf(sections["df"]); err == nil {
		snapshot.Filesystems = filesystems
	} else if sections["df"] != "" {
		parseErrs = append(parseErrs, err)
	}

	// 关键指标（CPU 与内存）都缺失时，说明设备输出与解析器不匹配，需要暴露错误
	if snapshot.Memory.TotalBytes == 0 && snapshot.CPU.Idle == 0 && snapshot.CPU.Used == 0 {
		if err != nil {
			parseErrs = append(parseErrs, err)
		}

		if len(parseErrs) > 0 {
			return nil, errors.Join(parseErrs...)
		}

		return nil, wrapError(ErrNoData, "collect: cpu and memory are unavailable")
	}

	return snapshot, nil
}

// sampleSeconds 返回脚本中 sleep 的整秒数
//   - 向上取整，保证实际采样间隔不小于配置值
//   - 最小为 1 秒：BusyBox 的 sleep 不支持小数秒
func (own *Collector) sampleSeconds() int {
	seconds := int(math.Ceil(own.interval.Seconds()))
	if seconds < 1 {
		return 1
	}

	return seconds
}

// script 生成采集脚本
//   - 只用 POSIX sh 与 /proc，兼容 GNU coreutils、BusyBox 与 ToyBox
//   - 每段前输出标记行，便于按段解析；冗余命令统一丢弃 stderr
func (own *Collector) script() string {
	seconds := own.sampleSeconds()

	var builder strings.Builder
	// 远端 locale 会翻译 df/top 的表头与关键字，统一为 C 避免解析失败
	builder.WriteString("export LC_ALL=C\n")

	section := func(name, command string) {
		fmt.Fprintf(&builder, "echo '%s%s'\n%s\n", _SECTION_PREFIX, name, command)
	}

	section("hostname", "uname -n 2>/dev/null || hostname 2>/dev/null")
	section("kernel", "uname -srm 2>/dev/null")
	section("uptime", "cat /proc/uptime 2>/dev/null")
	section("loadavg", "cat /proc/loadavg 2>/dev/null")
	section("cpuinfo", "grep -c '^processor' /proc/cpuinfo 2>/dev/null")
	section("stat1", "grep '^cpu' /proc/stat 2>/dev/null")
	fmt.Fprintf(&builder, "sleep %d\n", seconds)
	section("stat2", "grep '^cpu' /proc/stat 2>/dev/null")
	section("meminfo", "cat /proc/meminfo 2>/dev/null")
	section("netdev", "cat /proc/net/dev 2>/dev/null")
	section("df", "df -P -k 2>/dev/null")

	return builder.String()
}

// splitSections 按标记行切分脚本输出
//   - 标记行形如 `###gokit:meminfo`
//   - 未以标记开头的前导内容被忽略
func splitSections(output string) map[string]string {
	var (
		sections = make(map[string]string)
		builders = make(map[string]*strings.Builder)
		current  string
	)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSuffix(line, "\r")

		if strings.HasPrefix(line, _SECTION_PREFIX) {
			current = strings.TrimSpace(strings.TrimPrefix(line, _SECTION_PREFIX))
			builders[current] = &strings.Builder{}
			continue
		}

		if current == "" {
			continue
		}

		builders[current].WriteString(line)
		builders[current].WriteByte('\n')
	}

	for name, builder := range builders {
		sections[name] = builder.String()
	}

	return sections
}

// parseCPUCores 解析 `grep -c '^processor' /proc/cpuinfo` 的输出
func parseCPUCores(output string) int {
	cores, err := strconv.Atoi(strings.TrimSpace(firstLine(output)))
	if err != nil || cores < 0 {
		return 0
	}

	return cores
}
