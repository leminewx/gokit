package stream

import (
	"testing"
	"time"
)

func TestZipLeft(t *testing.T) {
	s1 := NewStreamOf(1, 2, 3)
	s2 := NewStreamOf("a", "b", "c")
	match := func(a Item[int], b Item[string]) bool {
		return a.GetValue() == int(b.GetValue()[0]-'a'+1)
	}
	mapper := func(a Item[int], b Item[string]) string {
		return string(rune('a'+a.GetValue()-1)) + b.GetValue()
	}
	zipped := ZipLeft(s1, s2, match, mapper, 1*time.Second, nil)
	got := collect(zipped)
	want := []string{"aa", "bb", "cc"}
	if !equalSlice(got, want) {
		t.Errorf("ZipLeft = %v, want %v", got, want)
	}
}

func TestZipLeftByKey(t *testing.T) {
	s1 := NewStreamOf(1, 2, 3)
	s2 := NewStreamOf(2, 1, 4)
	zipped := ZipLeftByKey(s1, s2,
		func(a Item[int]) int { return a.GetValue() },
		func(b Item[int]) int { return b.GetValue() },
		func(a, b Item[int]) string {
			return string(rune('a'+a.GetValue()-1)) + string(rune('a'+b.GetValue()-1))
		},
		500*time.Millisecond,
		func(a Item[int]) (string, bool) {
			return string(rune('a'+a.GetValue()-1)) + "?", true
		},
	)
	got := collect(zipped)
	// 1 -> 匹配到 s2 中的 1（第二个） -> "aa"
	// 2 -> 匹配到 s2 中的 2（第一个） -> "bb"
	// 3 -> 超时，使用默认 -> "c?"
	want := []string{"aa", "bb", "c?"}
	if !equalSlice(got, want) {
		t.Errorf("ZipLeftByKey = %v, want %v", got, want)
	}
}
