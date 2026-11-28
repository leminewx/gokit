package percentile

import (
	"math"
	"sort"
	"sync"
)

// *********************************************************************************************************************
// 精确模式：
// 采用排序+线性插值（R-7方法），计算结果与 Excel PERCENTILE.INC 一致
// 添加 O(log n) ，查询 O(1)
// *********************************************************************************************************************

// exactPercentile 使用精确排序算法计算百分位数（适合数据量不太大的场景）
type exactPercentile struct {
	sorted bool // 标记是否已排序，用于延迟排序优化
	data   []float64
	lock   sync.RWMutex
}

// NewExact 创建一个精确百分位数计算器实例
func NewExact() Percentiler {
	return &exactPercentile{
		sorted: true, // 空数据视为已排序
		data:   make([]float64, 0),
	}
}

// Add 添加一个数值
func (own *exactPercentile) Add(value float64) {
	own.lock.Lock()
	own.data = append(own.data, value)
	own.sorted = false // 新增元素后排序状态失效
	own.lock.Unlock()
}

// GetCount 返回数据量
func (own *exactPercentile) GetCount() int {
	own.lock.RLock()
	count := len(own.data)
	own.lock.RUnlock()
	return count
}

// Merge 合并另一个 exactPercentile 的数据
func (own *exactPercentile) Merge(other Percentiler) error {
	o, ok := other.(*exactPercentile)
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

// CalculatePercentile 计算精确百分位数（使用线性插值法 R-7）
func (own *exactPercentile) CalculatePercentile(p float64) (float64, error) {
	// ensureSorted 会就地排序并修改 sorted 标记，必须使用写锁
	own.lock.Lock()
	defer own.lock.Unlock()

	if p < 0 || p > 100 {
		return 0, ErrPercentileOutOfRange
	}

	n := len(own.data)
	if n == 0 {
		return 0, ErrNoDataPoints
	}

	own.ensureSorted()

	// 使用 R-7 方法（Excel, SciPy 默认）
	// h = (N-1)*p/100 + 1
	h := float64(n-1)*p/100 + 1
	idx := int(math.Floor(h)) - 1 // 转为 0-based 索引
	frac := h - float64(idx+1)
	if idx < 0 {
		return own.data[0], nil
	} else if idx >= n-1 {
		return own.data[n-1], nil
	}

	// 线性插值
	return own.data[idx]*(1-frac) + own.data[idx+1]*frac, nil
}

// ensureSorted 确保数据已排序（懒排序）
func (own *exactPercentile) ensureSorted() {
	if !own.sorted {
		sort.Float64s(own.data)
		own.sorted = true
	}
}
