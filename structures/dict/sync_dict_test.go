package dict

import (
	"sync"
	"testing"
)

func TestSyncDictTransformOnce(t *testing.T) {
	t.Parallel()

	instance := NewSyncDict[int, int]()
	instance.Update(map[int]int{1: 1, 2: 2, 3: 3})

	calls := 0
	instance.Transform(func(_ int, val int) int {
		calls++
		return val * 10
	})

	// 每个键的回调必须只执行一次
	if calls != 3 {
		t.Errorf("Transform 回调次数 = %d, want 3", calls)
	}
	for key, want := range map[int]int{1: 10, 2: 20, 3: 30} {
		if val, _ := instance.Get(key); val != want {
			t.Errorf("Transform 后 Get(%d) = %d, want %d", key, val, want)
		}
	}
}

func TestSyncDictZeroValue(t *testing.T) {
	t.Parallel()

	var instance SyncDict[string, int]
	instance.Set("a", 1)
	if val, ok := instance.Get("a"); !ok || val != 1 {
		t.Errorf("零值 Get(\"a\") = (%d, %v), want (1, true)", val, ok)
	}

	instance.Clear()
	if got := instance.GetLength(); got != 0 {
		t.Errorf("Clear 后长度 = %d, want 0", got)
	}
}

func TestSyncDictConcurrent(t *testing.T) {
	t.Parallel()

	const goroutines = 16
	const iterations = 200

	instance := NewSyncDict[int, int]()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := id*iterations + j
				instance.Set(key, key*2)

				if val, ok := instance.Get(key); !ok || val != key*2 {
					t.Errorf("并发场景 Get(%d) = (%d, %v)", key, val, ok)
					return
				}

				instance.HasAll(key)
				instance.GetLength()
			}
		}(i)
	}

	wg.Wait()

	if got := instance.GetLength(); got != goroutines*iterations {
		t.Errorf("并发写入后长度 = %d, want %d", got, goroutines*iterations)
	}
}
