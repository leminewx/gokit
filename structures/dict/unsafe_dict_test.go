package dict

import (
	"sort"
	"testing"
)

func TestUnsafeDict(t *testing.T) {
	dict := NewUnsafeDict[int, string]()
	for i := 0; i < 5; i++ {
		dict.Set(i, string(rune('a'+i)))
	}

	if got := dict.GetLength(); got != 5 {
		t.Fatalf("GetLength() = %d, want 5", got)
	}
	if val, ok := dict.Get(2); !ok || val != "c" {
		t.Errorf("Get(2) = (%q, %v), want (\"c\", true)", val, ok)
	}
	if _, ok := dict.Get(9); ok {
		t.Error("Get(9) 返回了不存在的键")
	}

	if !dict.HasAll(0, 1, 2) {
		t.Error("HasAll(0,1,2) = false, want true")
	}
	if dict.HasAll(0, 9) {
		t.Error("HasAll(0,9) = true, want false")
	}
	if !dict.HasAny(9, 4) {
		t.Error("HasAny(9,4) = false, want true")
	}

	dict.Update(map[int]string{10: "k", 11: "l"})
	if got := dict.GetLength(); got != 7 {
		t.Errorf("Update 后长度 = %d, want 7", got)
	}

	dict.Delete(10, 11)
	if got := dict.GetLength(); got != 5 {
		t.Errorf("Delete 后长度 = %d, want 5", got)
	}
}

func TestUnsafeDictZeroValue(t *testing.T) {
	var dict UnsafeDict[string, int]

	// 零值未初始化时必须可以安全写入
	dict.Set("a", 1)

	dict.Update(map[string]int{"b": 2})
	if got := dict.GetLength(); got != 2 {
		t.Fatalf("零值 Set/Update 后长度 = %d, want 2", got)
	}
	if val, ok := dict.Get("b"); !ok || val != 2 {
		t.Errorf("Get(\"b\") = (%d, %v), want (2, true)", val, ok)
	}

	// 空 map 更新不应破坏已有数据
	dict.Update(nil)
	if got := dict.GetLength(); got != 2 {
		t.Errorf("Update(nil) 后长度 = %d, want 2", got)
	}
}

func TestUnsafeDictZeroValueReadOnly(t *testing.T) {
	var dict UnsafeDict[string, int]

	if got := dict.GetLength(); got != 0 {
		t.Errorf("零值 GetLength() = %d, want 0", got)
	}
	if _, ok := dict.Get("missing"); ok {
		t.Error("零值 Get 返回了不存在的键")
	}
	if got := dict.GetKeys(); len(got) != 0 {
		t.Errorf("零值 GetKeys() = %v, want 空", got)
	}
	if got := dict.GetMap(); len(got) != 0 {
		t.Errorf("零值 GetMap() = %v, want 空", got)
	}
	if got := dict.Filter(func(string, int) bool { return true }); len(got) != 0 {
		t.Errorf("零值 Filter() = %v, want 空", got)
	}

	dict.Clear()
	dict.Delete("missing")
}

func TestUnsafeDictCollection(t *testing.T) {
	dict := NewUnsafeDict[int, int]()
	dict.Update(map[int]int{1: 10, 2: 20, 3: 30})

	keys := dict.GetKeys()
	sort.Ints(keys)
	if len(keys) != 3 || keys[0] != 1 || keys[2] != 3 {
		t.Errorf("GetKeys() = %v, want [1 2 3]", keys)
	}

	sum := 0
	dict.Range(func(_ int, val int) {
		sum += val
	})
	if sum != 60 {
		t.Errorf("Range 累加 = %d, want 60", sum)
	}

	dict.Transform(func(_ int, val int) int { return val * 2 })
	if val, _ := dict.Get(1); val != 20 {
		t.Errorf("Transform 后 Get(1) = %d, want 20", val)
	}

	removed := dict.Filter(func(key int, _ int) bool { return key%2 == 0 })
	if len(removed) != 1 || removed[2] != 40 {
		t.Errorf("Filter 返回 = %v, want map[2:40]", removed)
	}
	if got := dict.GetLength(); got != 2 {
		t.Errorf("Filter 后长度 = %d, want 2", got)
	}

	dict.Clear()
	if got := dict.GetLength(); got != 0 {
		t.Errorf("Clear 后长度 = %d, want 0", got)
	}
}
