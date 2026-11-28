package dict

import (
	"testing"
	"time"
)

// TestSafeDictRangeAllowsMutation 验证在 Range 回调内可以安全写字典
//   - 回归点：Range 曾持有读锁调用回调，回调内 Set 会因 RWMutex 不可重入而永久阻塞
func TestSafeDictRangeAllowsMutation(t *testing.T) {
	dict := NewSafeDict[int, string]()
	dict.Set(1, "one")
	dict.Set(2, "two")

	done := make(chan struct{})
	go func() {
		defer close(done)
		dict.Range(func(key int, _ string) {
			dict.Set(key+100, "moved")
			dict.Delete(key)
		})
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Range 回调内写字典导致死锁")
	}

	if dict.GetLength() != 2 {
		t.Fatalf("长度 = %d, want 2", dict.GetLength())
	}
	for _, key := range []int{101, 102} {
		if _, ok := dict.Get(key); !ok {
			t.Fatalf("缺少迁移后的键 %d", key)
		}
	}
}

func TestSafeDict(t *testing.T) {
	dict := NewSafeDict[int, string]()
	data := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	for i, v := range data {
		dict.Set(i, v)
	}

	t.Log(dict.Get(7))

	t.Log(dict.GetKeys())

	t.Log(dict.GetValues())

	t.Log(dict.GetLength())

	t.Log(dict.GetMap())

	dict.Update(map[int]string{123: "123", 234: "234"})
}
