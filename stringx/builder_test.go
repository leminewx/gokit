package stringx

import (
	"sync"
	"testing"
)

func TestGetBuilder(t *testing.T) {
	builder := GetBuilder()
	if builder == nil {
		t.Fatal("GetBuilder 返回 nil")
	}

	builder.WriteString("hello")
	if got := builder.String(); got != "hello" {
		t.Errorf("builder.String() = %q, want %q", got, "hello")
	}

	PutBuilder(builder)
}

func TestPutBuilderResets(t *testing.T) {
	builder := GetBuilder()
	builder.WriteString("dirty")

	PutBuilder(builder)
	if got := builder.Len(); got != 0 {
		t.Errorf("回收到池后 builder 未重置，长度 = %d", got)
	}
}

func TestPutBuilderIgnoresNil(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PutBuilder(nil) 发生 panic: %v", r)
		}
	}()

	PutBuilder(nil)
}

func TestPutBuilderDropsOversized(t *testing.T) {
	builder := GetBuilder()
	builder.Grow(_MAX_POOLED_BUILDER_CAP + 1)
	if builder.Cap() <= _MAX_POOLED_BUILDER_CAP {
		t.Fatalf("Grow 后容量 = %d，未超过上限", builder.Cap())
	}

	// 超过上限的缓冲不应被重置后回收
	builder.WriteString("keep")
	PutBuilder(builder)
	if builder.Len() == 0 {
		t.Error("超过容量上限的 builder 被回收并重置")
	}
}

func TestBuilderPoolConcurrent(t *testing.T) {
	const goroutines = 32
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				builder := GetBuilder()
				builder.WriteString("gokit")
				if builder.String() != "gokit" {
					t.Errorf("复用的 builder 未重置")
					return
				}
				PutBuilder(builder)
			}
		}()
	}

	wg.Wait()
}

func BenchmarkBuilderPool(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			builder := GetBuilder()
			builder.WriteString("benchmark")
			PutBuilder(builder)
		}
	})
}
