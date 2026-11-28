package dict

import (
	"strconv"
	"testing"
)

func TestDict(t *testing.T) {
	dict := NewShardSafeDict[string, int]()
	for i := -5000; i < 5001; i++ {
		dict.Set(strconv.Itoa(i), i)
	}

	t.Log(dict.Get("-5001"))
	t.Log(dict.Get("-5000"))
	t.Log(dict.Get("0"))
	t.Log(dict.Get("5000"))
	t.Log(dict.Get("5001"))
}

// cpu: Intel(R) Core(TM) i5-14600KF		20		84723025	        13.60 ns/op	       0 B/op	       0 allocs/op
func BenchmarkDict(b *testing.B) {
	dict := NewShardSafeDict[string, int]()
	for i := -5000; i < 5001; i++ {
		dict.Set(strconv.Itoa(i), i)
	}

	for i := 0; i < b.N; i++ {
		if val, ok := dict.Get("0"); !ok || val != 0 {
			b.Fatal(val)
		}
	}
}

// cpu: Intel(R) Core(TM) i5-14600KF		20		59955033	        17.78 ns/op	       0 B/op	       0 allocs/op
func BenchmarkSyncMap(b *testing.B) {
	dict := NewSafeDict[string, int]()
	for i := -5000; i < 5001; i++ {
		dict.Set(strconv.Itoa(i), i)
	}

	for i := 0; i < b.N; i++ {
		if val, ok := dict.Get("0"); !ok || val != 0 {
			b.Fatal(val)
		}
	}
}
