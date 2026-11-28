package knn

import (
	"container/heap"
	"math"
	"sync"

	"gitee.com/leminewx/gokit"
	"gitee.com/leminewx/gokit/maths/learning"
)

type (
	// SimpleModel 简单的 KNN 模型
	SimpleModel[T gokit.Number] struct {
		// 模型参数
		k       uint  // 邻居个数
		dim     uint  // 特征维度（用于校验）
		labels  []int // 训练标签（整数类别）
		samples [][]T // 训练样本及其特征

		// 距离计算器
		distanceCalculator learning.DistanceCalculator[T]

		lock sync.RWMutex
	}

	// neighbor 使用最大堆来筛选最小 k 个值（性能优化，避免全排序）
	neighbor struct {
		index    int // 原始顺序，用于稳定平票
		label    int
		distance float64
	}
)

// NewSimpleModel 创建一个简单的 KNN 模型
func NewSimpleModel[T gokit.Number](k uint) *SimpleModel[T] {
	return &SimpleModel[T]{
		k:                  k,
		distanceCalculator: learning.CalculateEuclideanDistance[T],
	}
}

// SetDistanceCalculator 设置距离计算器
func (own *SimpleModel[T]) SetDistanceCalculator(calculator learning.DistanceCalculator[T]) {
	own.lock.Lock()
	own.distanceCalculator = calculator
	own.lock.Unlock()
}

// IsTrained 返回模型是否已训练（有数据）
func (own *SimpleModel[T]) IsTrained() bool {
	own.lock.RLock()
	trained := len(own.labels) > 0
	own.lock.RUnlock()
	return trained
}

// Train 训练模型
func (own *SimpleModel[T]) Train(samples [][]T, labels []int) error {
	own.lock.Lock()
	defer own.lock.Unlock()

	if len(samples) == 0 {
		return ErrNoSamples
	}
	if len(samples) != len(labels) {
		return ErrLabelMismatch
	}
	if len(samples[0]) == 0 {
		return ErrNoFeatures
	}

	// 检查特征维度一致性
	dim := len(samples[0])
	for i, feat := range samples {
		if len(feat) != dim {
			return wrapDimensionError("inconsistent feature dimensions: need %d, has %d, sample index: %d", dim, len(feat), i)
		}
	}
	own.dim = uint(dim)

	// 深拷贝样本和标签
	own.samples = make([][]T, len(samples))
	for i, features := range samples {
		own.samples[i] = append([]T(nil), features...)
	}

	own.labels = append([]int(nil), labels...)
	return nil
}

// Predict 预测单个样本的类别
//   - 若没有训练数据，则返回 math.MinInt
func (own *SimpleModel[T]) Predict(object []T) (int, error) {
	own.lock.RLock()
	defer own.lock.RUnlock()

	if len(own.labels) == 0 {
		return math.MinInt, ErrNotTrained
	}

	if own.k == 0 {
		return math.MinInt, ErrInvalidK
	}

	if len(object) != int(own.dim) {
		return 0, wrapDimensionError("input feature dimension %d, model requires %d", len(object), own.dim)
	}

	// 最大堆，用于保留最小的 k 个距离（通过负距离实现）
	hp := &maxHeap{k: int(own.k), items: []neighbor{}}
	heap.Init(hp)

	// 遍历所有样本
	for i, sample := range own.samples {
		dist := own.distanceCalculator(object, sample)
		nei := neighbor{index: i, label: own.labels[i], distance: dist}
		if hp.Len() < int(own.k) {
			heap.Push(hp, nei)
		} else if dist < hp.Peek().distance {
			heap.Pop(hp)
			heap.Push(hp, nei)
		}
	}

	// 从堆中取出所有邻居（距离从小到大）
	neighbors := make([]neighbor, hp.Len())
	for i := hp.Len() - 1; i >= 0; i-- {
		neighbors[i] = heap.Pop(hp).(neighbor)
	}

	// 投票，记录每个标签获得的票数及最小距离和最早出现索引
	type voteInfo struct {
		count      int
		minDist    float64
		firstIndex int // 该标签在 neighbors 中最早出现的位置（距离最近的那个）
	}

	vote := make(map[int]*voteInfo)
	for pos, nei := range neighbors {
		info, exists := vote[nei.label]
		if !exists {
			vote[nei.label] = &voteInfo{
				count:      1,
				minDist:    nei.distance,
				firstIndex: pos,
			}
		} else {
			info.count++
			if nei.distance < info.minDist {
				info.minDist = nei.distance
				info.firstIndex = pos
			}
		}
	}

	// 选择票数最多的标签，平票时选择 firstIndex 更小（即距离更近）的标签
	bestLabel := -1
	bestCount := -1
	bestFirstIdx := math.MaxInt
	for label, info := range vote {
		if info.count > bestCount || (info.count == bestCount && info.firstIndex < bestFirstIdx) {
			bestLabel = label
			bestCount = info.count
			bestFirstIdx = info.firstIndex
		}
	}

	return bestLabel, nil
}
