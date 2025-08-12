package limiter

import (
	"sync"
	"time"
)

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	capacity   int64     // 桶容量
	tokens     int64     // 当前令牌数
	refillRate float64   // 令牌填充速率（tokens/second）
	lastRefill time.Time // 上次填充时间
	mutex      sync.Mutex
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(capacity int64, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity, // 初始满桶
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许请求
func (tb *TokenBucket) Allow() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.refill()

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// AllowN 检查是否允许N个请求
func (tb *TokenBucket) AllowN(n int64) bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.refill()

	if tb.tokens >= n {
		tb.tokens -= n
		return true
	}

	return false
}

// SetRate 动态设置填充速率
func (tb *TokenBucket) SetRate(rate float64) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.refillRate = rate
}

// GetTokens 获取当前令牌数
func (tb *TokenBucket) GetTokens() int64 {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.refill()
	return tb.tokens
}

// GetRate 获取当前填充速率
func (tb *TokenBucket) GetRate() float64 {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	return tb.refillRate
}

// refill 填充令牌（内部方法，需要加锁调用）
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()

	if elapsed > 0 {
		tokensToAdd := int64(elapsed * tb.refillRate)
		tb.tokens += tokensToAdd

		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}

		tb.lastRefill = now
	}
}

// Reset 重置令牌桶
func (tb *TokenBucket) Reset() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.tokens = tb.capacity
	tb.lastRefill = time.Now()
}

// IsEmpty 检查桶是否为空
func (tb *TokenBucket) IsEmpty() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.refill()
	return tb.tokens == 0
}

// IsFull 检查桶是否已满
func (tb *TokenBucket) IsFull() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.refill()
	return tb.tokens == tb.capacity
}

// GetCapacity 获取桶容量
func (tb *TokenBucket) GetCapacity() int64 {
	return tb.capacity
}

// SetCapacity 设置桶容量
func (tb *TokenBucket) SetCapacity(capacity int64) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.capacity = capacity
	if tb.tokens > capacity {
		tb.tokens = capacity
	}
}
