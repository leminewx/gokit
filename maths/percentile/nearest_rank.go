package percentile

import (
	"math"
	"sort"
	"sync"
)

// *********************************************************************************************************************
// 最近秩模式：
// 采用标准最近秩法（nearest-rank）：第 p 百分位对应下标 ceil(n*p/100)-1
// 结果为样本中的真实观测值，不做插值，适合延迟/耗时分位数等「必须落在样本点上」的报表
// *********************************************************************************************************************

// nearestRankPercentile 使用最近秩法计算百分位数
type nearestRankPercentile struct {
	sorted bool // 标记是否已排序，用于延迟排序优化
	data   []float64
	lock   sync.RWMutex
}

// NewNearestRank 创建一个最近秩百分位数计算器实例
func NewNearestRank() Percentiler {
	return &nearestRankPercentile{
		sorted: true, // 空数据视为已排序
		data:   make([]float64, 0),
	}
}

// Add 添加一个数值
func (own *nearestRankPercentile) Add(value float64) {
	own.lock.Lock()
	own.data = append(own.data, value)
	own.sorted = false // 新增元素后排序状态失效
	own.lock.Unlock()
}

// GetCount 返回数据量
func (own *nearestRankPercentile) GetCount() int {
	own.lock.RLock()
	count := len(own.data)
	own.lock.RUnlock()
	return count
}

// Merge 合并另一个 nearestRankPercentile 的数据
func (own *nearestRankPercentile) Merge(other Percentiler) error {
	o, ok := other.(*nearestRankPercentile)
	if !ok {
		return ErrNotMergeable
	}

	// 先复制对方数据并释放对方锁，避免 a.Merge(b) 与 b.Merge(a) 并发时相互等锁
	o.lock.RLock()
	dataCopy := make([]float64, len(o.data))
	copy(dataCopy, o.data)
	o.lock.RUnlock()

	own.lock.Lock()
	own.data = append(own.data, dataCopy...)
	own.sorted = false
	own.lock.Unlock()
	return nil
}

// CalculatePercentile 计算最近秩百分位数
func (own *nearestRankPercentile) CalculatePercentile(p float64) (float64, error) {
	// ensureSorted 会就地排序并修改 sorted 标记，必须使用写锁
	own.lock.Lock()
	defer own.lock.Unlock()

	if p < 0 || p > 100 {
		return 0, ErrPercentileOutOfRange
	}

	if len(own.data) == 0 {
		return 0, ErrNoDataPoints
	}

	own.ensureSorted()
	return CalculateNearestRank(own.data, p)
}

// ensureSorted 确保数据已排序（懒排序）
func (own *nearestRankPercentile) ensureSorted() {
	if !own.sorted {
		sort.Float64s(own.data)
		own.sorted = true
	}
}

// CalculateNearestRank 在「已排序」样本上直接计算最近秩百分位数（不拷贝、不排序）。
//
// 第 p 百分位对应下标 ceil(n*p/100)-1：p 为 0 时取最小值，p 为 100 时取最大值。
// 结果一定是样本中的某个观测值，适合「分位数必须是真实样本」的报表场景；
// 若需要与 Excel PERCENTILE.INC / SciPy 默认一致的插值结果，请用 NewExact。
//
// 样本必须由调用方保证升序；空样本返回 ErrNoDataPoints，p 越界返回
// ErrPercentileOutOfRange。
func CalculateNearestRank(sorted []float64, p float64) (float64, error) {
	if p < 0 || p > 100 {
		return 0, ErrPercentileOutOfRange
	}

	n := len(sorted)
	if n == 0 {
		return 0, ErrNoDataPoints
	}

	if p == 0 {
		return sorted[0], nil
	}

	// ceil(n*p/100)-1：p>0 时至少为 0，p=100 时恰好为 n-1
	index := int(math.Ceil(float64(n)*p/100)) - 1
	if index < 0 {
		index = 0
	} else if index >= n {
		index = n - 1
	}
	return sorted[index], nil
}
