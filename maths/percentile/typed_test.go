package percentile

import (
	"errors"
	"testing"
)

// TestTypedPercentilerMatchesUntyped 泛型入口与 float64 入口在相同数据上必须给出完全一致的结果。
func TestTypedPercentilerMatchesUntyped(t *testing.T) {
	t.Parallel()

	values := []int64{12, 30, 58, 91, 7, 44, 23, 68, 5, 77}
	cases := []struct {
		name       string
		newTyped   func() *TypedPercentiler[int64]
		newUntyped func() Percentiler
	}{
		{"exact", NewTypedExact[int64], NewExact},
		{
			"t-digest",
			func() *TypedPercentiler[int64] { return NewTypedTDigest[int64](100) },
			func() Percentiler { return NewTDigest(100) },
		},
		{"nearest-rank", NewTypedNearestRank[int64], NewNearestRank},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			t.Parallel()

			typed := item.newTyped()
			untyped := item.newUntyped()
			for _, value := range values {
				typed.Add(value)
				untyped.Add(float64(value))
			}

			if got, want := typed.GetCount(), untyped.GetCount(); got != want {
				t.Fatalf("GetCount = %d, want %d", got, want)
			}
			for _, p := range []float64{0, 10, 50, 95, 100} {
				got, err := typed.CalculatePercentile(p)
				if err != nil {
					t.Errorf("P%v 返回错误: %v", p, err)
					continue
				}
				want, err := untyped.CalculatePercentile(p)
				if err != nil {
					t.Fatalf("float64 版本 P%v 返回错误: %v", p, err)
				}
				if got != want {
					t.Errorf("P%v = %v, want %v", p, got, want)
				}
			}
		})
	}
}

// TestTypedPercentilerIntegerPrecision 记录并锁定 2^53 边界行为：超出精度的整数在 Add 时被舍入。
func TestTypedPercentilerIntegerPrecision(t *testing.T) {
	t.Parallel()

	const exactLimit = int64(1) << 53 // 2^53，float64 可精确表示的最大连续整数

	t.Run("int64 超过 2^53 被舍入（最近偶数）", func(t *testing.T) {
		t.Parallel()

		calculator := NewTypedNearestRank[int64]()
		calculator.Add(exactLimit)
		calculator.Add(exactLimit + 1) // 舍入为 2^53

		got, err := calculator.CalculatePercentile(100)
		if err != nil {
			t.Fatalf("P100 返回错误: %v", err)
		}
		if want := float64(exactLimit); got != want {
			t.Errorf("P100 = %v, want %v", got, want)
		}
		// 前提校验：2^53+1 在 float64 中已与 2^53 无法区分，故结果不可能还原原值
		if float64(exactLimit+1) != float64(exactLimit) {
			t.Fatalf("前提不成立：2^53+1 应被舍入为 2^53")
		}
	})

	t.Run("uint64 计数超过 2^53 被舍入", func(t *testing.T) {
		t.Parallel()

		calculator := NewTypedNearestRank[uint64]()
		calculator.Add(uint64(1)<<53 + 1)

		got, err := calculator.CalculatePercentile(100)
		if err != nil {
			t.Fatalf("P100 返回错误: %v", err)
		}
		if want := float64(uint64(1) << 53); got != want {
			t.Errorf("P100 = %v, want %v", got, want)
		}
	})

	t.Run("float32 提升为 float64 无损", func(t *testing.T) {
		t.Parallel()

		calculator := NewTypedNearestRank[float32]()
		calculator.Add(1.1)

		got, err := calculator.CalculatePercentile(100)
		if err != nil {
			t.Fatalf("P100 返回错误: %v", err)
		}
		if want := float64(float32(1.1)); got != want {
			t.Errorf("P100 = %v, want %v", got, want)
		}
	})

	t.Run("int32 无损", func(t *testing.T) {
		t.Parallel()

		calculator := NewTypedExact[int32]()
		for _, value := range []int32{1, 2, 3, 4} {
			calculator.Add(value)
		}

		got, err := calculator.CalculatePercentile(50)
		if err != nil {
			t.Fatalf("P50 返回错误: %v", err)
		}
		if want := 2.5; got != want { // R-7：两个中间样本的线性插值
			t.Errorf("P50 = %v, want %v", got, want)
		}
	})
}

// TestTypedPercentilerMerge 同算法同类型可合并；跨算法与 nil 返回 ErrNotMergeable。
func TestTypedPercentilerMerge(t *testing.T) {
	t.Parallel()

	left := NewTypedNearestRank[int64]()
	right := NewTypedNearestRank[int64]()
	left.Add(10)
	right.Add(20)
	right.Add(30)

	if err := left.Merge(right); err != nil {
		t.Fatalf("Merge 失败: %v", err)
	}
	if got, want := left.GetCount(), 3; got != want {
		t.Errorf("合并后 GetCount = %d, want %d", got, want)
	}
	got, err := left.CalculatePercentile(100)
	if err != nil {
		t.Fatalf("P100 返回错误: %v", err)
	}
	if want := 30.0; got != want {
		t.Errorf("P100 = %v, want %v", got, want)
	}

	if err := left.Merge(NewTypedExact[int64]()); !errors.Is(err, ErrNotMergeable) {
		t.Errorf("跨算法合并错误 = %v, want ErrNotMergeable", err)
	}
	if err := left.Merge(nil); !errors.Is(err, ErrNotMergeable) {
		t.Errorf("合并 nil 错误 = %v, want ErrNotMergeable", err)
	}
}

// TestTypedPercentilerInterop GetPercentiler 暴露的底层实例应与非泛型 API 双向可见。
func TestTypedPercentilerInterop(t *testing.T) {
	t.Parallel()

	typed := NewTypedExact[int64]()
	inner := typed.GetPercentiler()
	if inner == nil {
		t.Fatal("GetPercentiler 返回 nil")
	}

	inner.Add(42) // 经非泛型接口写入
	if got, want := typed.GetCount(), 1; got != want {
		t.Errorf("GetCount = %d, want %d", got, want)
	}
	got, err := typed.CalculatePercentile(100)
	if err != nil {
		t.Fatalf("P100 返回错误: %v", err)
	}
	if want := 42.0; got != want {
		t.Errorf("P100 = %v, want %v", got, want)
	}
}

// TestTypedPercentilerErrors 空数据与越界百分位必须返回可判定错误。
func TestTypedPercentilerErrors(t *testing.T) {
	t.Parallel()

	calculator := NewTypedTDigest[int64](100)
	if _, err := calculator.CalculatePercentile(50); !errors.Is(err, ErrNoDataPoints) {
		t.Errorf("空数据错误 = %v, want ErrNoDataPoints", err)
	}

	calculator.Add(1)
	if _, err := calculator.CalculatePercentile(-1); !errors.Is(err, ErrPercentileOutOfRange) {
		t.Errorf("P-1 错误 = %v, want ErrPercentileOutOfRange", err)
	}
	if _, err := calculator.CalculatePercentile(101); !errors.Is(err, ErrPercentileOutOfRange) {
		t.Errorf("P101 错误 = %v, want ErrPercentileOutOfRange", err)
	}
}

// TestTypedPercentilerConcurrentAdd 并发写入不丢样本（配合 -race 检查数据竞争）。
func TestTypedPercentilerConcurrentAdd(t *testing.T) {
	t.Parallel()

	const workers, perWorker = 8, 100
	calculator := NewTypedNearestRank[int]()

	done := make(chan struct{})
	for worker := 0; worker < workers; worker++ {
		go func(base int) {
			defer func() { done <- struct{}{} }()
			for offset := 0; offset < perWorker; offset++ {
				calculator.Add(base + offset)
			}
		}(worker * perWorker)
	}
	for worker := 0; worker < workers; worker++ {
		<-done
	}

	if got, want := calculator.GetCount(), workers*perWorker; got != want {
		t.Errorf("GetCount = %d, want %d", got, want)
	}
}
