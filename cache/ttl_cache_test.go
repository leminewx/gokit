package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

var testEpoch = time.Unix(1700000000, 0).UTC()

// newTestTTLCache 构造注入了手动时钟的 TTL 缓存
func newTestTTLCache[V any](t *testing.T, defaultTTL time.Duration, optionList ...Option[string, V]) (*TTLCache[string, V], *timex.Manual) {
	t.Helper()

	source := timex.NewManual(testEpoch)
	options := append([]Option[string, V]{WithClock[string, V](source)}, optionList...)

	return NewTTLCache[string, V](defaultTTL, options...), source
}

// waitUntil 轮询等待条件成立，超时则判定失败
func waitUntil(t *testing.T, check func() bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatal("condition not met before deadline")
}

func TestTTLCache_Basic(t *testing.T) {
	own, _ := newTestTTLCache[int](t, time.Minute)

	own.Put("x", 42, 0) // 使用默认 TTL
	if v, ok := own.Get("x"); !ok || v != 42 {
		t.Fatalf("expected x=42, got %v %v", v, ok)
	}
	if own.GetDefaultTTL() != time.Minute {
		t.Fatalf("expected default ttl 1m, got %v", own.GetDefaultTTL())
	}
}

func TestTTLCache_Expiry(t *testing.T) {
	own, source := newTestTTLCache[int](t, 0)

	own.Put("k", 1, 50*time.Millisecond)
	if _, ok := own.Get("k"); !ok {
		t.Fatal("expected k present immediately")
	}

	source.Advance(49 * time.Millisecond)
	if v, ok := own.Get("k"); !ok || v != 1 {
		t.Fatalf("expected k=1 before expiry, got %v %v", v, ok)
	}

	source.Advance(2 * time.Millisecond)
	if _, ok := own.Get("k"); ok {
		t.Fatal("expected k expired")
	}
	if own.GetLength() != 0 {
		t.Fatalf("expected len 0 after expired Get, got %d", own.GetLength())
	}
}

func TestTTLCache_NoExpiry(t *testing.T) {
	own, source := newTestTTLCache[int](t, 0)

	own.Put("p", 7, 0) // 永不过期
	source.Advance(24 * time.Hour)

	if v, ok := own.Get("p"); !ok || v != 7 {
		t.Fatalf("expected p=7 to persist, got %v %v", v, ok)
	}
	if !own.Contain("p") {
		t.Fatal("expected Contain(p) true")
	}
}

func TestTTLCache_Purge(t *testing.T) {
	events := make(map[string]int)
	var lock sync.Mutex

	own, source := newTestTTLCache[int](t, time.Minute, WithOnEvict[string, int](func(key string, value int) {
		lock.Lock()
		events[key] = value
		lock.Unlock()
	}))

	own.Put("a", 1, 10*time.Second)
	own.Put("b", 2, 10*time.Second)
	own.Put("c", 3, 0) // 默认 1 分钟，未过期

	source.Advance(20 * time.Second)

	if n := own.Purge(); n != 2 {
		t.Fatalf("expected purge 2, got %d", n)
	}
	if own.GetLength() != 1 {
		t.Fatalf("expected len 1, got %d", own.GetLength())
	}
	if n := own.Purge(); n != 0 {
		t.Fatalf("expected second purge 0, got %d", n)
	}

	lock.Lock()
	defer lock.Unlock()
	if len(events) != 2 || events["a"] != 1 || events["b"] != 2 {
		t.Fatalf("unexpected evict events: %v", events)
	}
}

func TestTTLCache_GetWithTTL(t *testing.T) {
	own, source := newTestTTLCache[int](t, 0)

	own.Put("k", 1, time.Second)
	v, remain, ok := own.GetWithTTL("k")
	if !ok || v != 1 || remain != time.Second {
		t.Fatalf("expected k=1 remain=1s, got %v %v %v", v, remain, ok)
	}

	source.Advance(200 * time.Millisecond)
	if _, remain, _ = own.GetWithTTL("k"); remain != 800*time.Millisecond {
		t.Fatalf("expected remain 800ms, got %v", remain)
	}

	// 永不过期项 remain 应为 0 但 ok=true
	own.Put("p", 9, 0)
	if _, remain, ok = own.GetWithTTL("p"); !ok || remain != 0 {
		t.Fatalf("expected permanent item ok=true remain=0, got ok=%v remain=%v", ok, remain)
	}
}

func TestTTLCache_DeleteAndClear(t *testing.T) {
	evicted := make([]string, 0)
	var lock sync.Mutex

	own, _ := newTestTTLCache[int](t, 0, WithOnEvict[string, int](func(key string, value int) {
		lock.Lock()
		evicted = append(evicted, key)
		lock.Unlock()
	}))

	own.Put("a", 1, 0)
	own.Put("b", 2, 0)

	if v, ok := own.Delete("a"); !ok || v != 1 {
		t.Fatalf("expected deleted a=1, got %v %v", v, ok)
	}
	if _, ok := own.Delete("a"); ok {
		t.Fatal("expected second delete miss")
	}

	if n := own.Clear(); n != 1 {
		t.Fatalf("expected clear 1, got %d", n)
	}
	if n := own.Clear(); n != 0 {
		t.Fatalf("expected second clear 0, got %d", n)
	}

	lock.Lock()
	defer lock.Unlock()
	if len(evicted) != 2 || evicted[0] != "a" || evicted[1] != "b" {
		t.Fatalf("unexpected evict events: %v", evicted)
	}
}

func TestTTLCache_Keys(t *testing.T) {
	own, _ := newTestTTLCache[int](t, 0)

	own.Put("a", 1, 0)
	own.Put("b", 2, 0)

	keys := own.GetKeys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %v", keys)
	}

	seen := map[string]bool{}
	for _, key := range keys {
		seen[key] = true
	}
	if !seen["a"] || !seen["b"] {
		t.Fatalf("unexpected keys: %v", keys)
	}
}

func TestTTLCache_AutoPurge(t *testing.T) {
	own, source := newTestTTLCache[int](t, time.Minute)

	own.Put("a", 1, 30*time.Second)
	own.Put("b", 2, 30*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := own.StartAutoPurge(ctx, time.Minute)

	// 元素过期，推进到首个清理周期
	source.Advance(time.Minute)
	waitUntil(t, func() bool { return own.GetLength() == 0 })

	// 停止函数可重复调用，不应 panic
	stop()
	stop()
}

func TestTTLCache_EvictCallbackReentrant(t *testing.T) {
	// 回调内操作缓存本身，验证回调在锁外执行且不会死锁
	var own *TTLCache[string, int]
	own = NewTTLCache[string, int](0, WithOnEvict[string, int](func(key string, value int) {
		own.Put("sink", value, 0)
	}))

	own.Put("a", 1, 0)
	own.Delete("a")

	if v, ok := own.Get("sink"); !ok || v != 1 {
		t.Fatalf("expected sink=1 written from callback, got %v %v", v, ok)
	}
}

func TestTTLCache_Concurrent(t *testing.T) {
	own, source := newTestTTLCache[int](t, time.Minute)

	var wait sync.WaitGroup
	for worker := range 8 {
		wait.Go(func() {
			for i := range 100 {
				key := "k" + string(rune('a'+worker))
				own.Put(key, i, time.Second)
				own.Get(key)
				own.Contain(key)
				own.GetKeys()
				if i%10 == 0 {
					own.Purge()
				}
			}
		})
	}

	source.Advance(2 * time.Second)
	wait.Wait()
	_ = own.Clear()
}

// FuzzTTLCache 随机 TTL 不应 panic
func FuzzTTLCache(f *testing.F) {
	f.Add([]byte{1, 2, 3})
	f.Fuzz(func(t *testing.T, data []byte) {
		c := NewTTLCache[int, int](time.Minute)
		for i, b := range data {
			c.Put(i, int(b), time.Duration(b)*time.Millisecond)
		}
	})
}
