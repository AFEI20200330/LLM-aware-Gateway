package utils

import (
	"sync"
	"time"

	"github.com/llm-aware-gateway/pkg/interfaces"
)

// tokenBucket 令牌桶实现
type tokenBucket struct {
	capacity     int64   // 桶容量
	tokens       int64   // 当前令牌数
	refillRate   float64 // 每秒补充速率
	lastRefill   time.Time
	mutex        sync.Mutex
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(capacity int64, refillRate float64) interfaces.TokenBucket {
	return &tokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许请求
func (tb *tokenBucket) Allow() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.refill()

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// SetRate 设置补充速率
func (tb *tokenBucket) SetRate(rate float64) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.refill()
	tb.refillRate = rate
}

// GetTokens 获取当前令牌数
func (tb *tokenBucket) GetTokens() int64 {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.refill()
	return tb.tokens
}

// GetCapacity 获取桶容量
func (tb *tokenBucket) GetCapacity() int64 {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	return tb.capacity
}

// refill 补充令牌
func (tb *tokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()

	if elapsed > 0 {
		tokensToAdd := int64(elapsed * tb.refillRate)
		tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
		tb.lastRefill = now
	}
}

// min 返回两个int64中的较小值
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
