package middleware

import (
	"context"
	"sync"
	"time"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(ctx context.Context) bool
}

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	capacity   int64      // 桶容量
	tokens     int64      // 当前令牌数
	refillRate int64      // 每秒补充的令牌数
	lastRefill time.Time  // 上次补充时间
	mu         sync.Mutex // 锁
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许请求
func (tb *TokenBucket) Allow(ctx context.Context) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()

	// 补充令牌
	tokensToAdd := int64(elapsed * float64(tb.refillRate))
	if tokensToAdd > 0 {
		tb.tokens = min(tb.tokens+tokensToAdd, tb.capacity)
		tb.lastRefill = now
	}

	// 检查是否有足够的令牌
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// SlidingWindow 滑动窗口限流器
type SlidingWindow struct {
	requests    []time.Time   // 请求时间记录
	window      time.Duration // 时间窗口
	maxRequests int           // 最大请求数
	mu          sync.Mutex
}

// NewSlidingWindow 创建滑动窗口限流器
func NewSlidingWindow(window time.Duration, maxRequests int) *SlidingWindow {
	return &SlidingWindow{
		requests:    make([]time.Time, 0),
		window:      window,
		maxRequests: maxRequests,
	}
}

// Allow 检查是否允许请求
func (sw *SlidingWindow) Allow(ctx context.Context) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-sw.window)

	// 清理窗口外的请求
	validRequests := make([]time.Time, 0)
	for _, reqTime := range sw.requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}
	sw.requests = validRequests

	// 检查是否超过限制
	if len(sw.requests) < sw.maxRequests {
		sw.requests = append(sw.requests, now)
		return true
	}

	return false
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
