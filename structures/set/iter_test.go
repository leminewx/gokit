package set

import (
	"sync"
	"testing"
)

// TestSetGetAll 验证三种实现的 GetAll 迭代器契约
func TestSetGetAll(t *testing.T) {
	impls := map[string]Set[int]{
		"UnsafeSet": NewUnsafeSet(1, 2, 3),
		"SafeSet":   NewSafeSet(1, 2, 3),
		"SyncSet":   NewSyncSet(1, 2, 3),
	}

	for name, instance := range impls {
		t.Run(name, func(t *testing.T) {
			collected := CollectSet(instance.GetAll())
			if got := collected.GetLength(); got != 3 {
				t.Fatalf("CollectSet(GetAll()) 长度 = %d, want 3", got)
			}
			if !collected.IsEqualTo(instance) {
				t.Error("CollectSet(GetAll()) 与原集合不相等")
			}

			// 迭代器支持提前终止
			visited := 0
			for range instance.GetAll() {
				visited++
				break
			}
			if visited != 1 {
				t.Errorf("提前终止后访问次数 = %d, want 1", visited)
			}
		})
	}
}

// TestUnionIntersect 验证多个集合的并集与交集
func TestUnionIntersect(t *testing.T) {
	first := NewUnsafeSet(1, 2, 3)
	second := NewSafeSet(2, 3, 4)
	third := NewSyncSet(3, 4, 5)

	union := Union(first, second, third)
	if !union.IsEqualTo(NewUnsafeSet(1, 2, 3, 4, 5)) {
		t.Errorf("Union() = %v, want [1 2 3 4 5]", union.GetSlice())
	}
	if !first.IsEqualTo(NewUnsafeSet(1, 2, 3)) {
		t.Error("Union 修改了原集合")
	}

	intersection := Intersect(first, second, third)
	if !intersection.IsEqualTo(NewUnsafeSet(3)) {
		t.Errorf("Intersect() = %v, want [3]", intersection.GetSlice())
	}

	if got := Intersect(first, NewUnsafeSet[int]()); !got.IsEmpty() {
		t.Errorf("与空集求交 = %v, want 空集", got.GetSlice())
	}
	if got := Intersect(first); !got.IsEqualTo(first) {
		t.Error("单参数 Intersect 应返回原集合的副本")
	}
}

// TestSafeSetAllConcurrent 验证 SafeSet.All 的快照遍历与并发写入不冲突
func TestSafeSetAllConcurrent(t *testing.T) {
	instance := NewSafeSet[int]()
	instance.Add(0, 1, 2, 3, 4)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 5; i < 500; i++ {
			instance.Add(i)
		}
	}()

	visited := 0
	for range instance.GetAll() {
		visited++
	}
	wg.Wait()

	if visited < 5 || visited > 500 {
		t.Errorf("GetAll() 访问次数 = %d, want [5, 500]", visited)
	}
}
