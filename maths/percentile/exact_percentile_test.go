package percentile

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestExactPercentile(t *testing.T) {
	exact := NewExact()
	// 生成测试数据：正态分布 + 少量离群
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 10000; i++ {
		val := rng.NormFloat64()*10 + 50 // 均值50，标准差10
		exact.Add(val)
	}
	// 添加离群值
	for i := 0; i < 10; i++ {
		val := 100 + float64(i)*10
		exact.Add(val)
	}

	// 计算多个百分位数
	percentiles := []float64{50, 90, 95, 99, 99.9}
	for _, p := range percentiles {
		exactVal, _ := exact.CalculatePercentile(p)
		fmt.Printf("p%.1f: exact=%.4f\n", p, exactVal)
	}

	// 并发测试：安全添加
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
		}()
	}
	wg.Wait()
}

// cpu: Intel(R) Core(TM) i5-14600KF		20    	43633348	        27.52 ns/op	      43 B/op	       0 allocs/op
func BenchmarkExactPercentile_Add(b *testing.B) {
	exact := NewExact()
	rng := rand.New(rand.NewSource(42))

	b.ReportAllocs()

	for b.Loop() {
		exact.Add(rng.NormFloat64())
	}
}

// cpu: Intel(R) Core(TM) i5-14600KF		20    	100000000	        10.59 ns/op	       0 B/op	       0 allocs/op
func BenchmarkExactPercentile_CalculatePercentile(b *testing.B) {
	exact := NewExact()
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 10000; i++ {
		exact.Add(rng.NormFloat64())
	}

	b.ReportAllocs()

	for b.Loop() {
		if _, err := exact.CalculatePercentile(95); err != nil {
			b.Fatal(err)
		}
	}
}

func TestExactPercentile_MergeSelf(t *testing.T) {
	p := NewExact()
	p.Add(1)
	p.Add(2)

	// 自合并不能死锁，且数据应翻倍
	if err := p.Merge(p); err != nil {
		t.Fatalf("merge self failed: %v", err)
	}
	if got := p.GetCount(); got != 4 {
		t.Fatalf("expected 4 data points after self merge, got %d", got)
	}
}

func TestExactPercentile_MergeMutual(t *testing.T) {
	a, b := NewExact(), NewExact()
	a.Add(1)
	b.Add(2)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = a.Merge(b)
	}()
	go func() {
		defer wg.Done()
		_ = b.Merge(a)
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("mutual merge deadlocked")
	}
}

func TestExactPercentile_CalculateIsRaceFree(t *testing.T) {
	p := NewExact()
	for i := 0; i < 100; i++ {
		p.Add(float64(i))
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				// 首次查询会触发懒排序，并发下必须无数据竞争
				_, _ = p.CalculatePercentile(50)
			}
		}()
	}
	wg.Wait()
}
