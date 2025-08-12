package gateway

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/llm-aware-gateway/pkg/interfaces"
	"github.com/llm-aware-gateway/pkg/types"
)

// metricsCollector Prometheus指标收集器
type metricsCollector struct {
	requestTotal         *prometheus.CounterVec
	requestDuration      *prometheus.HistogramVec
	rateLimitHits        *prometheus.CounterVec
	circuitBreakerState  *prometheus.GaugeVec
	clusterSize          *prometheus.GaugeVec
	clusterSeverity      *prometheus.GaugeVec
	policyApplied        *prometheus.CounterVec
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector() interfaces.MetricsCollector {
	mc := &metricsCollector{
		requestTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_requests_total",
				Help: "Total number of requests processed by gateway",
			},
			[]string{"method", "path", "status", "cluster_id"},
		),

		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gateway_request_duration_seconds",
				Help:    "Request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "cluster_id"},
		),

		rateLimitHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_rate_limit_hits_total",
				Help: "Total number of rate limit hits",
			},
			[]string{"cluster_id", "policy_type"},
		),

		circuitBreakerState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_circuit_breaker_state",
				Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
			},
			[]string{"cluster_id"},
		),

		clusterSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_cluster_size",
				Help: "Number of errors in cluster",
			},
			[]string{"cluster_id"},
		),

		clusterSeverity: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_cluster_severity",
				Help: "Cluster severity score",
			},
			[]string{"cluster_id"},
		),

		policyApplied: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_policy_applied_total",
				Help: "Total number of policies applied",
			},
			[]string{"cluster_id", "policy_type"},
		),
	}

	// 注册所有指标
	prometheus.MustRegister(
		mc.requestTotal,
		mc.requestDuration,
		mc.rateLimitHits,
		mc.circuitBreakerState,
		mc.clusterSize,
		mc.clusterSeverity,
		mc.policyApplied,
	)

	return mc
}

// RecordRequest 记录请求
func (mc *metricsCollector) RecordRequest(method, path, status, clusterID string, duration float64) {
	mc.requestTotal.WithLabelValues(method, path, status, clusterID).Inc()
	mc.requestDuration.WithLabelValues(method, path, clusterID).Observe(duration)
}

// RecordRateLimitHit 记录限流命中
func (mc *metricsCollector) RecordRateLimitHit(clusterID, policyType string) {
	mc.rateLimitHits.WithLabelValues(clusterID, policyType).Inc()
}

// RecordCircuitBreakerState 记录熔断器状态
func (mc *metricsCollector) RecordCircuitBreakerState(clusterID string, state types.BreakerState) {
	mc.circuitBreakerState.WithLabelValues(clusterID).Set(float64(state))
}

// UpdateClusterSize 更新簇大小
func (mc *metricsCollector) UpdateClusterSize(clusterID string, size int64) {
	mc.clusterSize.WithLabelValues(clusterID).Set(float64(size))
}

// UpdateClusterSeverity 更新簇严重度
func (mc *metricsCollector) UpdateClusterSeverity(clusterID string, severity float64) {
	mc.clusterSeverity.WithLabelValues(clusterID).Set(severity)
}

// RecordPolicyApplied 记录策略应用
func (mc *metricsCollector) RecordPolicyApplied(clusterID string, policyType types.PolicyType) {
	mc.policyApplied.WithLabelValues(clusterID, string(policyType)).Inc()
}
