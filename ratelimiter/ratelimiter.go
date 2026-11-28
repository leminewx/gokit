// Package ratelimiter 提供基于令牌桶算法的并发安全限流器。
//
// 仅依赖标准库，无第三方依赖。
package ratelimiter

import (
	"context"
	"math"
	"sync"
	"time"
)

// Limiter 令牌桶限流器。
type Limiter struct {
	mu       sync.Mutex
	capacity float64
	tokens   float64
	rate     float64 // tokens per second
	last     time.Time
	monoNow  func() time.Time
}

// New 创建令牌桶限流器。
//   - rate: 每秒补充的令牌数，不得为负（为 0 表示不补充，仅使用初始突发额度）
//   - capacity: 桶容量（突发上限），必须为正
//   - 参数非法时 panic ErrInvalidConfig，与 scheduler 等包的构造行为保持一致
func New(rate float64, capacity float64) *Limiter {
	if math.IsNaN(rate) || math.IsNaN(capacity) || rate < 0 || capacity <= 0 || math.IsInf(capacity, 0) {
		panic(ErrInvalidConfig)
	}

	return &Limiter{
		capacity: capacity,
		tokens:   capacity,
		rate:     rate,
		last:     time.Now(),
		monoNow:  time.Now,
	}
}

func (own *Limiter) refill() {
	now := own.monoNow()
	elapsed := now.Sub(own.last).Seconds()
	if elapsed <= 0 {
		return
	}
	own.tokens += elapsed * own.rate
	if own.tokens > own.capacity {
		own.tokens = own.capacity
	}
	own.last = now
}

// Allow 非阻塞地尝试获取一个令牌。
func (own *Limiter) Allow() bool {
	return own.AllowN(1)
}

// AllowN 非阻塞地尝试获取 n 个令牌。
//   - n < 1 时无法获取（与 WaitN 的参数约束保持一致），且不会改变桶内令牌数
func (own *Limiter) AllowN(n int) bool {
	if n < 1 {
		return false
	}

	own.mu.Lock()
	defer own.mu.Unlock()
	own.refill()
	if own.tokens >= float64(n) {
		own.tokens -= float64(n)
		return true
	}
	return false
}

// Wait 阻塞直到获取到一个令牌或 ctx 被取消。
func (own *Limiter) Wait(ctx context.Context) error {
	return own.WaitN(ctx, 1)
}

// WaitN 阻塞直到获取到 n 个令牌或 ctx 被取消。
//   - ctx 为 nil 时按 context.Background() 处理
//   - n < 1、n 超过桶容量或速率为 0 且当前令牌不足时返回 ErrUnattainable，避免永久阻塞
func (own *Limiter) WaitN(ctx context.Context, n int) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if n < 1 {
		return wrapError(ErrUnattainable, "tokens must be positive, got %d", n)
	}

	for {
		own.mu.Lock()
		own.refill()
		if own.tokens >= float64(n) {
			own.tokens -= float64(n)
			own.mu.Unlock()
			return nil
		}

		// 计算下次可能有令牌的时间，同时判断请求是否永远无法满足
		deficit := float64(n) - own.tokens
		rate, capacity := own.rate, own.capacity
		own.mu.Unlock()

		if rate <= 0 || float64(n) > capacity {
			return wrapError(ErrUnattainable, "want %d tokens with rate %.3f and capacity %.3f", n, rate, capacity)
		}

		wait := time.Duration(deficit / rate * float64(time.Second))
		if wait <= 0 {
			// 理论上不会出现：避免极小间隔导致忙等
			wait = time.Millisecond
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// SetRate 动态修改速率与容量。
//   - 参数约束与 New 一致，非法时 panic ErrInvalidConfig
//   - 容量下调后，超出新容量的令牌会被立即丢弃
func (own *Limiter) SetRate(rate, capacity float64) {
	if math.IsNaN(rate) || math.IsNaN(capacity) || rate < 0 || capacity <= 0 || math.IsInf(capacity, 0) {
		panic(ErrInvalidConfig)
	}

	own.mu.Lock()
	defer own.mu.Unlock()
	own.refill()
	own.rate = rate
	own.capacity = capacity
	if own.tokens > capacity {
		own.tokens = capacity
	}
}
