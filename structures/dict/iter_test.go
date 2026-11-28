package dict

import (
	"sort"
	"strconv"
	"testing"
)

// TestDictIterHelpers 验证 ExtractKeys/ExtractValues/Collect 三个迭代器辅助函数
func TestDictIterHelpers(t *testing.T) {
	instance := NewSyncDict[string, int]()
	instance.Update(map[string]int{"a": 1, "b": 2, "c": 3})

	keys := make([]string, 0, 3)
	for key := range ExtractKeys(instance.GetItems()) {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) != 3 || keys[0] != "a" || keys[2] != "c" {
		t.Errorf("ExtractKeys() = %v, want [a b c]", keys)
	}

	values := 0
	for val := range ExtractValues(instance.GetItems()) {
		values += val
	}
	if values != 6 {
		t.Errorf("ExtractValues() 之和 = %d, want 6", values)
	}

	collected := Collect(instance.GetItems())
	if got := collected.GetLength(); got != 3 {
		t.Errorf("Collect() 长度 = %d, want 3", got)
	}
	if val, ok := collected.Get("b"); !ok || val != 2 {
		t.Errorf("Collect() 后 Get(b) = (%d, %v), want (2, true)", val, ok)
	}
}

// distinctKey 用于验证结构体等可比较类型的键能被正确哈希
type distinctKey struct {
	Region int
	ID     string
}

// TestShardSafeDictComparableKey 验证结构体键的分片哈希与查找
func TestShardSafeDictComparableKey(t *testing.T) {
	instance := NewShardSafeDict[distinctKey, int]()
	const total = 1000
	for i := 0; i < total; i++ {
		instance.Set(distinctKey{Region: i % 8, ID: strconv.Itoa(i)}, i)
	}

	if got := instance.GetLength(); got != total {
		t.Fatalf("GetLength() = %d, want %d", got, total)
	}
	if val, ok := instance.Get(distinctKey{Region: 4, ID: "500"}); !ok || val != 500 {
		t.Errorf("Get(结构体键) = (%d, %v), want (500, true)", val, ok)
	}
	if !instance.HasAll(distinctKey{Region: 0, ID: "0"}, distinctKey{Region: 7, ID: "999"}) {
		t.Error("HasAll(存在的结构体键) = false, want true")
	}
	if instance.HasAll(distinctKey{Region: 0, ID: "0"}, distinctKey{ID: "缺失"}) {
		t.Error("HasAll(含不存在的键) = true, want false")
	}

	// 结构体键应分散到多个分片，而不是全部落在 0 号分片
	used := 0
	for _, shard := range instance.shards {
		if shard.GetLength() > 0 {
			used++
		}
	}
	if used < 2 {
		t.Errorf("结构体键仅落在 %d 个分片, want >= 2", used)
	}
}

// TestNewShardSafeDictWithCount 验证分片数向上取整与掩码计算
func TestNewShardSafeDictWithCount(t *testing.T) {
	instance := NewShardSafeDictWithCount[int, int](3)
	if got := len(instance.shards); got != 4 {
		t.Fatalf("分片数 = %d, want 4", got)
	}
	if instance.mask != 3 {
		t.Fatalf("掩码 = %d, want 3", instance.mask)
	}

	for i := 0; i < 64; i++ {
		instance.Set(i, i*i)
	}
	if val, ok := instance.Get(63); !ok || val != 3969 {
		t.Errorf("Get(63) = (%d, %v), want (3969, true)", val, ok)
	}
	if got := instance.GetLength(); got != 64 {
		t.Errorf("GetLength() = %d, want 64", got)
	}

	instance.Clear()
	if got := instance.GetLength(); got != 0 {
		t.Errorf("Clear() 后长度 = %d, want 0", got)
	}
}
