
import (
	"time"
)

// ErrorEvent 错误事件结构
type ErrorEvent struct {
	EventID      string    `json:"event_id"`
	TraceID      string    `json:"trace_id"`
	SpanID       string    `json:"span_id"`
	RequestPath  string    `json:"request_path"`
	Method       string    `json:"method"`
	ServiceName  string    `json:"service_name"`
	StatusCode   int       `json:"status_code"`
	ErrorMessage string    `json:"error_message"`
	StackTrace   []string  `json:"stack_trace"`
	Timestamp    time.Time `json:"timestamp"`
	ClusterID    string    `json:"cluster_id,omitempty"`
}

// Cluster 异常簇结构
type Cluster struct {
	ID          string    `json:"id"`
	Centroid    []float32 `json:"centroid"`
	Members     []string  `json:"members"`
	ErrorCount  int64     `json:"error_count"`
	CreateTime  time.Time `json:"create_time"`
	UpdateTime  time.Time `json:"update_time"`
	Severity    float64   `json:"severity"`
	Description string    `json:"description"`
}

// PolicyType 策略类型
type PolicyType string

const (
	PolicyTypeRateLimit    PolicyType = "RATE_LIMIT"
	PolicyTypeCircuitBreak PolicyType = "CIRCUIT_BREAK"
	PolicyTypeDegrade      PolicyType = "DEGRADE"
)

// Policy 策略结构
type Policy struct {
	PolicyID     string               `json:"policy_id"`
	ClusterID    string               `json:"cluster_id"`
	PolicyType   PolicyType           `json:"policy_type"`
	Severity     float64              `json:"severity"`
	RateLimit    *RateLimitPolicy     `json:"rate_limit,omitempty"`
	CircuitBreak *CircuitBreakPolicy  `json:"circuit_break,omitempty"`
	CreateTime   time.Time            `json:"create_time"`
	ExpireTime   time.Time            `json:"expire_time"`
	IsActive     bool                 `json:"is_active"`
}

// RateLimitPolicy 限流策略
type RateLimitPolicy struct {
	LimitRate float64       `json:"limit_rate"` // 限制比例 0.0-1.0
	Duration  time.Duration `json:"duration"`
}

// CircuitBreakPolicy 熔断策略
type CircuitBreakPolicy struct {
	BreakDuration time.Duration `json:"break_duration"`
	RecoveryStep  float64       `json:"recovery_step"` // 恢复步长
}

// BreakerState 熔断器状态
type BreakerState int

const (
	BreakerStateClosed BreakerState = iota
	BreakerStateOpen
	BreakerStateHalfOpen
)

// SearchResult 向量搜索结果
type SearchResult struct {
	ID         string  `json:"id"`
	Similarity float64 `json:"similarity"`
	Vector     []float32 `json:"vector,omitempty"`
}

// ClusterStats 簇统计信息
type ClusterStats struct {
	ClusterID    string    `json:"cluster_id"`
	ErrorRate    float64   `json:"error_rate"`
	GrowthRate   float64   `json:"growth_rate"`
	LastUpdate   time.Time `json:"last_update"`
	WindowSize   time.Duration `json:"window_size"`
}

// GatewayConfig 网关配置
type GatewayConfig struct {
	Server   ServerConfig   `yaml:"server"`
	Limiter  LimiterConfig  `yaml:"limiter"`
	Breaker  BreakerConfig  `yaml:"breaker"`
	Sampler  SamplerConfig  `yaml:"sampler"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	ETCD     ETCDConfig     `yaml:"etcd"`
	Metrics  MetricsConfig  `yaml:"metrics"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// LimiterConfig 限流器配置
type LimiterConfig struct {
	DefaultRate      float64 `yaml:"default_rate"`
	MaxRate          float64 `yaml:"max_rate"`
	CleanupInterval  time.Duration `yaml:"cleanup_interval"`
}

// BreakerConfig 熔断器配置
type BreakerConfig struct {
	FailureThreshold  int64         `yaml:"failure_threshold"`
	RecoveryTimeout   time.Duration `yaml:"recovery_timeout"`
	RecoveryIncrement float64       `yaml:"recovery_increment"`
}

// SamplerConfig 采样器配置
type SamplerConfig struct {
	SamplingRate float64 `yaml:"sampling_rate"`
	BufferSize   int     `yaml:"buffer_size"`
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`
}

// ETCDConfig ETCD配置
type ETCDConfig struct {
	Endpoints []string      `yaml:"endpoints"`
	Username  string        `yaml:"username"`
	Password  string        `yaml:"password"`
	Timeout   time.Duration `yaml:"timeout"`
}

// MetricsConfig 监控配置
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

// ControlPlaneConfig 控制面配置
type ControlPlaneConfig struct {
	Server      ServerConfig      `yaml:"server"`
	Embedding   EmbeddingConfig   `yaml:"embedding"`
	Clustering  ClusteringConfig  `yaml:"clustering"`
	Policy      PolicyConfig      `yaml:"policy"`
	VectorDB    VectorDBConfig    `yaml:"vectordb"`
	Kafka       KafkaConfig       `yaml:"kafka"`
	ETCD        ETCDConfig        `yaml:"etcd"`
}

// EmbeddingConfig 嵌入服务配置
type EmbeddingConfig struct {
	ModelPath  string `yaml:"model_path"`
	BatchSize  int    `yaml:"batch_size"`
	CacheSize  int    `yaml:"cache_size"`
	Dimension  int    `yaml:"dimension"`
}

// ClusteringConfig 聚类配置
type ClusteringConfig struct {
	SimilarityThreshold   float64       `yaml:"similarity_threshold"`
	ReclusteringInterval  time.Duration `yaml:"reclustering_interval"`
	MinClusterSize        int           `yaml:"min_cluster_size"`
	MaxClusters           int           `yaml:"max_clusters"`
}

// PolicyConfig 策略配置
type PolicyConfig struct {
	ErrorRateThreshold   float64       `yaml:"error_rate_threshold"`
	GrowthRateThreshold  float64       `yaml:"growth_rate_threshold"`
	EvaluationInterval   time.Duration `yaml:"evaluation_interval"`
	PolicyExpireDuration time.Duration `yaml:"policy_expire_duration"`
}

// VectorDBConfig 向量数据库配置
type VectorDBConfig struct {
	PostgreSQL PostgreSQLConfig `yaml:"postgresql"`
	Redis      RedisConfig      `yaml:"redis"`
	CacheSize  int              `yaml:"cache_size"`
}

// PostgreSQLConfig PostgreSQL配置
type PostgreSQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}
