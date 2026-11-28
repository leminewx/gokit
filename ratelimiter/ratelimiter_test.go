package ratelimiter

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLimiter_AllowBurst(t *testing.T) {
	l := New(10, 5) // 速率10/s，容量5
	// 容量内允许突发
	for i := 0; i < 5; i++ {
		if !l.Allow() {
			t.Fatalf("expected allow at %d", i)
		}
	}
	if l.Allow() {
		t.Fatal("expected deny after burst capacity exceeded")
	}
}

func TestLimiter_Refill(t *testing.T) {
	l := New(100, 1) // 速率100/s，容量1
	if !l.Allow() {
		t.Fatal("expected first allow")
	}
	if l.Allow() {
		t.Fatal("expected deny immediately after burst")
	}
	time.Sleep(20 * time.Millisecond) // 应补充 ~2 tokens
	if !l.Allow() {
		t.Fatal("expected allow after refill")
	}
}

func TestLimiter_Wait(t *testing.T) {
	l := New(1000, 1) // 速率1000/s，容量1
	if err := l.Wait(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := l.Wait(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLimiter_WaitContextCancel(t *testing.T) {
	l := New(1, 1) // 速率1/s，容量1
	_ = l.Allow()  // 耗尽
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := l.Wait(ctx); err == nil {
		t.Fatal("expected context deadline error")
	}
}

func TestLimiter_AllowN(t *testing.T) {
	l := New(100, 10)
	if !l.AllowN(10) {
		t.Fatal("expected allow N=10 within capacity")
	}
	if l.AllowN(1) {
		t.Fatal("expected deny after capacity consumed")
	}
}

func TestLimiter_AllowNNonPositive(t *testing.T) {
	l := New(1, 2) // 速率1/s，容量2

	if l.AllowN(0) {
		t.Fatal("AllowN(0) 应返回 false")
	}
	if l.AllowN(-1) {
		t.Fatal("AllowN(-1) 应返回 false")
	}

	// 负值不得反向增加令牌：容量内仍只能允许 2 个
	if !l.AllowN(2) {
		t.Fatal("expected allow N=2 within capacity")
	}
	if l.AllowN(1) {
		t.Fatal("negative n 不应增加桶内令牌")
	}
}

func TestLimiter_Concurrent(t *testing.T) {
	l := New(100000, 1000)
	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				l.Allow()
			}
			done <- true
		}()
	}
	for i := 0; i < 50; i++ {
		<-done
	}
}

// TestNewInvalidConfigPanics 验证非法参数在构造期被拒绝
func TestNewInvalidConfigPanics(t *testing.T) {
	cases := []struct {
		rate     float64
		capacity float64
	}{
		{-1, 1},
		{1, 0},
		{1, -1},
	}

	for _, tc := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("expected panic for rate=%v capacity=%v", tc.rate, tc.capacity)
				}
			}()

			New(tc.rate, tc.capacity)
		}()
	}
}

// TestWaitNUnattainable 验证永远无法满足的等待请求不会被永久阻塞
func TestWaitNUnattainable(t *testing.T) {
	// 速率为 0：只能消费初始突发额度
	l := New(0, 2)

	if err := l.WaitN(context.Background(), 3); !errors.Is(err, ErrUnattainable) {
		t.Fatalf("expected ErrUnattainable for n > capacity, got %v", err)
	}
	if err := l.WaitN(context.Background(), 0); !errors.Is(err, ErrUnattainable) {
		t.Fatalf("expected ErrUnattainable for n < 1, got %v", err)
	}

	// 初始令牌仍可正常获取
	if err := l.Wait(context.Background()); err != nil {
		t.Fatalf("unexpected error while tokens remain: %v", err)
	}
	if !l.Allow() {
		t.Fatal("expected remaining token to be available")
	}

	// 额度耗尽且不再补充时立即报错，而不是永久等待
	if err := l.Wait(context.Background()); !errors.Is(err, ErrUnattainable) {
		t.Fatalf("expected ErrUnattainable with zero rate, got %v", err)
	}
}
