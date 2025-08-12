package interfaces

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/llm-aware-gateway/pkg/types"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(ctx *gin.Context) bool
	UpdatePolicy(clusterID string, policy *types.Policy) error
	GetStats(clusterID string) (*types.ClusterStats, error)
	Cleanup() error
}

// CircuitBreaker 熔断器接口
type CircuitBreaker interface {
	Allow(ctx context.Context, clusterID string) bool
	RecordSuccess(clusterID string) error
	RecordFailure(clusterID string) error
	GetState(clusterID string) types.BreakerState
	UpdatePolicy(clusterID string, policy *types.Policy) error
}

// ErrorSampler 错误采样器接口
type ErrorSampler interface {
	SampleError(ctx *gin.Context, err error) error
	Start() error
	Stop() error
}

// VectorAgent 向量代理接口
type VectorAgent interface {
	IdentifyCluster(errorSignature string) (string, error)
	GenerateVector(text string) ([]float32, error)
	UpdateClusters(clusters map[string]*types.Cluster) error
}

// ConfigWatcher 配置监听器接口
type ConfigWatcher interface {
	WatchPolicyUpdates() error
	GetPolicy(clusterID string) (*types.Policy, error)
	RegisterCallback(callback PolicyUpdateCallback) error
	Start() error
	Stop() error
}

// PolicyUpdateCallback 策略更新回调接口
type PolicyUpdateCallback interface {
	OnPolicyUpdate(clusterID string, policy *types.Policy) error
	OnPolicyDelete(clusterID string) error
}

// EmbeddingService 嵌入服务接口
type EmbeddingService interface {
	EmbedText(text string) ([]float32, error)
	EmbedBatch(texts []string) ([][]float32, error)
	PreprocessText(text string) string
}

// ClusteringEngine 聚类引擎接口
type ClusteringEngine interface {
	ProcessErrorEvent(event *types.ErrorEvent) error
	FindMostSimilarCluster(vector []float32) (string, float64, error)
	CreateNewCluster(event *types.ErrorEvent, vector []float32) (string, error)
	GetCluster(clusterID string) (*types.Cluster, error)
	GetAllClusters() (map[string]*types.Cluster, error)
	ReCluster() error
	Start() error
	Stop() error
}

// PolicyEngine 策略引擎接口
type PolicyEngine interface {
	EvaluatePolicies() error
	GeneratePolicy(cluster *types.Cluster, errorRate, growthRate float64) (*types.Policy, error)
	ApplyPolicy(policy *types.Policy) error
	ShouldTriggerPolicy(errorRate, growthRate float64) bool
	CalculateErrorRate(clusterID string, windowSize int64) (float64, error)
	CalculateGrowthRate(clusterID string, windowSize int64) (float64, error)
	Start() error
	Stop() error
}

// VectorDB 向量数据库接口
type VectorDB interface {
	AddVector(id string, vector []float32) error
	SearchSimilar(query []float32, topK int) ([]types.SearchResult, error)
	GetVector(id string) ([]float32, error)
	DeleteVector(id string) error
	GetVectorCount() (int64, error)
}

// ConfigStore 配置存储接口
type ConfigStore interface {
	Put(key string, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	Watch(prefix string) (<-chan *ConfigChangeEvent, error)
	Close() error
}

// ConfigChangeEvent 配置变更事件
type ConfigChangeEvent struct {
	Type  ConfigChangeType
	Key   string
	Value string
}

// ConfigChangeType 配置变更类型
type ConfigChangeType int

const (
	ConfigChangeTypePut ConfigChangeType = iota
	ConfigChangeTypeDelete
)

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	RecordRequest(method, path, status, clusterID string, duration float64)
	RecordRateLimitHit(clusterID, policyType string)
	RecordCircuitBreakerState(clusterID string, state types.BreakerState)
	UpdateClusterSize(clusterID string, size int64)
	UpdateClusterSeverity(clusterID string, severity float64)
	RecordPolicyApplied(clusterID string, policyType types.PolicyType)
}

// Desensitizer 脱敏器接口
type Desensitizer interface {
	Desensitize(text string) string
	AddPattern(name string, pattern string, replacement string)
}

// KafkaProducer Kafka生产者接口
type KafkaProducer interface {
	SendMessage(topic string, key string, value []byte) error
	Close() error
}

// KafkaConsumer Kafka消费者接口
type KafkaConsumer interface {
	Subscribe(topic string, handler MessageHandler) error
	Start() error
	Stop() error
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleMessage(message []byte) error
}

// TokenBucket 令牌桶接口
type TokenBucket interface {
	Allow() bool
	SetRate(rate float64)
	GetTokens() int64
	GetCapacity() int64
}

// Cache 缓存接口
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl int64) error
	Delete(key string) error
	Clear() error
	Size() int64
}
