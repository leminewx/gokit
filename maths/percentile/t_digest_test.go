package percentile

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

func TestTDigestPercentile(t *testing.T) {
	// 近似模式，压缩因子100
	approx := NewTDigest(100)

	// 生成测试数据：正态分布 + 少量离群
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 10000; i++ {
		val := rng.NormFloat64()*10 + 50 // 均值50，标准差10
		approx.Add(val)
	}
	// 添加离群值
	for i := 0; i < 10; i++ {
		val := 100 + float64(i)*10
		approx.Add(val)
	}

	// 计算多个百分位数
	percentiles := []float64{50, 90, 95, 99, 99.9}
	for _, p := range percentiles {
		approxVal, _ := approx.CalculatePercentile(p)
		fmt.Printf("p%.1f:  approx=%.4f\n", p, approxVal)
	}

	// 并发测试：安全添加
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			approx.Add(rand.Float64() * 100)
		}()
	}

	wg.Wait()
}

// cpu: Intel(R) Core(TM) i5-14600KF		20    	35011144	        35.57 ns/op	       0 B/op	       0 allocs/op
func BenchmarkTDigestPercentile_Add(b *testing.B) {
	exact := NewTDigest(150)
	rng := rand.New(rand.NewSource(42))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exact.Add(rng.NormFloat64())
	}
}

// cpu: Intel(R) Core(TM) i5-14600KF		20    	83224101	        14.08 ns/op	       0 B/op	       0 allocs/op
func BenchmarkTDigestPercentile_CalculatePercentile(b *testing.B) {
	exact := NewTDigest(150)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 10000; i++ {
		exact.Add(rng.NormFloat64())
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := exact.CalculatePercentile(95); err != nil {
			b.Fatal(err)
		}
	}
}
