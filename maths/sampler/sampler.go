// Package sampler 提供可注入随机源的随机采样
//   - 均匀分布：区间内等概率取值；
//   - 正态分布：Box-Muller 变换；
//   - 指数分布：逆变换法；
//   - 泊松分布：小均值用 Knuth 乘积法，大均值用正态近似。
//
// 设计目标：
//   - 零第三方依赖，只使用标准库 math/rand/v2；
//   - 随机源以 Sourcer 接口注入，同一种子可完全复现序列；
//   - 统一返回 (样本, error)，参数非法或随机源越界立即失败，不 panic；
//   - 读取随机源会推进内部状态，同一个 Sampler 不可并发调用。
//
// 已知限制：
//   - 泊松分布在均值大于 50 时改用正态近似，尾部概率精度下降；
//   - 不提供批量样本接口，需要多条样本时由调用方循环或自行并行；
//   - 采样质量取决于注入的随机源，生产环境应使用 PCG 等高质量生成器。
package sampler

import (
	"math"
	"math/rand/v2"
)

// Sourcer 均匀随机数源，返回 [0, 1) 区间的浮点数
type Sourcer interface {
	GetFloat64() float64
}

// randSource 把 math/rand/v2 的伪随机数生成器适配为 Sourcer
type randSource struct {
	random *rand.Rand
}

var _ Sourcer = (*randSource)(nil)

// GetFloat64 返回 [0, 1) 区间的伪随机数
func (own *randSource) GetFloat64() float64 {
	return own.random.Float64()
}

// Sampler 基于可注入随机源的采样器
// 读取随机源会推进内部状态，同一个实例不可并发调用。
type Sampler struct {
	source Sourcer
}

const (
	// _PCG_STREAM 为默认 PCG 生成器的第二路种子增量
	_PCG_STREAM = 0x9e3779b97f4a7c15
	// _POISSON_EXACT_LIMIT 为泊松分布切换到正态近似的均值阈值
	_POISSON_EXACT_LIMIT = 50.0
)

// NewSource 用给定伪随机数生成器构造随机源，random 为 nil 时返回 ErrNilRandom
func NewSource(random *rand.Rand) (Sourcer, error) {
	if random == nil {
		return nil, ErrNilRandom
	}

	return &randSource{random: random}, nil
}

// NewSampler 用给定随机源构造采样器，source 为 nil 时返回 ErrNilSource
func NewSampler(source Sourcer) (*Sampler, error) {
	if source == nil {
		return nil, ErrNilSource
	}

	return &Sampler{source: source}, nil
}

// NewDefaultSampler 用 PCG 伪随机数生成器构造采样器，相同 seed 得到相同序列
func NewDefaultSampler(seed uint64) *Sampler {
	return &Sampler{
		source: &randSource{random: rand.New(rand.NewPCG(seed, seed^_PCG_STREAM))},
	}
}

// SampleUniform 从 [lower, upper) 均匀分布抽取一个样本
func (own *Sampler) SampleUniform(lower, upper float64) (float64, error) {
	if !isFinite(lower) || !isFinite(upper) || lower >= upper {
		return 0, wrapError(ErrInvalidRange, "[%g, %g]", lower, upper)
	}

	value, err := own.draw()
	if err != nil {
		return 0, err
	}

	return lower + value*(upper-lower), nil
}

// ShuffleIndices 返回 0..count-1 的随机排列（Fisher-Yates 洗牌）
func (own *Sampler) ShuffleIndices(count int) ([]int, error) {
	if count <= 0 {
		return nil, wrapError(ErrInvalidCount, "%d", count)
	}

	indices := make([]int, count)
	for index := range indices {
		indices[index] = index
	}

	for index := count - 1; index > 0; index-- {
		position, err := own.calculateIndex(index + 1)
		if err != nil {
			return nil, err
		}

		indices[index], indices[position] = indices[position], indices[index]
	}

	return indices, nil
}

// draw 从随机源取一个 [0, 1) 的值并校验范围
func (own *Sampler) draw() (float64, error) {
	value := own.source.GetFloat64()
	if !isFinite(value) || value < 0 || value >= 1 {
		return 0, wrapError(ErrInvalidSource, "%g", value)
	}

	return value, nil
}

// calculateIndex 把 [0, 1) 的均匀值映射为 [0, limit) 的下标
func (own *Sampler) calculateIndex(limit int) (int, error) {
	value, err := own.draw()
	if err != nil {
		return 0, err
	}

	index := int(value * float64(limit))
	if index >= limit {
		index = limit - 1
	}

	return index, nil
}

// isFinite 判断浮点数是否为有限值
func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
