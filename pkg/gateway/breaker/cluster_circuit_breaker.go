package breaker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/llm-aware-gateway/pkg/interfaces"
	"github.com/llm-aware-gateway/pkg/types"
	"github.com/llm-aware-gateway/pkg/utils"
)

// clusterCircuitBreaker 基于簇的熔断器
type clusterCircuitBreaker struct {
	config   *types.BreakerConfig
	clusters map[string]*clusterBreaker
	mutex    sync.RWMutex
}

// clusterBreaker 簇熔断器
type clusterBreaker struct {
	ClusterID     string
	State         types.BreakerState
	Policy        *types.Policy
	FailureCount  int64
	SuccessCount  int64
	LastFailTime  time.Time
	NextRetry     time.Time
	Config        *types.BreakerConfig
	Stats         *breakerStats
	mutex         sync.RWMutex
}

// breakerStats 熔断器统计
type breakerStats struct {
	TotalRequests    int64
	FailedRequests   int64
	SuccessRequests  int64
	BreakerOpenCount int64
	LastStateChange  time.Time
	mutex            sync.RWMutex
}

// NewClusterCircuitBreaker 创建基于簇的熔断器
func NewClusterCircuitBreaker(config *types.BreakerConfig) interfaces.CircuitBreaker {
	return &clusterCircuitBreaker{
		config:   config,
		clusters: make(map[string]*clusterBreaker),
	}
}

// Allow 检查是否允许请求
func (ccb *clusterCircuitBreaker) Allow(ctx context.Context, clusterID string) bool {
	if clusterID == "" {
		return true // 无簇信息，默认允许
	}

	ccb.mutex.RLock()
	breaker, exists := ccb.clusters[clusterID]
	ccb.mutex.RUnlock()

	if !exists {
		// 簇不存在熔断器，默认允许
		return true
	}

	breaker.mutex.Lock()
	defer breaker.mutex.Unlock()

	// 记录请求统计
	breaker.Stats.recordRequest()

	switch breaker.State {
	case types.BreakerStateClosed:
		// 关闭状态：允许请求
		return true

	case types.BreakerStateOpen:
		// 开启状态：检查是否可以转换为半开
		if time.Now().After(breaker.NextRetry) {
			breaker.setState(types.BreakerStateHalfOpen)
			log.Printf("Circuit breaker for cluster %s changed to HALF_OPEN", clusterID)
			return true
		}
		return false

	case types.BreakerStateHalfOpen:
		// 半开状态：允许部分请求
		return true

	default:
		return false
	}
}

// RecordSuccess 记录成功请求
func (ccb *clusterCircuitBreaker) RecordSuccess(clusterID string) error {
	if clusterID == "" {
		return nil
	}

	ccb.mutex.RLock()
	breaker, exists := ccb.clusters[clusterID]
	ccb.mutex.RUnlock()

	if !exists {
		return nil
	}

	breaker.mutex.Lock()
	defer breaker.mutex.Unlock()

	breaker.SuccessCount++
	breaker.Stats.recordSuccess()

	switch breaker.State {
	case types.BreakerStateHalfOpen:
		// 半开状态下的成功，可能转换为关闭状态
		recoveryThreshold := int64(float64(breaker.Config.FailureThreshold) * breaker.Config.RecoveryIncrement)
		if breaker.SuccessCount >= recoveryThreshold {
			breaker.setState(types.BreakerStateClosed)
			breaker.reset()
			log.Printf("Circuit breaker for cluster %s recovered to CLOSED", clusterID)
		}

	case types.BreakerStateOpen:
		// 开启状态下收到成功，重置一些计数器
		breaker.SuccessCount++
	}

	return nil
}

// RecordFailure 记录失败请求
func (ccb *clusterCircuitBreaker) RecordFailure(clusterID string) error {
	if clusterID == "" {
		return nil
	}

	ccb.mutex.RLock()
	breaker, exists := ccb.clusters[clusterID]
	ccb.mutex.RUnlock()

	if !exists {
		return nil
	}

	breaker.mutex.Lock()
	defer breaker.mutex.Unlock()

	breaker.FailureCount++
	breaker.LastFailTime = time.Now()
	breaker.Stats.recordFailure()

	switch breaker.State {
	case types.BreakerStateClosed:
		// 关闭状态下的失败，检查是否需要开启熔断
		if breaker.FailureCount >= breaker.Config.FailureThreshold {
			breaker.setState(types.BreakerStateOpen)
			breaker.NextRetry = time.Now().Add(breaker.Config.RecoveryTimeout)
			breaker.Stats.recordBreakerOpen()
			log.Printf("Circuit breaker for cluster %s opened due to failures", clusterID)
		}

	case types.BreakerStateHalfOpen:
		// 半开状态下的失败，重新开启熔断
		breaker.setState(types.BreakerStateOpen)
		breaker.NextRetry = time.Now().Add(breaker.Config.RecoveryTimeout)
		breaker.Stats.recordBreakerOpen()
		log.Printf("Circuit breaker for cluster %s re-opened due to failure in half-open state", clusterID)
	}

	return nil
}

// GetState 获取熔断器状态
func (ccb *clusterCircuitBreaker) GetState(clusterID string) types.BreakerState {
	if clusterID == "" {
		return types.BreakerStateClosed
	}

	ccb.mutex.RLock()
	breaker, exists := ccb.clusters[clusterID]
	ccb.mutex.RUnlock()

	if !exists {
		return types.BreakerStateClosed
	}

	breaker.mutex.RLock()
	defer breaker.mutex.RUnlock()

	return breaker.State
}

// UpdatePolicy 更新簇策略
func (ccb *clusterCircuitBreaker) UpdatePolicy(clusterID string, policy *types.Policy) error {
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	ccb.mutex.Lock()
	defer ccb.mutex.Unlock()

	breaker, exists := ccb.clusters[clusterID]
	if !exists {
		// 创建新的簇熔断器
		breaker = &clusterBreaker{
			ClusterID: clusterID,
			State:     types.BreakerStateClosed,
			Config:    ccb.config,
			Stats:     newBreakerStats(),
		}
		ccb.clusters[clusterID] = breaker
	}

	// 更新策略
	breaker.Policy = policy

	// 根据策略类型更新熔断参数
	if policy.PolicyType == types.PolicyTypeCircuitBreak && policy.CircuitBreak != nil {
		// 更新熔断配置
		breaker.mutex.Lock()
		breaker.Config = &types.BreakerConfig{
			FailureThreshold:  ccb.config.FailureThreshold,
			RecoveryTimeout:   policy.CircuitBreak.BreakDuration,
			RecoveryIncrement: policy.CircuitBreak.RecoveryStep,
		}

		// 如果策略要求立即熔断
		if policy.Severity >= 0.8 {
			breaker.setState(types.BreakerStateOpen)
			breaker.NextRetry = time.Now().Add(policy.CircuitBreak.BreakDuration)
			breaker.Stats.recordBreakerOpen()
			log.Printf("Circuit breaker for cluster %s immediately opened due to high severity", clusterID)
		}
		breaker.mutex.Unlock()

		log.Printf("Updated circuit breaker for cluster %s: timeout=%v, step=%.2f",
			clusterID, policy.CircuitBreak.BreakDuration, policy.CircuitBreak.RecoveryStep)
	}

	return nil
}

// setState 设置状态
func (cb *clusterBreaker) setState(state types.BreakerState) {
	cb.State = state
	cb.Stats.recordStateChange()
}

// reset 重置计数器
func (cb *clusterBreaker) reset() {
	cb.FailureCount = 0
	cb.SuccessCount = 0
}

// newBreakerStats 创建熔断器统计
func newBreakerStats() *breakerStats {
	return &breakerStats{
		LastStateChange: time.Now(),
	}
}

// recordRequest 记录请求
func (bs *breakerStats) recordRequest() {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()
	bs.TotalRequests++
}

// recordSuccess 记录成功
func (bs *breakerStats) recordSuccess() {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()
	bs.SuccessRequests++
}

// recordFailure 记录失败
func (bs *breakerStats) recordFailure() {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()
	bs.FailedRequests++
}

// recordBreakerOpen 记录熔断器开启
func (bs *breakerStats) recordBreakerOpen() {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()
	bs.BreakerOpenCount++
}

// recordStateChange 记录状态变更
func (bs *breakerStats) recordStateChange() {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()
	bs.LastStateChange = time.Now()
}

// getStats 获取统计信息
func (bs *breakerStats) getStats() (int64, int64, int64, int64) {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()
	return bs.TotalRequests, bs.SuccessRequests, bs.FailedRequests, bs.BreakerOpenCount
}
