package stream

import (
	"context"
	"sync"
	"testing"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

func TestSampleByStep(t *testing.T) {
	s := NewStreamOf(1, 2, 3, 4, 5)
	sampled := s.SampleByStep(2)
	got := collect(sampled)
	want := []int{1, 3, 5}
	if !equalSlice(got, want) {
		t.Errorf("SampleByStep = %v, want %v", got, want)
	}
}

func TestSampleByTime(t *testing.T) {
	// 真实时间采样，仅保证不 panic
	s := NewStreamOf(1, 2, 3)
	sampled := s.SampleByTime(10 * time.Millisecond)
	go func() {
		for range sampled.GetChannel() {
		}
	}()
	time.Sleep(50 * time.Millisecond)
}

func TestFanOut(t *testing.T) {
	s := NewStreamOf(1, 2, 3)
	streams := s.FanOut(2)
	if len(streams) != 2 {
		t.Fatalf("FanOut count = %d, want 2", len(streams))
	}
	var wg sync.WaitGroup
	wg.Add(2)
	results := make([][]int, 2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = collect(streams[idx])
		}(i)
	}
	wg.Wait()
	for i := 0; i < 2; i++ {
		if !equalSlice(results[i], []int{1, 2, 3}) {
			t.Errorf("FanOut stream %d = %v, want [1,2,3]", i, results[i])
		}
	}
}

// TestFanOutItemIsolation 验证子流之间不共享元素实例，下游修改不会污染兄弟子流
//   - 回归点：所有子流曾共享同一个 *item，Transform 原地修改后互相覆盖
func TestFanOutItemIsolation(t *testing.T) {
	SetStreamCacheSize(10)

	s := NewStreamOf(1, 2, 3)
	subs := s.FanOut(2)
	if len(subs) != 2 {
		t.Fatalf("FanOut count = %d, want 2", len(subs))
	}

	// 第一个子流原地放大元素值，第二个子流应保持原值
	scaledStream := subs[0].Transform(func(item Item[int]) int { return item.GetValue() * 10 })

	var wg sync.WaitGroup
	wg.Add(2)

	var scaled []int
	go func() {
		defer wg.Done()
		scaled = collect(scaledStream)
	}()

	var raw []int
	go func() {
		defer wg.Done()
		raw = collect(subs[1])
	}()

	wg.Wait()

	if !equalSlice(scaled, []int{10, 20, 30}) {
		t.Errorf("transformed stream = %v, want [10,20,30]", scaled)
	}
	if !equalSlice(raw, []int{1, 2, 3}) {
		t.Errorf("sibling stream polluted = %v, want [1,2,3]", raw)
	}
}

// TestSampleByTimeWithClock 验证时间采样：每个间隔输出最新的元素
func TestSampleByTimeWithClock(t *testing.T) {
	t.Parallel()

	const interval = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fake := timex.NewManual(testEpoch)
	source, s := testSource[int](ctx, fake)
	sampled := s.SampleByTime(interval)

	samples := make(chan int, 8)
	go func() {
		for {
			item, ok := sampled.Read()
			if !ok {
				return
			}

			samples <- item.GetValue()
		}
	}()

	// 同一间隔内到达多条，只输出最新的一条
	source <- NewItem(1)
	source <- NewItem(2)
	if value := nextSample(t, fake, interval, samples); value != 2 {
		t.Fatalf("采样值错误: 期望 2, 实际 %d", value)
	}

	// 空闲间隔不重复输出
	fake.Advance(interval)
	select {
	case value := <-samples:
		t.Fatalf("空闲间隔不应重复输出, 实际 %d", value)
	case <-time.After(50 * time.Millisecond):
	}

	// 新元素到达后恢复输出
	source <- NewItem(3)
	if value := nextSample(t, fake, interval, samples); value != 3 {
		t.Fatalf("采样值错误: 期望 3, 实际 %d", value)
	}

	// 流结束时输出尚未输出的最后一条
	source <- NewItem(4)
	time.Sleep(5 * time.Millisecond)
	close(source)

	select {
	case value := <-samples:
		if value != 4 {
			t.Fatalf("流结束时的采样值错误: 期望 4, 实际 %d", value)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("流结束时未输出最后一条元素")
	}
}

// nextSample 反复推进时钟直到产出下一个采样值
func nextSample(t *testing.T, fake *timex.Manual, interval time.Duration, samples <-chan int) int {
	t.Helper()

	deadline := time.After(2 * time.Second)
	for {
		fake.Advance(interval)

		select {
		case value := <-samples:
			return value
		case <-deadline:
			t.Fatal("等待采样输出超时")
		case <-time.After(5 * time.Millisecond):
		}
	}
}
