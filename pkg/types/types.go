package types

import (
	"time"
)

// ErrorEvent 错误事件结构
type ErrorEvent struct {
	TraceID      string    `json:"trace_id"`
	SpanID       string    `json:"span_id"`
	RequestPath  string    `json:"request_path"`
	Method       string    `json:"method"`
	ServiceName  string    `json:"service_name"`
	StatusCode   int       `json:"status_code"`
	ErrorMessage string    `json:"error_message"`
	StackTrace   []string  `json:"stack_trace"`
	Timestamp    time.Time `json:"timestamp"`
	EventID      string    `json:"event_id"`
}

// Cluster 错误簇结构
type Cluster struct {
	ID          string      `json:"id"`
	Centroid    []float32   `json:"centroid"`
	Members     []string    `json:"members"`
	ErrorCount  int64       `json:"error_count"`
	CreateTime  time.Time   `json:"create_time"`
	UpdateTime  time.Time   `json:"update_time"`
	Severity    float64     `json:"severity"`
	Description string      `json:"description"`
}

// PolicyType 策略类型
type PolicyType string

const (
	RATE_LIMIT     PolicyType = "rate_limit"
	CIRCUIT_BREAK  PolicyType = "circuit_break"
	DEGRADE        PolicyType = "degrade"
)

// Policy 策略结构
type Policy struct {
	ClusterID     string              `json:"cluster_id"`
	PolicyType    PolicyType          `json:"policy_type"`
	Severity      float64             `json:"severity"`
	RateLimit     *RateLimitPolicy    `json:"rate_limit,omitempty"`
	CircuitBreak  *CircuitBreakPolicy `json:"circuit_break,omitempty"`
	CreateTime    time.Time           `json:"create_time"`
	ExpireTime    time.Time           `json:"expire_time"`
	IsActive      bool                `json:"is_active"`
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
	CLOSED    BreakerState = 0
	OPEN      BreakerState = 1
	HALF_OPEN BreakerState = 2
)

// BreakerConfig 熔断器配置
type BreakerConfig struct {
	FailureThreshold  int64         `json:"failure_threshold"`  // 失败次数阈值
	RecoveryTimeout   time.Duration `json:"recovery_timeout"`   // 恢复超时时间
	RecoveryIncrement float64       `json:"recovery_increment"` // 恢复增量 (20%)
}

// SearchResult 搜索结果
type SearchResult struct {
	ID         string  `json:"id"`
	Similarity float64 `json:"similarity"`
	Vector     []float32 `json:"vector,omitempty"`
}

// GatewayConfig 网关配置
type GatewayConfig struct {
	Server       ServerConfig       `yaml:"server"`
	RateLimit    RateLimitConfig    `yaml:"rate_limit"`
	CircuitBreak CircuitBreakConfig `yaml:"circuit_break"`
	ErrorSampler ErrorSamplerConfig `yaml:"error_sampler"`
	Kafka        KafkaConfig        `yaml:"kafka"`
	ETCD         ETCDConfig         `yaml:"etcd"`
	Redis        RedisConfig        `yaml:"redis"`
	Monitoring   MonitoringConfig   `yaml:"monitoring"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	DefaultQPS    float64 `yaml:"default_qps"`
	MaxQPS        float64 `yaml:"max_qps"`
	BucketSize    int64   `yaml:"bucket_size"`
	WindowSize    time.Duration `yaml:"window_size"`
}

// CircuitBreakConfig 熔断配置
type CircuitBreakConfig struct {
	FailureThreshold int64         `yaml:"failure_threshold"`
	RecoveryTimeout  time.Duration `yaml:"recovery_timeout"`
	HalfOpenMaxCalls int64         `yaml:"half_open_max_calls"`
}

// ErrorSamplerConfig 错误采样配置
type ErrorSamplerConfig struct {
	SamplingRate float64 `yaml:"sampling_rate"`
	MaxQueueSize int     `yaml:"max_queue_size"`
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`
	GroupID string   `yaml:"group_id"`
}

// ETCDConfig ETCD配置
type ETCDConfig struct {
	Endpoints   []string      `yaml:"endpoints"`
	DialTimeout time.Duration `yaml:"dial_timeout"`
	Username    string        `yaml:"username"`
	Password    string        `yaml:"password"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addresses []string      `yaml:"addresses"`
	Password  string        `yaml:"password"`
	DB        int           `yaml:"db"`
	PoolSize  int           `yaml:"pool_size"`
	Timeout   time.Duration `yaml:"timeout"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	MetricsPath string `yaml:"metrics_path"`
	EnableTrace bool   `yaml:"enable_trace"`
}

// ControlPlaneConfig 控制面配置
type ControlPlaneConfig struct {
	Embedding EmbeddingConfig `yaml:"embedding"`
	Clustering ClusteringConfig `yaml:"clustering"`
	VectorDB  VectorDBConfig  `yaml:"vector_db"`
	Policy    PolicyConfig    `yaml:"policy"`
	Kafka     KafkaConfig     `yaml:"kafka"`
	ETCD      ETCDConfig      `yaml:"etcd"`
	Storage   StorageConfig   `yaml:"storage"`
}

// EmbeddingConfig 向量化配置
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
	MinClusterSize       int           `yaml:"min_cluster_size"`
	MaxClusters          int           `yaml:"max_clusters"`
}

// VectorDBConfig 向量数据库配置
type VectorDBConfig struct {
	IndexType    string `yaml:"index_type"` // "faiss" or "pgvector"
	CacheSize    int    `yaml:"cache_size"`
	IndexParams  map[string]interface{} `yaml:"index_params"`
}

// PolicyConfig 策略配置
type PolicyConfig struct {
	ErrorRateThreshold  float64       `yaml:"error_rate_threshold"`
	GrowthRateThreshold float64       `yaml:"growth_rate_threshold"`
	WindowSize          time.Duration `yaml:"window_size"`
	PolicyTTL           time.Duration `yaml:"policy_ttl"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	PostgreSQL PostgreSQLConfig `yaml:"postgresql"`
	Redis      RedisConfig      `yaml:"redis"`
}

// PostgreSQLConfig PostgreSQL配置
type PostgreSQLConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	Database     string        `yaml:"database"`
	Username     string        `yaml:"username"`
	Password     string        `yaml:"password"`
	MaxOpenConns int           `yaml:"max_open_conns"`
	MaxIdleConns int           `yaml:"max_idle_conns"`
	ConnTimeout  time.Duration `yaml:"conn_timeout"`
}
