package middleware

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota // 关闭状态（正常）
	StateOpen                              // 开启状态（熔断）
	StateHalfOpen                          // 半开状态（尝试恢复）
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name          string
	maxFailures   int           // 最大失败次数
	resetTimeout  time.Duration // 重置超时时间
	halfOpenMax   int           // 半开状态最大请求数
	failures      int           // 当前失败次数
	halfOpenCount int           // 半开状态请求计数
	state         CircuitBreakerState
	lastFailTime  time.Time
	mu            sync.RWMutex
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(name string, maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:         name,
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		halfOpenMax:  3,
		state:        StateClosed,
	}
}

// Call 执行调用
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	cb.mu.Lock()
	state := cb.state

	// 检查是否需要状态转换
	if state == StateOpen {
		if time.Since(cb.lastFailTime) >= cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.halfOpenCount = 0
			state = StateHalfOpen
		} else {
			cb.mu.Unlock()
			return errors.New("circuit breaker is open")
		}
	}

	if state == StateHalfOpen {
		if cb.halfOpenCount >= cb.halfOpenMax {
			cb.mu.Unlock()
			return errors.New("circuit breaker half-open limit reached")
		}
		cb.halfOpenCount++
	}

	cb.mu.Unlock()

	// 执行函数
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.state == StateHalfOpen {
			// 半开状态下失败，重新开启熔断
			cb.state = StateOpen
			cb.halfOpenCount = 0
		} else if cb.failures >= cb.maxFailures {
			// 达到最大失败次数，开启熔断
			cb.state = StateOpen
		}
	} else {
		// 成功，重置计数
		if cb.state == StateHalfOpen {
			cb.state = StateClosed
			cb.halfOpenCount = 0
		}
		cb.failures = 0
	}

	return err
}

// GetState 获取当前状态
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
