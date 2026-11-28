package stream

import (
	"testing"
	"time"
)

func TestGroup(t *testing.T) {
	s := NewStreamOf(1, 2, 3)
	grouped := s.Group(func(item Item[int]) string {
		if item.GetValue()%2 == 0 {
			return "even"
		}
		return "odd"
	})
	// Verify group labels
	var groups []string
	for item := range grouped.GetChannel() {
		groups = append(groups, item.GetGroup())
	}
	wantGroups := []string{"odd", "even", "odd"}
	if !equalSlice(groups, wantGroups) {
		t.Errorf("Group labels = %v, want %v", groups, wantGroups)
	}
}

func TestConcat(t *testing.T) {
	s1 := NewStreamOf(1, 2)
	s2 := NewStreamOf(3, 4)
	s3 := NewStreamOf(5)
	concat := s1.Concat(s2, s3)
	got := collect(concat)
	if len(got) != 5 {
		t.Errorf("Concat length = %d, want 5", len(got))
	}
	contains := make(map[int]bool)
	for _, v := range got {
		contains[v] = true
	}
	for i := 1; i <= 5; i++ {
		if !contains[i] {
			t.Errorf("Concat missing value %d", i)
		}
	}
}

func TestDeduplicateNoWindow(t *testing.T) {
	s := NewStreamOf(1, 2, 1, 3, 2)
	dedup := s.Deduplicate(func(item Item[int]) any { return item.GetValue() })
	got := collect(dedup)
	want := []int{1, 2, 3}
	if !equalSlice(got, want) {
		t.Errorf("Deduplicate no window = %v, want %v", got, want)
	}
}

func TestDeduplicateWithWindow(t *testing.T) {
	s := NewStreamOf(1, 2, 1)
	dedup := s.Deduplicate(func(item Item[int]) any { return item.GetValue() }, 100*time.Millisecond)
	got := collect(dedup)
	want := []int{1, 2}
	if !equalSlice(got, want) {
		t.Errorf("Deduplicate with window = %v, want %v", got, want)
	}
}

func TestTakeFirst(t *testing.T) {
	s := NewStreamOf(1, 2, 3, 4)
	head := s.TakeFirst(2)
	got := collect(head)
	want := []int{1, 2}
	if !equalSlice(got, want) {
		t.Errorf("TakeFirst = %v, want %v", got, want)
	}
}

func TestTakeLast(t *testing.T) {
	s := NewStreamOf(1, 2, 3, 4)
	tail := s.TakeLast(2)
	got := collect(tail)
	want := []int{3, 4}
	if !equalSlice(got, want) {
		t.Errorf("TakeLast = %v, want %v", got, want)
	}
}
