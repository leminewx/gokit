package dict

import (
	"sort"
	"testing"
)

// dictFactory 生成不同实现的字典，用于统一契约测试
type dictFactory func() Dict[int, string]

func TestDictImplementations(t *testing.T) {
	factories := map[string]dictFactory{
		"UnsafeDict":    func() Dict[int, string] { return NewUnsafeDict[int, string]() },
		"SafeDict":      func() Dict[int, string] { return NewSafeDict[int, string]() },
		"SyncDict":      func() Dict[int, string] { return NewSyncDict[int, string]() },
		"ShardSafeDict": func() Dict[int, string] { return NewShardSafeDict[int, string]() },
	}

	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			testDictContract(t, factory())
		})
	}
}

func testDictContract(t *testing.T, instance Dict[int, string]) {
	t.Helper()

	if got := instance.GetLength(); got != 0 {
		t.Fatalf("初始长度 = %d, want 0", got)
	}

	instance.Set(1, "one")
	instance.Set(2, "two")
	instance.Update(map[int]string{3: "three", 4: "four"})

	if got := instance.GetLength(); got != 4 {
		t.Fatalf("Set/Update 后长度 = %d, want 4", got)
	}
	if val, ok := instance.Get(3); !ok || val != "three" {
		t.Errorf("Get(3) = (%q, %v), want (three, true)", val, ok)
	}
	if val, ok := instance.Get(99); ok || val != "" {
		t.Errorf("Get(99) = (%q, %v), want (\"\", false)", val, ok)
	}
	if !instance.HasAll(1, 2, 3, 4) {
		t.Error("HasAll(1,2,3,4) = false, want true")
	}
	if instance.HasAll(1, 99) {
		t.Error("HasAll(1,99) = true, want false")
	}
	if !instance.HasAny(99, 2) {
		t.Error("HasAny(99,2) = false, want true")
	}
	if instance.HasAny(98, 99) {
		t.Error("HasAny(98,99) = true, want false")
	}

	keys := instance.GetKeys()
	sort.Ints(keys)
	if len(keys) != 4 || keys[0] != 1 || keys[3] != 4 {
		t.Errorf("GetKeys() = %v, want [1 2 3 4]", keys)
	}
	if got := len(instance.GetValues()); got != 4 {
		t.Errorf("GetValues() 长度 = %d, want 4", got)
	}
	if got := len(instance.GetMap()); got != 4 {
		t.Errorf("GetMap() 长度 = %d, want 4", got)
	}

	visited := 0
	instance.Range(func(_ int, _ string) { visited++ })
	if visited != 4 {
		t.Errorf("Range 访问次数 = %d, want 4", visited)
	}

	items := Collect(instance.GetItems()).GetMap()
	if len(items) != 4 {
		t.Errorf("GetItems() 元素数 = %d, want 4", len(items))
	}
	for key, val := range instance.GetMap() {
		if items[key] != val {
			t.Errorf("GetItems() 中键 %d = %q, want %q", key, items[key], val)
		}
	}

	// 迭代器支持提前终止，不会遍历全部元素
	iterated := 0
	for range instance.GetItems() {
		iterated++
		if iterated == 2 {
			break
		}
	}
	if iterated != 2 {
		t.Errorf("GetItems() 提前终止后访问次数 = %d, want 2", iterated)
	}

	instance.Transform(func(_ int, val string) string { return val + "!" })
	if val, _ := instance.Get(1); val != "one!" {
		t.Errorf("Transform 后 Get(1) = %q, want one!", val)
	}

	removed := instance.Filter(func(key int, _ string) bool { return key%2 == 0 })
	if len(removed) != 2 || removed[2] != "two!" || removed[4] != "four!" {
		t.Errorf("Filter 返回 = %v, want 键 2/4", removed)
	}
	if got := instance.GetLength(); got != 2 {
		t.Errorf("Filter 后长度 = %d, want 2", got)
	}

	instance.Delete(1)
	if instance.HasAny(1) {
		t.Error("Delete(1) 后键 1 仍存在")
	}

	instance.Clear()
	if got := instance.GetLength(); got != 0 {
		t.Errorf("Clear 后长度 = %d, want 0", got)
	}
}
