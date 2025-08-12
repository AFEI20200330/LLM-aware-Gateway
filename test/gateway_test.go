package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/llm-aware-gateway/pkg/gateway"
	"github.com/llm-aware-gateway/pkg/types"
)

func TestGatewayBasicFunctionality(t *testing.T) {
	// 设置测试环境
	gin.SetMode(gin.TestMode)

	// 创建测试配置
	config := &types.GatewayConfig{
		Server: types.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Limiter: types.LimiterConfig{
			DefaultRate:     1000.0,
			MaxRate:         10000.0,
			CleanupInterval: 5 * time.Minute,
		},
		Breaker: types.BreakerConfig{
			FailureThreshold:  10,
			RecoveryTimeout:   30 * time.Second,
			RecoveryIncrement: 0.2,
		},
		Sampler: types.SamplerConfig{
			SamplingRate: 0.05,
			BufferSize:   1000,
		},
		Kafka: types.KafkaConfig{
			Brokers: []string{"localhost:9092"},
			Topic:   "error-events",
		},
		ETCD: types.ETCDConfig{
			Endpoints: []string{"localhost:2379"},
			Timeout:   5 * time.Second,
		},
		Metrics: types.MetricsConfig{
			Enabled: true,
			Port:    9090,
			Path:    "/metrics",
		},
	}

	// 创建网关实例
	gw, err := gateway.NewGateway(config)
	require.NoError(t, err)

	// 获取路由器
	router := gw.GetRouter() // 需要添加这个方法

	t.Run("健康检查", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])
	})

	t.Run("就绪检查", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/ready", nil)
		router.ServeHTTP(w, req)

		// 由于测试环境可能没有连接外部依赖，状态可能不是ready
		assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["status"])
	})

	t.Run("API代理请求", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Request processed successfully", response["message"])
		assert.Equal(t, "test", response["service"])
	})

	t.Run("模拟错误请求", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test?simulate_error=true", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Simulated error for testing", response["error"])
	})

	t.Run("CORS处理", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("OPTIONS", "/api/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("ID生成", func(t *testing.T) {
		// 这里需要引入utils包
		// id := utils.GenerateID()
		// assert.NotEmpty(t, id)
		// assert.Len(t, id, 32) // hex编码的16字节
	})

	t.Run("向量相似度计算", func(t *testing.T) {
		// vec1 := []float32{1.0, 0.0, 0.0}
		// vec2 := []float32{1.0, 0.0, 0.0}
		// similarity := utils.CosineSimilarity(vec1, vec2)
		// assert.Equal(t, 1.0, similarity)
	})
}

// BenchmarkGatewayThroughput 网关吞吐量基准测试
func BenchmarkGatewayThroughput(b *testing.B) {
	gin.SetMode(gin.TestMode)

	config := &types.GatewayConfig{
		Server: types.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Limiter: types.LimiterConfig{
			DefaultRate: 100000.0, // 高限流阈值用于测试
		},
	}

	gw, err := gateway.NewGateway(config)
	require.NoError(b, err)

	router := gw.GetRouter()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/benchmark", nil)
			router.ServeHTTP(w, req)
		}
	})
}

// TestRateLimiter 限流器测试
func TestRateLimiter(t *testing.T) {
	t.Skip("需要完整的环境才能测试")

	// 这里可以添加更复杂的限流测试
}

// TestCircuitBreaker 熔断器测试
func TestCircuitBreaker(t *testing.T) {
	t.Skip("需要完整的环境才能测试")

	// 这里可以添加熔断器测试
}
