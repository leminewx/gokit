// Package percentile 提供流式、并发安全的百分位数（Percentile）计算能力
//   - 适用于监控指标（P50/P95/P99 延迟、响应耗时分布等）与统计分析场景。
//
// 设计目标：
//   - 零第三方依赖，纯标准库实现；
//   - 流式（streaming）接入：数据点可逐个 Add，无需预先收集全量样本；
//   - 并发安全：所有实现均通过 sync.RWMutex 保护，可在多 goroutine 下并发写入；
//   - 可合并（mergeable）：支持将多个独立实例的数据合并为一个，
//     便于分片采集后汇总（如每个 worker 各持一个实例，上报时 Merge）。
//
// 统一接口：
//
//	// Percentiler 百分位数计算器接口
//	type Percentiler interface {
//	    Add(value float64)                       // 添加一个数据点
//	    GetCount() int                           // 当前数据点总数
//	    Merge(other Percentiler) error           // 合并另一个同类型实例
//	    CalculatePercentile(p float64) (float64, error) // 计算百分位数 p∈[0,100]
//	}
//
// 三种实现，按需权衡精度、口径与内存/性能：
//
//  1. 精确模式（exactPercentile，NewExact）
//     保留全部原始数据点，按需排序后采用 R-7 线性插值法（与 Excel
//     PERCENTILE.INC、SciPy 默认一致）计算。
//     - 优点：结果完全精确；
//     - 缺点：内存随数据量线性增长；
//     - 适用：样本量不大（如万级及以下）且要求精确值的场景；
//     - 复杂度：添加 O(1)（懒排序，查询时 O(n log n)）。
//
//  2. 近似模式（tDigestPercentile，NewTDigest）
//     基于 T-Digest 算法（参考 github.com/tdunning/t-digest），以有限个
//     “质心（centroid）”对分布做有损压缩，在极端分位（如 P99.9）处精度更高。
//     - 优点：内存恒定（仅与压缩因子 compression 相关，通常取 100~200，
//     质心数约数百），支持流式与合并；
//     - 缺点：结果为近似值，平滑分布下误差通常 < 1%；
//     - 适用：海量数据流、长周期监控、分布式汇总；
//     - 复杂度：添加与查询均为 O(k)（k 为质心数）。
//
//  3. 最近秩模式（nearestRankPercentile，NewNearestRank）
//     采用标准最近秩法：第 p 百分位取下标 ceil(n*p/100)-1 的样本值，不做插值。
//     - 优点：结果必定是真实观测值，口径与多数压测工具（wrk、JMeter）一致；
//     - 缺点：内存随数据量线性增长，样本少时分位值跳动明显；
//     - 适用：延迟/耗时分位数报表、需要「分位数落在样本点上」的合规场景；
//     - 注意：与 R-7 插值结果不同（如 4 个样本的 P50 取第 2 个而非两点均值），
//     两套口径不可混用；已排序的切片可直接调用 CalculateNearestRank 免去实例开销。
//
// 典型用法：
//
//	// 精确模式
//	p := percentile.NewExact()
//	p.Add(1.2); p.Add(3.4); p.Add(5.6)
//	v, _ := p.CalculatePercentile(95) // 返回 P95
//
//	// 近似模式（压缩因子 100）
//	d := percentile.NewTDigest(100)
//	for _, sample := range samples {
//	    d.Add(sample)
//	}
//	p99, _ := d.CalculatePercentile(99)
//
//	// 最近秩模式
//	n := percentile.NewNearestRank()
//	n.Add(12); n.Add(30); n.Add(58)
//	p50, _ := n.CalculatePercentile(50)
//
// 注意：
//   - 计算分位前至少需存在一个数据点，否则返回 error；
//   - p 的取值范围为 [0, 100]，越界会返回 error；
//   - Merge 仅支持合并同类型实例（Exact 与 T-Digest 不可互并）。
package percentile

// Percentiler 百分位数计算器接口
type Percentiler interface {
	// Add 添加一个数据点（支持并发）
	Add(value float64)
	// GetCount 返回当前数据点总数
	GetCount() int
	// Merge 合并另一个同类型实现的 Percentiler 的数据
	Merge(other Percentiler) error
	// CalculatePercentile 计算给定百分位数 p (0 <= p <= 100) 的值
	CalculatePercentile(p float64) (float64, error)
}

// 编译期断言：三种算法均实现 Percentiler 接口
var (
	_ Percentiler = (*exactPercentile)(nil)
	_ Percentiler = (*tDigestPercentile)(nil)
	_ Percentiler = (*nearestRankPercentile)(nil)
)
