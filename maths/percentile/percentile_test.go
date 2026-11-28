package percentile

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

func TestPercentile(t *testing.T) {
	// 精确模式
	exact := NewExact()
	// 近似模式，压缩因子100
	approx := NewTDigest(100)

	// 生成测试数据：正态分布 + 少量离群
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 10000; i++ {
		val := rng.NormFloat64()*10 + 50 // 均值50，标准差10
		exact.Add(val)
		approx.Add(val)
	}
	// 添加离群值
	for i := 0; i < 10; i++ {
		val := 100 + float64(i)*10
		exact.Add(val)
		approx.Add(val)
	}

	// 计算多个百分位数
	percentiles := []float64{50, 90, 95, 99, 99.9}
	for _, p := range percentiles {
		exactVal, _ := exact.CalculatePercentile(p)
		approxVal, _ := approx.CalculatePercentile(p)
		fmt.Printf("p%.1f: exact=%.4f, approx=%.4f, diff=%.4f\n",
			p, exactVal, approxVal, exactVal-approxVal)
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
	fmt.Printf("Final count: %d\n", approx.GetCount())
}
