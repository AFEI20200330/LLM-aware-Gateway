package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 网关指标
	RequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Total number of requests processed by the gateway",
		},
		[]string{"method", "path", "status", "cluster_id"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "cluster_id"},
	)

	// 限流指标
	RateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"cluster_id", "policy_type"},
	)

	RateLimitAllowed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_allowed_total",
			Help: "Total number of rate limit allowed requests",
		},
		[]string{"cluster_id"},
	)

	// 熔断指标
	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"cluster_id"},
	)

	CircuitBreakerTrips = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_trips_total",
			Help: "Total number of circuit breaker trips",
		},
		[]string{"cluster_id"},
	)

	// 聚类指标
	ClusterSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cluster_size",
			Help: "Number of errors in cluster",
		},
		[]string{"cluster_id"},
	)

	ClusterSeverity = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cluster_severity",
			Help: "Cluster severity score",
		},
		[]string{"cluster_id"},
	)

	ClustersTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "clusters_total",
			Help: "Total number of active clusters",
		},
	)

	// 向量化指标
	VectorEmbeddingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "vector_embedding_duration_seconds",
			Help:    "Time spent on vector embedding",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		},
		[]string{"model"},
	)

	VectorEmbeddingTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vector_embedding_total",
			Help: "Total number of vector embeddings",
		},
		[]string{"model", "status"},
	)

	VectorCacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "vector_cache_hits_total",
			Help: "Total number of vector cache hits",
		},
	)

	VectorCacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "vector_cache_misses_total",
			Help: "Total number of vector cache misses",
		},
	)

	// 策略指标
	PolicyGenerated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "policy_generated_total",
			Help: "Total number of policies generated",
		},
		[]string{"cluster_id", "policy_type"},
	)

	PolicyApplied = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "policy_applied_total",
			Help: "Total number of policies applied",
		},
		[]string{"cluster_id", "policy_type"},
	)

	PolicyExpired = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "policy_expired_total",
			Help: "Total number of policies expired",
		},
		[]string{"cluster_id", "policy_type"},
	)

	ActivePolicies = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_policies",
			Help: "Number of active policies",
		},
		[]string{"policy_type"},
	)

	// Kafka指标
	KafkaMessagesProduced = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_produced_total",
			Help: "Total number of messages produced to Kafka",
		},
		[]string{"topic"},
	)

	KafkaMessagesConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_consumed_total",
			Help: "Total number of messages consumed from Kafka",
		},
		[]string{"topic", "group"},
	)

	KafkaProduceErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_produce_errors_total",
			Help: "Total number of Kafka produce errors",
		},
		[]string{"topic"},
	)

	KafkaConsumeErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_consume_errors_total",
			Help: "Total number of Kafka consume errors",
		},
		[]string{"topic", "group"},
	)

	// ETCD指标
	ETCDOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "etcd_operations_total",
			Help: "Total number of ETCD operations",
		},
		[]string{"operation", "status"},
	)

	ETCDOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "etcd_operation_duration_seconds",
			Help:    "Time spent on ETCD operations",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		},
		[]string{"operation"},
	)

	// Redis指标
	RedisOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_operations_total",
			Help: "Total number of Redis operations",
		},
		[]string{"operation", "status"},
	)

	RedisOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_operation_duration_seconds",
			Help:    "Time spent on Redis operations",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		},
		[]string{"operation"},
	)

	// 错误采样指标
	ErrorSampleRate = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "error_sample_rate",
			Help: "Current error sampling rate",
		},
	)

	ErrorSampled = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "error_sampled_total",
			Help: "Total number of errors sampled",
		},
	)

	ErrorSkipped = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "error_skipped_total",
			Help: "Total number of errors skipped (not sampled)",
		},
	)
)

// MetricsCollector 指标收集器
type MetricsCollector struct{}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// RecordRequest 记录请求指标
func (m *MetricsCollector) RecordRequest(method, path, status, clusterID string, duration float64) {
	RequestTotal.WithLabelValues(method, path, status, clusterID).Inc()
	RequestDuration.WithLabelValues(method, path, clusterID).Observe(duration)
}

// RecordRateLimit 记录限流指标
func (m *MetricsCollector) RecordRateLimit(clusterID, policyType string, allowed bool) {
	if allowed {
		RateLimitAllowed.WithLabelValues(clusterID).Inc()
	} else {
		RateLimitHits.WithLabelValues(clusterID, policyType).Inc()
	}
}

// RecordCircuitBreaker 记录熔断器指标
func (m *MetricsCollector) RecordCircuitBreaker(clusterID string, state int, trip bool) {
	CircuitBreakerState.WithLabelValues(clusterID).Set(float64(state))
	if trip {
		CircuitBreakerTrips.WithLabelValues(clusterID).Inc()
	}
}

// RecordCluster 记录聚类指标
func (m *MetricsCollector) RecordCluster(clusterID string, size int64, severity float64) {
	ClusterSize.WithLabelValues(clusterID).Set(float64(size))
	ClusterSeverity.WithLabelValues(clusterID).Set(severity)
}

// RecordClustersTotal 记录总聚类数
func (m *MetricsCollector) RecordClustersTotal(total int) {
	ClustersTotal.Set(float64(total))
}

// RecordVectorEmbedding 记录向量化指标
func (m *MetricsCollector) RecordVectorEmbedding(model, status string, duration float64) {
	VectorEmbeddingTotal.WithLabelValues(model, status).Inc()
	VectorEmbeddingDuration.WithLabelValues(model).Observe(duration)
}

// RecordVectorCache 记录向量缓存指标
func (m *MetricsCollector) RecordVectorCache(hit bool) {
	if hit {
		VectorCacheHits.Inc()
	} else {
		VectorCacheMisses.Inc()
	}
}

// RecordPolicy 记录策略指标
func (m *MetricsCollector) RecordPolicy(clusterID, policyType, action string) {
	switch action {
	case "generated":
		PolicyGenerated.WithLabelValues(clusterID, policyType).Inc()
	case "applied":
		PolicyApplied.WithLabelValues(clusterID, policyType).Inc()
	case "expired":
		PolicyExpired.WithLabelValues(clusterID, policyType).Inc()
	}
}

// RecordActivePolicies 记录活跃策略数
func (m *MetricsCollector) RecordActivePolicies(policyType string, count int) {
	ActivePolicies.WithLabelValues(policyType).Set(float64(count))
}

// RecordKafka 记录Kafka指标
func (m *MetricsCollector) RecordKafka(topic, group, operation, status string) {
	switch operation {
	case "produce":
		if status == "success" {
			KafkaMessagesProduced.WithLabelValues(topic).Inc()
		} else {
			KafkaProduceErrors.WithLabelValues(topic).Inc()
		}
	case "consume":
		if status == "success" {
			KafkaMessagesConsumed.WithLabelValues(topic, group).Inc()
		} else {
			KafkaConsumeErrors.WithLabelValues(topic, group).Inc()
		}
	}
}

// RecordETCD 记录ETCD指标
func (m *MetricsCollector) RecordETCD(operation, status string, duration float64) {
	ETCDOperations.WithLabelValues(operation, status).Inc()
	ETCDOperationDuration.WithLabelValues(operation).Observe(duration)
}

// RecordRedis 记录Redis指标
func (m *MetricsCollector) RecordRedis(operation, status string, duration float64) {
	RedisOperations.WithLabelValues(operation, status).Inc()
	RedisOperationDuration.WithLabelValues(operation).Observe(duration)
}

// RecordErrorSampling 记录错误采样指标
func (m *MetricsCollector) RecordErrorSampling(rate float64, sampled bool) {
	ErrorSampleRate.Set(rate)
	if sampled {
		ErrorSampled.Inc()
	} else {
		ErrorSkipped.Inc()
	}
}
