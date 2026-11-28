package set

import (
	"sort"
	"testing"
)

func TestConvertToSetImplementations(t *testing.T) {
	t.Parallel()

	converters := map[string]func(Set[int]) Set[int]{
		"UnsafeSet": ConvertToUnsafeSet[int],
		"SafeSet":   ConvertToSafeSet[int],
		"SyncSet":   ConvertToSyncSet[int],
	}

	for name, convert := range converters {
		source := NewUnsafeSet(1, 2, 3)
		converted := convert(source)

		got := converted.GetSlice()
		sort.Ints(got)
		if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
			t.Errorf("%s: 转换后元素 = %v, want [1 2 3]", name, got)
			continue
		}
		if converted.GetLength() != 3 {
			t.Errorf("%s: 转换后长度 = %d, want 3", name, converted.GetLength())
		}

		// 转换结果必须与原集合相互独立
		converted.Add(4)
		if source.HasAll(4) {
			t.Errorf("%s: 转换结果与原集合共享了底层数据", name)
		}
	}
}
