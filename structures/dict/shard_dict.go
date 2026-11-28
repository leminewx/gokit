package dict

import (
	"hash/maphash"
	"iter"
	"math/bits"
	"runtime"
)

var _ Dict[int, int] = (*ShardSafeDict[int, int])(nil)

const (
	// _SHARDS_PER_CPU 每个 CPU 核分配的分片数
	_SHARDS_PER_CPU = 32
	// _MAX_SHARD_COUNT 分片数量上限，避免大规格机器创建过多分片
	_MAX_SHARD_COUNT = 1024
)

// ShardSafeDict 定义分片且并发安全的字典类型
//   - 必须通过 NewShardSafeDict 创建，零值不可用
//   - 分片数量为 2 的幂，用位与替代取模
type ShardSafeDict[K comparable, V any] struct {
	seed   maphash.Seed
	mask   uint64
	shards []*SafeDict[K, V]
}

// NewShardSafeDict 创建一个分片的并发安全字典
//   - 分片数量：runtime.NumCPU() * 32，向上取整到 2 的幂且不超过 1024
func NewShardSafeDict[K comparable, V any]() *ShardSafeDict[K, V] {
	count := min(runtime.NumCPU()*_SHARDS_PER_CPU, _MAX_SHARD_COUNT)
	return NewShardSafeDictWithCount[K, V](count)
}

// NewShardSafeDictWithCount 按指定分片数创建分片字典
//   - count 会向上取整到 2 的幂，最小为 1
func NewShardSafeDictWithCount[K comparable, V any](count int) *ShardSafeDict[K, V] {
	count = roundUpPowerOfTwo(count)
	shards := make([]*SafeDict[K, V], count)
	for i := range shards {
		shards[i] = NewSafeDict[K, V]()
	}

	return &ShardSafeDict[K, V]{
		seed:   maphash.MakeSeed(),
		mask:   uint64(count - 1),
		shards: shards,
	}
}

// roundUpPowerOfTwo 向上取整到 2 的幂，保证可以用掩码替代取模
func roundUpPowerOfTwo(count int) int {
	if count <= 1 {
		return 1
	}

	return 1 << bits.Len(uint(count-1))
}

// Set 设置键值对
func (own *ShardSafeDict[K, V]) Set(key K, val V) {
	own.getShard(key).Set(key, val)
}

// Update 更新字典
func (own *ShardSafeDict[K, V]) Update(items map[K]V) {
	for key, val := range items {
		own.getShard(key).Set(key, val)
	}
}

// Delete 删除键值对
func (own *ShardSafeDict[K, V]) Delete(keys ...K) {
	for _, key := range keys {
		own.getShard(key).Delete(key)
	}
}

// HasAll 判断所有指定键值对是否存在
func (own *ShardSafeDict[K, V]) HasAll(keys ...K) bool {
	for _, key := range keys {
		if _, ok := own.getShard(key).Get(key); !ok {
			return false
		}
	}

	return true
}

// HasAny 判断任一指定元素是存在
func (own *ShardSafeDict[K, V]) HasAny(keys ...K) bool {
	for _, key := range keys {
		if _, ok := own.getShard(key).Get(key); ok {
			return true
		}
	}

	return false
}

// Get 查询键值对
func (own *ShardSafeDict[K, V]) Get(key K) (V, bool) {
	return own.getShard(key).Get(key)
}

// GetLength 获取字典长度
func (own *ShardSafeDict[K, V]) GetLength() (length int) {
	for _, dict := range own.shards {
		length += dict.GetLength()
	}

	return
}

// GetKeys 获取字典中所有的键
func (own *ShardSafeDict[K, V]) GetKeys() []K {
	keys := make([]K, 0, own.GetLength())
	for _, dict := range own.shards {
		keys = append(keys, dict.GetKeys()...)
	}

	return keys
}

// GetValues 获取字典中所有的值
func (own *ShardSafeDict[K, V]) GetValues() []V {
	values := make([]V, 0, own.GetLength())
	for _, dict := range own.shards {
		values = append(values, dict.GetValues()...)
	}

	return values
}

// GetMap 获取Map
func (own *ShardSafeDict[K, V]) GetMap() map[K]V {
	items := make(map[K]V, own.GetLength())
	for _, dict := range own.shards {
		for key, val := range dict.GetMap() {
			items[key] = val
		}
	}

	return items
}

// GetItems 返回遍历字典的迭代器，可在 for-range 中直接使用
//   - 逐个分片遍历，遍历期间不加全局锁，结果弱一致
func (own *ShardSafeDict[K, V]) GetItems() iter.Seq2[K, V] {
	return func(yield func(key K, val V) bool) {
		for _, dict := range own.shards {
			for key, val := range dict.GetItems() {
				if !yield(key, val) {
					return
				}
			}
		}
	}
}

// Range 遍历字典
func (own *ShardSafeDict[K, V]) Range(fn func(key K, val V)) {
	for _, dict := range own.shards {
		dict.Range(fn)
	}
}

// Filter 从字典中过滤出符合要求的键值对
func (own *ShardSafeDict[K, V]) Filter(fn func(key K, val V) bool) map[K]V {
	items := make(map[K]V)
	for _, dict := range own.shards {
		for key, val := range dict.Filter(fn) {
			items[key] = val
		}
	}

	return items
}

// Transform 遍历并更新每个元素
func (own *ShardSafeDict[K, V]) Transform(fn func(key K, val V) V) {
	for _, dict := range own.shards {
		dict.Transform(fn)
	}
}

// Clear 清空字典
func (own *ShardSafeDict[K, V]) Clear() {
	for _, dict := range own.shards {
		dict.Clear()
	}
}

// getShard 根据键定位分片
//   - maphash.Comparable 覆盖所有可比较类型（含结构体、数组、布尔值），避免自行实现哈希导致分布倾斜
func (own *ShardSafeDict[K, V]) getShard(key K) *SafeDict[K, V] {
	return own.shards[maphash.Comparable(own.seed, key)&own.mask]
}
