package command

import (
	"fmt"
	"io"
	"time"
)

// LoadAverage 系统负载均值
//   - 取自 /proc/loadavg 或 uptime 的 "load average" 字段
//   - Running / Total 来自 /proc/loadavg 第 4 段的 "1/160"，缺失时为 0
type LoadAverage struct {
	One     float64 `json:"one"`
	Five    float64 `json:"five"`
	Fifteen float64 `json:"fifteen"`
	Running int     `json:"running,omitempty"` // 可运行任务数
	Total   int     `json:"total,omitempty"`   // 任务（线程）总数
}

// String 返回 "0.26 0.37 0.44" 形式
func (own LoadAverage) String() string {
	return fmt.Sprintf("%.2f %.2f %.2f", own.One, own.Five, own.Fifteen)
}

// CPUTimes 是 /proc/stat 中 CPU 行累计的时间片计数
//   - 单位为 USER_HZ（Linux 上通常为 1/100 秒），只用于两次采样求差
type CPUTimes struct {
	User    uint64 `json:"user"`
	Nice    uint64 `json:"nice"`
	System  uint64 `json:"system"`
	Idle    uint64 `json:"idle"`
	IOWait  uint64 `json:"io_wait"`
	IRQ     uint64 `json:"hard_irq"`
	SoftIRQ uint64 `json:"soft_irq"`
	Steal   uint64 `json:"steal"`
}

// GetTotal 返回所有时间片之和
func (own CPUTimes) GetTotal() uint64 {
	return own.User + own.Nice + own.System + own.Idle + own.IOWait + own.IRQ + own.SoftIRQ + own.Steal
}

// CPUUsage CPU 使用率（百分比，取值 0~100）
type CPUUsage struct {
	User    float64 `json:"user"`
	System  float64 `json:"system"`
	Nice    float64 `json:"nice"`
	IOWait  float64 `json:"io_wait"`
	IRQ     float64 `json:"hard_irq"`
	SoftIRQ float64 `json:"soft_irq"`
	Steal   float64 `json:"steal"`
	Idle    float64 `json:"idle"`
	// Used 为 busy 占比，等于 100 - Idle - IOWait
	Used float64 `json:"used"`
}

// String 返回紧凑的百分比描述
func (own CPUUsage) String() string {
	return fmt.Sprintf("used %.1f%% | user %.1f%% system %.1f%% nice %.1f%% iowait %.1f%% irq %.1f%% sirq %.1f%% steal %.1f%% idle %.1f%%",
		own.Used, own.User, own.System, own.Nice, own.IOWait, own.IRQ, own.SoftIRQ, own.Steal, own.Idle)
}

// MemoryStats 物理内存统计（单位字节）
type MemoryStats struct {
	TotalBytes     uint64  `json:"total_bytes"`
	FreeBytes      uint64  `json:"free_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	BuffersBytes   uint64  `json:"buffers_bytes"`
	CachedBytes    uint64  `json:"cached_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	UsedPercent    float64 `json:"used_percent"`
	// AvailableEstimated 为 true 表示 AvailableBytes 由 Free + Buffers + Cached 推算，
	// 多见于内核 3.14 之前（/proc/meminfo 尚无 MemAvailable 字段）的嵌入式设备
	AvailableEstimated bool `json:"available_estimated,omitempty"`
}

// String 返回紧凑的内存摘要
func (own MemoryStats) String() string {
	return fmt.Sprintf("used %.2f%% | available %s / %s", own.UsedPercent,
		humanBytes(own.AvailableBytes), humanBytes(own.TotalBytes))
}

// SwapStats 交换分区统计（单位字节）
type SwapStats struct {
	TotalBytes  uint64  `json:"total_bytes"`
	FreeBytes   uint64  `json:"free_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

// MemInfo 是 /proc/meminfo 的解析结果
type MemInfo struct {
	Memory MemoryStats `json:"memory"`
	Swap   SwapStats   `json:"swap"`
}

// FilesystemStats 单个挂载点的容量统计（单位字节）
type FilesystemStats struct {
	Filesystem  string  `json:"filesystem"`
	MountedOn   string  `json:"mounted_on"`
	TotalBytes  uint64  `json:"total_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	FreeBytes   uint64  `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

// NetworkStats 单个网卡的收发与错误统计（取自 /proc/net/dev）
type NetworkStats struct {
	Name      string `json:"name"`
	RxBytes   uint64 `json:"rx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	RxErrors  uint64 `json:"rx_errors"`
	RxDropped uint64 `json:"rx_dropped"`
	TxBytes   uint64 `json:"tx_bytes"`
	TxPackets uint64 `json:"tx_packets"`
	TxErrors  uint64 `json:"tx_errors"`
	TxDropped uint64 `json:"tx_dropped"`
}

// ProcessStats 单个进程的统计信息
//   - CPUPercent / MEMPercent 仅在 ps 或 top 输出包含对应列时有效，否则为 0
//   - RSSBytes / VSZBytes 由 KiB 换算而来
type ProcessStats struct {
	PID        int     `json:"pid"`
	PPID       int     `json:"ppid"`
	User       string  `json:"user"`
	State      string  `json:"state,omitempty"`
	CPUPercent float64 `json:"cpu_percent"`
	MEMPercent float64 `json:"mem_percent"`
	RSSBytes   uint64  `json:"rss_bytes,omitempty"`
	VSZBytes   uint64  `json:"vsz_bytes,omitempty"`
	Command    string  `json:"command"`
}

// Snapshot 一次采集得到的整机性能快照
type Snapshot struct {
	Host           string              `json:"host"`
	Kernel         string              `json:"kernel"`
	CollectedAt    time.Time           `json:"collected_at"`
	SampleInterval time.Duration       `json:"sample_interval"`
	Uptime         time.Duration       `json:"uptime"`
	CPUCores       int                 `json:"cpu_cores"`
	Load           LoadAverage         `json:"load_average"`
	CPU            CPUUsage            `json:"cpu"`
	CoreUsage      map[string]CPUUsage `json:"cpu_per_core,omitempty"`
	Memory         MemoryStats         `json:"memory"`
	Swap           SwapStats           `json:"swap"`
	Filesystems    []FilesystemStats   `json:"filesystems,omitempty"`
	Networks       []NetworkStats      `json:"networks,omitempty"`
}

// 编译期断言：快照可按人类可读文本写出
var _ io.WriterTo = (*Snapshot)(nil)
