package cache

import (
	"sync"
	"testing"
)

func TestLRUCache_Basic(t *testing.T) {
	own := NewLRUCache[string, int](3)
	if own.GetCapacity() != 3 {
		t.Fatalf("expected capacity 3, got %d", own.GetCapacity())
	}

	own.Put("a", 1)
	own.Put("b", 2)
	own.Put("c", 3)

	if v, ok := own.Get("a"); !ok || v != 1 {
		t.Fatalf("expected a=1, got %v %v", v, ok)
	}
	if own.GetLength() != 3 {
		t.Fatalf("expected len 3, got %d", own.GetLength())
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	own := NewLRUCache[string, int](2)
	own.Put("a", 1)
	own.Put("b", 2)

	// 访问 a 使其变为最近使用
	if _, ok := own.Get("a"); !ok {
		t.Fatal("expected a present")
	}

	// 插入 c，应淘汰 b（最久未使用）
	if evicted := own.Put("c", 3); !evicted {
		t.Fatal("expected Put to report eviction")
	}

	if _, ok := own.Get("b"); ok {
		t.Fatal("expected b to be evicted")
	}
	if _, ok := own.Get("a"); !ok {
		t.Fatal("expected a to remain")
	}
	if _, ok := own.Get("c"); !ok {
		t.Fatal("expected c to remain")
	}
}

func TestLRUCache_Update(t *testing.T) {
	own := NewLRUCache[string, int](2)
	own.Put("a", 1)
	own.Put("a", 99)

	if v, ok := own.Get("a"); !ok || v != 99 {
		t.Fatalf("expected updated value 99, got %v %v", v, ok)
	}
	if own.GetLength() != 1 {
		t.Fatalf("expected len 1, got %d", own.GetLength())
	}
}

func TestLRUCache_Delete(t *testing.T) {
	own := NewLRUCache[string, int](3)
	own.Put("a", 1)

	if v, ok := own.Delete("a"); !ok || v != 1 {
		t.Fatalf("expected deleted a=1, got %v %v", v, ok)
	}
	if _, ok := own.Get("a"); ok {
		t.Fatal("expected a deleted")
	}
	if _, ok := own.Delete("a"); ok {
		t.Fatal("expected second delete miss")
	}
}

func TestLRUCache_PeekAndContainKeepOrder(t *testing.T) {
	own := NewLRUCache[string, int](2)
	own.Put("a", 1)
	own.Put("b", 2) // b 为最近使用

	if v, ok := own.Peek("a"); !ok || v != 1 {
		t.Fatalf("expected peek a=1, got %v %v", v, ok)
	}
	if !own.Contain("a") {
		t.Fatal("expected contains a")
	}

	// Peek/Contain 不应刷新顺序，插入 c 仍淘汰 a
	own.Put("c", 3)
	if _, ok := own.Get("a"); ok {
		t.Fatal("expected a evicted, Peek/Contain must not refresh order")
	}
}

func TestLRUCache_Keys(t *testing.T) {
	own := NewLRUCache[string, int](3)
	own.Put("a", 1)
	own.Put("b", 2)
	own.Put("c", 3)

	keys := own.GetKeys()
	if len(keys) != 3 || keys[0] != "c" || keys[1] != "b" || keys[2] != "a" {
		t.Fatalf("expected keys [c b a], got %v", keys)
	}

	// Get 刷新顺序后，键列表随之变化
	own.Get("a")
	keys = own.GetKeys()
	if keys[0] != "a" {
		t.Fatalf("expected a first after Get, got %v", keys)
	}
}

func TestLRUCache_OnEvict(t *testing.T) {
	evicted := make(map[string]int)
	own := NewLRUCache[string, int](1, WithOnEvict[string, int](func(key string, value int) {
		evicted[key] = value
	}))

	own.Put("a", 1)
	own.Put("b", 2) // 淘汰 a
	own.Delete("b")

	if len(evicted) != 2 || evicted["a"] != 1 || evicted["b"] != 2 {
		t.Fatalf("unexpected evict events: %v", evicted)
	}

	own.Put("c", 3)
	if n := own.Clear(); n != 1 || evicted["c"] != 3 {
		t.Fatalf("expected clear to evict c, got n=%d events=%v", n, evicted)
	}
}

func TestLRUCache_EvictCallbackReentrant(t *testing.T) {
	// 回调内操作缓存本身，验证回调在锁外执行且不会死锁
	var own *LRUCache[string, int]
	own = NewLRUCache[string, int](1, WithOnEvict[string, int](func(key string, value int) {
		if key == "a" {
			own.Put("sink", value)
		}
	}))

	own.Put("a", 1)
	own.Put("b", 2) // 淘汰 a，并在回调中写入 sink

	if v, ok := own.Get("sink"); !ok || v != 1 {
		t.Fatalf("expected sink=1 written from callback, got %v %v", v, ok)
	}
}

func TestLRUCache_Concurrent(t *testing.T) {
	own := NewLRUCache[int, int](100)

	var wait sync.WaitGroup
	for worker := range 10 {
		wait.Go(func() {
			for i := range 100 {
				own.Put(worker*1000+i, i)
				own.Get(worker * 1000)
				own.Peek(worker * 1000)
				own.GetKeys()
			}
		})
	}
	wait.Wait()

	if own.GetLength() > 100 {
		t.Fatalf("len %d exceeds capacity", own.GetLength())
	}
}

// FuzzLRUCache 随机混合操作不应 panic 且保持一致性
func FuzzLRUCache(f *testing.F) {
	f.Add([]byte{1, 2, 3})
	f.Fuzz(func(t *testing.T, data []byte) {
		c := NewLRUCache[int, int](16)
		for i, b := range data {
			switch b % 3 {
			case 0:
				c.Put(i, int(b))
			case 1:
				c.Get(i)
			case 2:
				c.Delete(i)
			}
		}

		// 容量约束
		if c.GetLength() > 16 {
			t.Fatalf("len %d exceeds capacity", c.GetLength())
		}
	})
}
