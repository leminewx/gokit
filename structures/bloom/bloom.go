package bloom

import (
	"hash"
	"hash/fnv"
	"math"
	"sync"
	"sync/atomic"
)

// BloomFilter 定义布隆过滤器的接口类型
type BloomFilter[T any] interface {
	// Add 添加元素到布隆过滤器
	Add(item T)
	// Has 检查元素是否可能在布隆过滤器中
	Has(item T) bool
	// Clear 清空布隆过滤器
	Clear()
	// EstimateCount 返回估计的元素数量
	EstimateCount() uint64
	// CalculateFalsePositiveRate 计算当前假阳性概率
	CalculateFalsePositiveRate() float64
}

// HashFunc 默认哈希函数创建器
type HashFunc func() hash.Hash64

// Config 布隆过滤器配置
type Config struct {
	// ExpectedSize 预期插入的元素数量
	ExpectedSize uint64
	// FalsePositiveRate 期望的假阳性率
	FalsePositiveRate float64
	// HashFunctions 哈希函数列表，如果为空则使用默认的 FNV 哈希
	HashFunctions []HashFunc
	// Concurrent 是否启用并发安全
	Concurrent bool
}

// bloomFilter 布隆过滤器实现
type bloomFilter[T any] struct {
	bits        []atomic.Uint64 // 使用atomic支持并发读取
	m           uint64          // 位数组大小
	k           uint64          // 哈希函数数量
	count       atomic.Uint64   // 估计的元素数量
	hashersPool sync.Pool       // 按调用复用哈希实例，避免并发共享同一实例
	toBytes     func(T) []byte
	concurrent  bool

	mutex sync.RWMutex
}

// 编译期断言：内置布隆过滤器实现 BloomFilter 接口
var _ BloomFilter[int] = (*bloomFilter[int])(nil)

// NewBloomFilter 创建新的布隆过滤器
//   - ExpectedSize 为 0 时取 1000000
//   - FalsePositiveRate 必须位于 (0, 1) 区间，否则回退为 0.01
//   - HashFunctions 数量少于最优哈希个数时按循环方式补齐，避免部分位永远不被使用
//   - toBytes 为 nil 时直接 panic：缺少元素序列化方式时过滤器无法工作，
//     默默在 Add/Has 时才崩溃会让问题远离出错点，因此构造期就明确报错
func NewBloomFilter[T any](cfg Config, toBytes func(T) []byte) BloomFilter[T] {
	if toBytes == nil {
		panic("bloom: toBytes must not be nil")
	}

	if cfg.ExpectedSize == 0 {
		cfg.ExpectedSize = 1000000
	}
	if cfg.FalsePositiveRate <= 0 || cfg.FalsePositiveRate >= 1 || math.IsNaN(cfg.FalsePositiveRate) {
		cfg.FalsePositiveRate = 0.01
	}

	// 计算最优参数
	m, k := optimalParameters(cfg.ExpectedSize, cfg.FalsePositiveRate)

	// 初始化哈希函数
	hashFunctions := cfg.HashFunctions
	if len(hashFunctions) == 0 {
		hashFunctions = generateDefaultHashFunctions(k)
	}

	bf := &bloomFilter[T]{
		bits:       make([]atomic.Uint64, (m+63)/64),
		m:          m,
		k:          k,
		toBytes:    toBytes,
		concurrent: cfg.Concurrent,
	}

	// 每个调用方从池中取一组独立哈希实例，避免共享 hasher 在并发下被 Reset/Write
	bf.hashersPool.New = func() any {
		// 哈希实例数量必须与 k 一致：调用方传入的函数不足时循环复用
		hashers := make([]hash.Hash64, int(k))
		for i := range hashers {
			hashers[i] = hashFunctions[i%len(hashFunctions)]()
		}
		// 存指针，避免 Put 时把切片装箱到接口产生额外分配
		return &hashers
	}

	return bf
}

// 计算最优参数
func optimalParameters(n uint64, p float64) (m, k uint64) {
	// m = -n * ln(p) / (ln(2)^2)
	m = uint64(math.Ceil(-float64(n) * math.Log(p) / (math.Ln2 * math.Ln2)))

	// k = m / n * ln(2)
	k = uint64(math.Ceil(float64(m) / float64(n) * math.Ln2))

	// 确保至少有一个哈希函数
	if k < 1 {
		k = 1
	}

	// 确保至少有64位
	if m < 64 {
		m = 64
	}

	return m, k
}

// 默认哈希函数
func generateDefaultHashFunctions(k uint64) []HashFunc {
	functions := make([]HashFunc, k)
	for i := range functions {
		functions[i] = func() hash.Hash64 { return fnv.New64a() }
	}

	return functions
}

// Add 添加元素
func (own *bloomFilter[T]) Add(item T) {
	positions := own.hashPositions(item)

	if own.concurrent {
		own.mutex.Lock()
		defer own.mutex.Unlock()
	}

	added := false
	for _, pos := range positions {
		idx := pos / 64
		bit := pos % 64

		// 使用原子操作设置位
		oldValue := own.bits[idx].Load()
		for {
			newValue := oldValue | (1 << bit)
			if own.bits[idx].CompareAndSwap(oldValue, newValue) {
				if (oldValue>>bit)&1 == 0 {
					added = true
				}
				break
			}
			oldValue = own.bits[idx].Load()
		}
	}

	if added {
		own.count.Add(1)
	}
}

// Has 检查元素是否存在
func (own *bloomFilter[T]) Has(item T) bool {
	positions := own.hashPositions(item)

	if own.concurrent {
		own.mutex.RLock()
		defer own.mutex.RUnlock()
	}

	for _, pos := range positions {
		idx := pos / 64
		bit := pos % 64

		// 使用原子操作读取位
		bits := own.bits[idx].Load()
		if (bits>>bit)&1 == 0 {
			return false
		}
	}

	return true
}

// hashItem 计算元素的所有哈希位置
func (own *bloomFilter[T]) hashPositions(item T) []uint64 {
	data := own.toBytes(item)
	positions := make([]uint64, own.k)

	// 从池中获取独立的哈希实例，保证同一实例不会被多个 goroutine 同时 Reset/Write
	hashers := own.hashersPool.Get().(*[]hash.Hash64)
	defer own.hashersPool.Put(hashers)

	for i, h := range *hashers {
		h.Reset()
		_, _ = h.Write(data)

		// 使用双重哈希技术减少哈希计算
		hashValue := h.Sum64()
		if i > 0 {
			// 对于后续哈希，使用前一个哈希值进行混合
			hashValue = hashValue ^ positions[i-1]
		}

		positions[i] = hashValue % own.m
	}

	return positions
}

// Clear 清空布隆过滤器
func (own *bloomFilter[T]) Clear() {
	if own.concurrent {
		own.mutex.Lock()
		defer own.mutex.Unlock()
	}

	for i := range own.bits {
		own.bits[i].Store(0)
	}

	own.count.Store(0)
}

// EstimateCount 估计的元素数量
func (own *bloomFilter[T]) EstimateCount() uint64 {
	return own.count.Load()
}

// CalculateFalsePositiveRate 估算当前假阳性概率
//   - 理论公式 p = (1 - e^(-k*n/m))^k，其中 n 为已插入元素个数
//   - 属于“基于计数”的近似值：n 只统计实际上改变了位的元素，
//     且不包含哈希函数相关性/缓存局部性等真实影响因素，仅供监控参考
func (own *bloomFilter[T]) CalculateFalsePositiveRate() float64 {
	n := float64(own.count.Load())
	m := float64(own.m)
	k := float64(own.k)

	// p = (1 - e^(-k*n/m))^k
	return math.Pow(1-math.Exp(-k*n/m), k)
}

// GetSize 返回位数组大小
func (own *bloomFilter[T]) GetSize() uint64 {
	return own.m
}

// GetHashCount 返回哈希函数数量
func (own *bloomFilter[T]) GetHashCount() uint64 {
	return own.k
}

// GetMemoryUsage 返回内存使用量（字节）
func (own *bloomFilter[T]) GetMemoryUsage() uint64 {
	return uint64(len(own.bits)) * 8 // 每个uint64是8字节
}
