package percentile

import (
	"math"
	"sort"
	"sync"
)

// *********************************************************************************************************************
// T-Digest 近似模式：
// 支持流式、可合并，误差小于1%（数据分布光滑时更优）
// 添加和查询均为 O(k)（k为质心数，~几百），内存恒定
// *********************************************************************************************************************

type (
	// tDigestPercentile 实现了 T-Digest 近似百分位数算法
	// 参考文献：https://github.com/tdunning/t-digest
	tDigestPercentile struct {
		count       int        // 总数据点数
		min, max    float64    // 记录最小最大值，提高边界百分位数精度
		compression float64    // 压缩因子，控制质心数量（越大越精确），通常取 100
		centroids   []centroid // 按均值排序的质心列表
		lock        sync.RWMutex
	}

	// centroid 表示一个质心（中心值和权重）
	centroid struct {
		mean   float64 // 质心的平均值
		weight float64 // 该质心包含的数据点数
	}
)

// NewTDigest 创建一个 T-Digest 实例，compression 建议 100~200
func NewTDigest(compression float64) Percentiler {
	if compression <= 0 {
		compression = 100
	}

	return &tDigestPercentile{
		centroids:   make([]centroid, 0),
		compression: compression,
		min:         math.Inf(1),
		max:         math.Inf(-1),
	}
}

// Add 添加一个数据点
func (own *tDigestPercentile) Add(value float64) {
	own.lock.Lock()
	own._add(value, 1)
	own.lock.Unlock()
}

// _add 内部添加，不加锁，需外部持锁
func (own *tDigestPercentile) _add(value float64, weight float64) {
	// 更新 min/max
	if value < own.min {
		own.min = value
	}
	if value > own.max {
		own.max = value
	}

	own.count += int(weight)

	// 寻找插入位置（二分查找）
	idx := sort.Search(len(own.centroids), func(i int) bool {
		return own.centroids[i].mean >= value
	})

	// 计算当前数据对应的近似分位数位置（用于合并阈值）
	// 使用累积权重估算：rank = (own.count - int(weight)) + 1? 简化：使用中位数0.5
	// 更精确：根据插入位置估算该值在分布中的分位点
	q := own._estimateQuantile(idx, value)

	// 尝试合并到相邻质心
	merged := false
	if idx < len(own.centroids) {
		merged = own._tryMerge(idx, value, weight, q)
	}
	if !merged && idx > 0 {
		// 尝试与前一个合并，使用前一个质心的分位数
		qPrev := own._centroidQuantile(idx - 1)
		merged = own._tryMerge(idx-1, value, weight, qPrev)
	}
	if !merged {
		// 无法合并，插入新质心
		newCent := centroid{mean: value, weight: weight}
		own.centroids = append(own.centroids, centroid{})
		copy(own.centroids[idx+1:], own.centroids[idx:])
		own.centroids[idx] = newCent
	}

	// 压缩质心列表：当质心数量超过压缩因子的3倍时触发
	if len(own.centroids) > 3*int(own.compression) {
		own._compress()
	}
}

// _estimateQuantile 估算指定索引附近值的分位数（0~1）
func (own *tDigestPercentile) _estimateQuantile(idx int, value float64) float64 {
	if own.count == 0 {
		return 0.5
	}

	var cumBefore float64
	for i := 0; i < idx && i < len(own.centroids); i++ {
		cumBefore += own.centroids[i].weight
	}

	// 确定左右的边界（均值）
	left := own.min
	if idx > 0 {
		left = own.centroids[idx-1].mean
	}

	right := own.max
	if idx < len(own.centroids) {
		right = own.centroids[idx].mean
	}
	if right <= left {
		// 退化情况：使用中点估算
		return (cumBefore + 0.5) / float64(own.count)
	}

	// 限制 t 在 [0,1] 内
	t := (value - left) / (right - left)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	return (cumBefore + t) / float64(own.count)
}

// _centroidQuantile 返回第 i 个质心代表的分位数（累积中点）
func (own *tDigestPercentile) _centroidQuantile(i int) float64 {
	if own.count == 0 || i < 0 || i >= len(own.centroids) {
		return 0.5
	}
	var cum float64
	for j := 0; j < i; j++ {
		cum += own.centroids[j].weight
	}
	// 质心中点分位数
	return (cum + own.centroids[i].weight/2) / float64(own.count)
}

// maxWeight 返回给定分位数 q 下允许的最大权重
func (own *tDigestPercentile) maxWeight(q float64) float64 {
	// 标准 T-Digest 公式：4 * compression * n * q * (1 - q)
	// 加上最小权重 1 避免完全无法合并
	w := 4 * own.compression * float64(own.count) * q * (1 - q)
	if w < 1 {
		return 1
	}
	return w
}

// _tryMerge 尝试将 (value, weight) 合并到第 i 个质心，返回是否成功
func (own *tDigestPercentile) _tryMerge(i int, value float64, weight float64, q float64) bool {
	if i < 0 || i >= len(own.centroids) {
		return false
	}
	c := &own.centroids[i]
	if c.weight == 0 {
		c.mean = value
		c.weight = weight
		return true
	}
	// 检查合并后的总权重是否不超过该分位数允许的最大权重
	newWeight := c.weight + weight
	if newWeight <= own.maxWeight(q) {
		// 更新均值（加权平均）
		c.mean = (c.mean*c.weight + value*weight) / newWeight
		c.weight = newWeight
		return true
	}
	return false
}

// _compress 执行压缩，确保质心数量符合理论界限
func (own *tDigestPercentile) _compress() {
	if len(own.centroids) <= 1 {
		return
	}
	newCentroids := make([]centroid, 0, len(own.centroids))
	newCentroids = append(newCentroids, own.centroids[0])
	for i := 1; i < len(own.centroids); i++ {
		c := own.centroids[i]
		last := &newCentroids[len(newCentroids)-1]
		// 估算合并后两个质心合并后的分位数（取中点）
		q := (own._centroidQuantile(len(newCentroids)-1) + own._centroidQuantile(i)) / 2
		if last.weight+c.weight <= own.maxWeight(q) {
			// 合并
			totalWeight := last.weight + c.weight
			last.mean = (last.mean*last.weight + c.mean*c.weight) / totalWeight
			last.weight = totalWeight
		} else {
			newCentroids = append(newCentroids, c)
		}
	}
	own.centroids = newCentroids
}

// GetCount 返回数据量
func (own *tDigestPercentile) GetCount() int {
	own.lock.RLock()
	defer own.lock.RUnlock()
	return own.count
}

// Merge 合并另一个 tDigestPercentile（仅支持同压缩因子）
func (own *tDigestPercentile) Merge(other Percentiler) error {
	o, ok := other.(*tDigestPercentile)
	if !ok {
		return ErrNotMergeable
	}
	if math.Abs(own.compression-o.compression) > 1e-6 {
		return ErrCompressionMismatch
	}

	// 先复制对方质心列表，避免长时间持有对方锁
	o.lock.RLock()
	centroidsCopy := make([]centroid, len(o.centroids))
	copy(centroidsCopy, o.centroids)
	minCopy := o.min
	maxCopy := o.max
	o.lock.RUnlock()

	own.lock.Lock()
	defer own.lock.Unlock()
	for _, c := range centroidsCopy {
		own._add(c.mean, c.weight)
	}
	if minCopy < own.min {
		own.min = minCopy
	}
	if maxCopy > own.max {
		own.max = maxCopy
	}
	return nil
}

// CalculatePercentile 计算近似百分位数 p (0-100)
func (own *tDigestPercentile) CalculatePercentile(p float64) (float64, error) {
	own.lock.RLock()
	defer own.lock.RUnlock()

	if p < 0 || p > 100 {
		return 0, ErrPercentileOutOfRange
	}
	if own.count == 0 {
		return 0, ErrNoDataPoints
	}
	if p == 0 {
		return own.min, nil
	}
	if p == 100 {
		return own.max, nil
	}

	// 目标 rank (0-based)
	rank := float64(own.count) * p / 100.0

	var cumWeight float64
	for i, c := range own.centroids {
		nextCum := cumWeight + c.weight
		if rank <= nextCum {
			// 该质心内插值
			frac := (rank - cumWeight) / c.weight // 0~1 之间
			// 确定该质心覆盖的值区间
			low := own.min
			if i > 0 {
				low = (own.centroids[i-1].mean + c.mean) / 2
			}
			high := own.max
			if i < len(own.centroids)-1 {
				high = (c.mean + own.centroids[i+1].mean) / 2
			}
			// 线性插值
			return low + (high-low)*frac, nil
		}
		cumWeight = nextCum
	}
	return own.max, nil
}
