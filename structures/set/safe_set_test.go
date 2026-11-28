package set

import (
	"reflect"
	"sort"
	"testing"
)

// 测试 SafeSet 的所有方法
func TestSafeSet(t *testing.T) {
	// 使用 int 作为测试元素类型
	newSet := func(elems ...int) Set[int] {
		return NewSafeSet(elems...)
	}

	// 测试 Add 和 HasAll/HasAny
	t.Run("AddAndHas", func(t *testing.T) {
		s := newSet()
		s.Add(1, 2, 3)
		if !s.HasAll(1, 2, 3) {
			t.Error("HasAll failed after Add")
		}
		if s.HasAll(1, 2, 4) {
			t.Error("HasAll should be false")
		}
		if !s.HasAny(1, 4) {
			t.Error("HasAny should be true")
		}
		if s.HasAny(4, 5) {
			t.Error("HasAny should be false")
		}
	})

	// 测试 Delete
	t.Run("Delete", func(t *testing.T) {
		s := newSet(1, 2, 3)
		s.Delete(2, 3)
		if !s.HasAll(1) {
			t.Error("Delete removed wrong elements")
		}
		if s.HasAny(2, 3) {
			t.Error("Delete failed to remove elements")
		}
	})

	// 测试 GetLength 和 IsEmpty
	t.Run("LengthAndEmpty", func(t *testing.T) {
		s := newSet()
		if !s.IsEmpty() || s.GetLength() != 0 {
			t.Error("New set should be empty")
		}
		s.Add(1)
		if s.IsEmpty() || s.GetLength() != 1 {
			t.Error("Set should have length 1")
		}
		s.Delete(1)
		if !s.IsEmpty() || s.GetLength() != 0 {
			t.Error("Set should be empty after delete")
		}
	})

	// 测试 GetMap 和 GetSlice
	t.Run("GetMapAndSlice", func(t *testing.T) {
		s := newSet(1, 2, 3)
		m := s.GetMap()
		if len(m) != 3 {
			t.Errorf("GetMap length = %d, want 3", len(m))
		}
		for _, v := range []int{1, 2, 3} {
			if _, ok := m[v]; !ok {
				t.Errorf("GetMap missing %d", v)
			}
		}

		slice := s.GetSlice()
		if len(slice) != 3 {
			t.Errorf("GetSlice length = %d, want 3", len(slice))
		}
		// 排序后比较
		sort.Ints(slice)
		if !reflect.DeepEqual(slice, []int{1, 2, 3}) {
			t.Errorf("GetSlice = %v, want [1 2 3]", slice)
		}
	})

	// 测试 IsEqualTo
	t.Run("IsEqualTo", func(t *testing.T) {
		s1 := newSet(1, 2, 3)
		s2 := newSet(1, 2, 3)
		s3 := newSet(1, 2)
		s4 := newSet(1, 2, 4)

		if !s1.IsEqualTo(s2) {
			t.Error("Equal sets should be equal")
		}
		if s1.IsEqualTo(s3) || s1.IsEqualTo(s4) {
			t.Error("Different sets should not be equal")
		}
	})

	// 测试 IsSubsetOf 和 IsSupersetOf
	t.Run("SubsetSuperset", func(t *testing.T) {
		s123 := newSet(1, 2, 3)
		s12 := newSet(1, 2)
		s1234 := newSet(1, 2, 3, 4)
		s45 := newSet(4, 5)

		if !s12.IsSubsetOf(s123) {
			t.Error("s12 should be subset of s123")
		}
		if s123.IsSubsetOf(s12) {
			t.Error("s123 should not be subset of s12")
		}
		if !s123.IsSupersetOf(s12) {
			t.Error("s123 should be superset of s12")
		}
		if s12.IsSupersetOf(s123) {
			t.Error("s12 should not be superset of s123")
		}
		if !s123.IsSubsetOf(s1234) {
			t.Error("s123 should be subset of s1234")
		}
		if !s1234.IsSupersetOf(s123) {
			t.Error("s1234 should be superset of s123")
		}
		if s123.IsSubsetOf(s45) || s123.IsSupersetOf(s45) {
			t.Error("Disjoint sets should not be subset/superset")
		}
		// 空集
		empty := newSet()
		if !empty.IsSubsetOf(s123) {
			t.Error("Empty set should be subset of any set")
		}
		if !s123.IsSupersetOf(empty) {
			t.Error("Any set should be superset of empty set")
		}
	})

	// 测试 Union (并集)
	t.Run("Union", func(t *testing.T) {
		s1 := newSet(1, 2)
		s2 := newSet(2, 3)
		union := s1.Union(s2)
		expected := newSet(1, 2, 3)
		if !union.IsEqualTo(expected) {
			t.Errorf("Union result = %v, want %v", union.GetSlice(), expected.GetSlice())
		}
		// 原集合不应被修改
		if !s1.IsEqualTo(newSet(1, 2)) || !s2.IsEqualTo(newSet(2, 3)) {
			t.Error("Union modified original set")
		}
	})

	// 测试 Intersect (交集)
	t.Run("Intersect", func(t *testing.T) {
		s1 := newSet(1, 2, 3)
		s2 := newSet(2, 3, 4)
		intersection := s1.Intersect(s2)
		expected := newSet(2, 3)
		if !intersection.IsEqualTo(expected) {
			t.Errorf("Intersect result = %v, want %v", intersection.GetSlice(), expected.GetSlice())
		}
		// 无交集
		s3 := newSet(5, 6)
		empty := s1.Intersect(s3)
		if !empty.IsEmpty() {
			t.Error("Intersect of disjoint sets should be empty")
		}
	})

	// 测试 Complement (差集)
	t.Run("Complement", func(t *testing.T) {
		s1 := newSet(1, 2, 3, 4)
		s2 := newSet(2, 4, 6)
		diff := s1.Complement(s2) // s1 - s2
		expected := newSet(1, 3)
		if !diff.IsEqualTo(expected) {
			t.Errorf("Complement result = %v, want %v", diff.GetSlice(), expected.GetSlice())
		}
		// 不应包含另一集合独有的元素
		if diff.HasAny(6) {
			t.Error("Complement should not contain elements only in other set")
		}
	})

	// 测试 Extract 和 Filter
	t.Run("ExtractAndFilter", func(t *testing.T) {
		s := newSet(1, 2, 3, 4, 5)
		// Extract 不修改原集合
		evens := s.Extract(func(x int) bool { return x%2 == 0 })
		sort.Ints(evens)
		if !reflect.DeepEqual(evens, []int{2, 4}) {
			t.Errorf("Extract evens = %v, want [2 4]", evens)
		}
		if !s.IsEqualTo(newSet(1, 2, 3, 4, 5)) {
			t.Error("Extract modified original set")
		}
		// Filter 删除并返回
		odds := s.Filter(func(x int) bool { return x%2 != 0 })
		sort.Ints(odds)
		if !reflect.DeepEqual(odds, []int{1, 3, 5}) {
			t.Errorf("Filter odds = %v, want [1 3 5]", odds)
		}
		if !s.IsEqualTo(newSet(2, 4)) {
			t.Errorf("After Filter, set should be {2,4}, got %v", s.GetSlice())
		}
	})

	// 测试 Range 和 RangeUntil
	t.Run("RangeAndRangeUntil", func(t *testing.T) {
		s := newSet(1, 2, 3, 4, 5)
		// Range 遍历所有
		sum := 0
		s.Range(func(x int) { sum += x })
		if sum != 15 {
			t.Errorf("Range sum = %d, want 15", sum)
		}
		// RangeUntil 提前停止
		collected := []int{}
		s.RangeUntil(func(x int) bool {
			if x > 3 {
				return false
			}
			collected = append(collected, x)
			return true
		})
		// 由于遍历顺序不确定，只检查长度和最大值条件
		if len(collected) > 3 {
			t.Errorf("RangeUntil collected %d elements, should stop after 3", len(collected))
		}
		for _, v := range collected {
			if v > 3 {
				t.Errorf("RangeUntil collected %d > 3", v)
			}
		}
	})

	// 测试 Clone
	t.Run("Clone", func(t *testing.T) {
		s1 := newSet(1, 2, 3)
		s2 := s1.Clone()
		if !s1.IsEqualTo(s2) {
			t.Error("Clone should be equal to original")
		}
		s2.Add(4)
		if s1.HasAny(4) {
			t.Error("Clone should not share underlying data")
		}
	})

	// 测试 Clear
	t.Run("Clear", func(t *testing.T) {
		s := newSet(1, 2, 3)
		s.Clear()
		if !s.IsEmpty() {
			t.Error("Clear should empty the set")
		}
		s.Add(5)
		if !s.HasAll(5) {
			t.Error("Set should be usable after Clear")
		}
	})
}

/*
BenchmarkSafeSet/Add/10-8				2143550           591.5 ns/op             576 B/op          7 allocs/op
BenchmarkSafeSet/HasAll/10-8			15450835          78.63 ns/op               0 B/op          0 allocs/op
BenchmarkSafeSet/Delete/10-8			1408862           845.5 ns/op             576 B/op          7 allocs/op
BenchmarkSafeSet/Union/10-8				580876             1892 ns/op            1296 B/op         10 allocs/op
BenchmarkSafeSet/Intersect/10-8			1000000            1270 ns/op             688 B/op         17 allocs/op
BenchmarkSafeSet/Complement/10-8		806917             1855 ns/op             768 B/op         36 allocs/op
BenchmarkSafeSet/Range/10-8				9096441           133.6 ns/op               0 B/op          0 allocs/op

BenchmarkSafeSet/Add/100-8				197227             6327 ns/op            4704 B/op         13 allocs/op
BenchmarkSafeSet/HasAll/100-8			1673944           800.5 ns/op               0 B/op          0 allocs/op
BenchmarkSafeSet/Delete/100-8			170906             8016 ns/op            4704 B/op         13 allocs/op
BenchmarkSafeSet/Union/100-8			77052             14977 ns/op           11344 B/op         16 allocs/op
BenchmarkSafeSet/Intersect/100-8		119425             9855 ns/op            6352 B/op        113 allocs/op
BenchmarkSafeSet/Complement/100-8		73086             17276 ns/op            7056 B/op        312 allocs/op
BenchmarkSafeSet/Range/100-8			1410358           837.0 ns/op               0 B/op          0 allocs/op

BenchmarkSafeSet/Add/1k-8				13722             78186 ns/op           74512 B/op         24 allocs/op
BenchmarkSafeSet/HasAll/1k-8			143930             8144 ns/op               0 B/op          0 allocs/op
BenchmarkSafeSet/Delete/1k-8			12667             90874 ns/op           74512 B/op         24 allocs/op
BenchmarkSafeSet/Union/1k-8				8884             169745 ns/op          164736 B/op         34 allocs/op
BenchmarkSafeSet/Intersect/1k-8			10000            102621 ns/op           90656 B/op       1024 allocs/op
BenchmarkSafeSet/Complement/1k-8		7522             170276 ns/op           98464 B/op       3023 allocs/op
BenchmarkSafeSet/Range/1k-8				129712             8809 ns/op               0 B/op          0 allocs/op

BenchmarkSafeSet/Add/10k-8				2318             623945 ns/op          591730 B/op         83 allocs/op
BenchmarkSafeSet/HasAll/10k-8			15235             82532 ns/op               0 B/op          0 allocs/op
BenchmarkSafeSet/Delete/10k-8			1552             839078 ns/op          591731 B/op         83 allocs/op
BenchmarkSafeSet/Union/10k-8			816             1723890 ns/op         1346629 B/op        149 allocs/op
BenchmarkSafeSet/Intersect/10k-8		1005            1307623 ns/op          753604 B/op      10083 allocs/op
BenchmarkSafeSet/Complement/10k-8		516             1946016 ns/op          831688 B/op      30082 allocs/op
BenchmarkSafeSet/Range/10k-8			13390             89961 ns/op               0 B/op          0 allocs/op
*/
func BenchmarkSafeSet(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}
	for _, size := range sizes {
		elems := benchElems(size)
		b.Run("Add/"+itoa(size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s := NewSafeSet[int]()
				s.Add(elems...)
			}
		})

		b.Run("HasAll/"+itoa(size), func(b *testing.B) {
			s := NewSafeSet(elems...)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.HasAll(elems...)
			}
		})

		b.Run("Delete/"+itoa(size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s := NewSafeSet(elems...)
				s.Delete(elems...)
			}
		})
		b.Run("Union/"+itoa(size), func(b *testing.B) {
			elems1 := benchElems(size)
			elems2 := benchElems(size * 2)[size:] // 另一组不重叠元素，并集大小为 size*2
			s1 := NewSafeSet(elems1...)
			s2 := NewSafeSet(elems2...)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = s1.Union(s2)
			}
		})

		b.Run("Intersect/"+itoa(size), func(b *testing.B) {
			// 创建两个集合，一半重叠
			elems1 := benchElems(size * 2)        // 0..2*size-1
			elems2 := benchElems(size * 2)[size:] // size..3*size-1，与 elems1 重叠一半
			s1 := NewSafeSet(elems1...)
			s2 := NewSafeSet(elems2...)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = s1.Intersect(s2)
			}
		})

		b.Run("Complement/"+itoa(size), func(b *testing.B) {
			// 创建两个集合，一半重叠，计算 s1 - s2
			elems1 := benchElems(size * 2)        // 0..2*size-1
			elems2 := benchElems(size * 2)[size:] // size..3*size-1
			s1 := NewSafeSet(elems1...)
			s2 := NewSafeSet(elems2...)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = s1.Complement(s2)
			}
		})

		b.Run("Range/"+itoa(size), func(b *testing.B) {
			s := NewSafeSet(elems...)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.Range(func(int) {})
			}
		})
	}
}
