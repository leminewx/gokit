package stream

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

// ---------- 构造函数测试 ----------
func TestNewStreamOf(t *testing.T) {
	SetStreamCacheSize(10)
	s := NewStreamOf(1, 2, 3)
	got := collect(s)
	want := []int{1, 2, 3}
	if !equalSlice(got, want) {
		t.Errorf("NewStreamOf = %v, want %v", got, want)
	}
}

func TestNewStreamBy(t *testing.T) {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)
	s := NewStreamBy(ch)
	got := collect(s)
	want := []int{1, 2, 3}
	if !equalSlice(got, want) {
		t.Errorf("NewStreamBy = %v, want %v", got, want)
	}
}

// ---------- 算子测试 ----------
func TestMap(t *testing.T) {
	s := NewStreamOf(1, 2, 3)
	mapped := Map(s, func(item Item[int]) string {
		return string(rune('a' + item.GetValue() - 1))
	})
	got := collect(mapped)
	want := []string{"a", "b", "c"}
	if !equalSlice(got, want) {
		t.Errorf("Map = %v, want %v", got, want)
	}
}

func TestTransform(t *testing.T) {
	s := NewStreamOf(1, 2, 3)
	transformed := s.Transform(func(item Item[int]) int {
		return item.GetValue() * 2
	})
	got := collect(transformed)
	want := []int{2, 4, 6}
	if !equalSlice(got, want) {
		t.Errorf("Transform = %v, want %v", got, want)
	}
}

func TestFilter(t *testing.T) {
	s := NewStreamOf(1, 2, 3, 4)
	filtered := s.Filter(func(item Item[int]) bool {
		return item.GetValue()%2 == 0
	})
	got := collect(filtered)
	want := []int{2, 4}
	if !equalSlice(got, want) {
		t.Errorf("Filter = %v, want %v", got, want)
	}
}

func TestPeek(t *testing.T) {
	var seen []int
	s := NewStreamOf(1, 2, 3)
	peeked := s.Peek(func(item Item[int]) {
		seen = append(seen, item.GetValue())
	})
	collect(peeked)
	want := []int{1, 2, 3}
	if !equalSlice(seen, want) {
		t.Errorf("Peek recorded %v, want %v", seen, want)
	}
}

func TestConsumeSync(t *testing.T) {
	s := NewStreamOf(1, 2)
	var consumed []int
	s.Consume(func(item Item[int]) {
		consumed = append(consumed, item.GetValue())
	})
	if !equalSlice(consumed, []int{1, 2}) {
		t.Errorf("Consume sync = %v, want [1,2]", consumed)
	}
}

func TestConsumeAsync(t *testing.T) {
	s := NewStreamOf(1, 2)
	var mu sync.Mutex
	var consumed []int
	s.Consume(func(item Item[int]) {
		mu.Lock()
		consumed = append(consumed, item.GetValue())
		mu.Unlock()
	}, true)
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if !equalSlice(consumed, []int{1, 2}) {
		t.Errorf("Consume async = %v, want [1,2]", consumed)
	}
}

func TestWait(t *testing.T) {
	s := NewStreamOf(1, 2)
	done := make(chan struct{})
	go func() {
		s.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Error("Wait did not finish")
	}
}

func TestReadAndChannel(t *testing.T) {
	// Test GetChannel
	s := NewStreamOf(1, 2)
	ch := s.GetChannel()
	var got []int
	for item := range ch {
		got = append(got, item.GetValue())
	}
	if !equalSlice(got, []int{1, 2}) {
		t.Errorf("Channel = %v, want [1,2]", got)
	}
	// Test Read
	s2 := NewStreamOf(3, 4)
	item, ok := s2.Read()
	if !ok || item.GetValue() != 3 {
		t.Errorf("Read first = %v, want 3", item.GetValue())
	}
	item, ok = s2.Read()
	if !ok || item.GetValue() != 4 {
		t.Errorf("Read second = %v, want 4", item.GetValue())
	}
	_, ok = s2.Read()
	if ok {
		t.Error("Read after finish should return false")
	}
}

// ---------- 边缘情况 ----------
func TestEmptyStream(t *testing.T) {
	s := NewStreamOf[int]()
	got := collect(s)
	if len(got) != 0 {
		t.Errorf("Empty stream returned %v, want empty", got)
	}
}

func TestNilFunctions(t *testing.T) {
	s := NewStreamOf(1, 2)
	// Should not panic
	_ = s.Transform(nil)
	_ = s.Group(nil)
	_ = s.Filter(nil)
	_ = s.Deduplicate(nil)
	_ = s.Peek(nil)
	s.Consume(nil)
}

// ---------- 辅助函数 ----------
// failWriter 用于模拟写入失败
type failWriter struct {
	io.Writer
	writeCount int
	failAfter  int
}

func (fw *failWriter) Write(p []byte) (n int, err error) {
	fw.writeCount++
	if fw.writeCount >= fw.failAfter {
		return 0, errors.New("模拟写入错误")
	}
	return fw.Writer.Write(p)
}

func collect[T any](s Stream[T]) []T {
	var result []T
	for item, ok := s.Read(); ok; item, ok = s.Read() {
		result = append(result, item.GetValue())
	}
	return result
}

func equalSlice[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func assertEqualIntSlice(t *testing.T, expected, actual []int) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("长度不匹配: 期望 %d, 实际 %d", len(expected), len(actual))
		return
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("索引 %d: 期望 %d, 实际 %d", i, expected[i], actual[i])
		}
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("非预期错误: %v", err)
	}
}

func assertLen(t *testing.T, slice interface{}, length int) {
	t.Helper()
	switch v := slice.(type) {
	case []int:
		if len(v) != length {
			t.Errorf("长度不匹配: 期望 %d, 实际 %d", length, len(v))
		}
	default:
		t.Errorf("不支持的类型: %T", slice)
	}
}
