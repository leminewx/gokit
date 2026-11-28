package bloom

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestBloomFilter(t *testing.T) {
	// 为int类型创建布隆过滤器
	intBloom := NewBloomFilter(Config{
		ExpectedSize:      1000000,
		FalsePositiveRate: 0.01,
		Concurrent:        true,
	}, func(i int) []byte {
		// 将int转换为字节数组
		return fmt.Appendf(nil, "%d", i)
	})

	// 添加元素
	for i := 0; i < 1000; i++ {
		intBloom.Add(i)
	}

	// 检查元素
	fmt.Printf("Contains 500: %v\n", intBloom.Has(500))
	fmt.Printf("Contains 2000: %v\n", intBloom.Has(2000))
	fmt.Printf("Estimated count: %d\n", intBloom.EstimateCount())
	fmt.Printf("False positive rate: %.6f\n", intBloom.CalculateFalsePositiveRate())

	// 为字符串类型创建布隆过滤器
	stringBloom := NewBloomFilter(Config{
		ExpectedSize:      1000000,
		FalsePositiveRate: 0.001,
	}, func(s string) []byte {
		return []byte(s)
	})

	stringBloom.Add("hello")
	stringBloom.Add("world")
	fmt.Printf("\nContains 'hello': %v\n", stringBloom.Has("hello"))
	fmt.Printf("Contains 'test': %v\n", stringBloom.Has("test"))
}

// TestNewBloomFilter_NilToBytes 验证缺少 toBytes 时在构造期就明确 panic
//   - 回归点：曾允许传 nil，直到 Add/Has 才因空指针崩溃，现场远离出错点
func TestNewBloomFilter_NilToBytes(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("nil toBytes should panic")
		}
		if msg, ok := r.(string); !ok || !strings.Contains(msg, "toBytes") {
			t.Fatalf("unexpected panic value: %v", r)
		}
	}()

	_ = NewBloomFilter[int](Config{ExpectedSize: 100}, nil)
}

// TestBloomFilter_FalsePositiveRateEstimate 验证 FPR 估算值与理论配置一致
func TestBloomFilter_FalsePositiveRateEstimate(t *testing.T) {
	const expected = 1000
	bf := NewBloomFilter[int](Config{ExpectedSize: expected, FalsePositiveRate: 0.01}, func(i int) []byte {
		return []byte(fmt.Sprintf("%d", i))
	})

	for i := 0; i < expected; i++ {
		bf.Add(i)
	}

	// 插入量达到预期后，估算值应与配置量级接近（远低于 10%）
	if fpr := bf.CalculateFalsePositiveRate(); fpr <= 0 || fpr > 0.05 {
		t.Fatalf("FPR = %v, want (0, 0.05]", fpr)
	}

	// 未插入时 FPR 应为 0
	empty := NewBloomFilter[int](Config{ExpectedSize: expected}, func(i int) []byte { return []byte{byte(i)} })
	if fpr := empty.CalculateFalsePositiveRate(); fpr != 0 {
		t.Fatalf("empty filter FPR = %v, want 0", fpr)
	}
}

// TestBloomFilter_ConcurrentAddHas 校验并发写入后成员立即可见
func TestBloomFilter_ConcurrentAddHas(t *testing.T) {
	bf := NewBloomFilter[string](Config{
		ExpectedSize:      10000,
		FalsePositiveRate: 0.01,
		Concurrent:        true,
	}, func(item string) []byte { return []byte(item) })

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				item := fmt.Sprintf("key-%d-%d", base, j)
				bf.Add(item)
				if !bf.Has(item) {
					t.Errorf("item %q should exist after add", item)
					return
				}
			}
		}(i)
	}

	wg.Wait()
}
