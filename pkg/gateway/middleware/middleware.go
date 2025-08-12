package middleware

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/llm-aware-gateway/pkg/interfaces"
	"github.com/llm-aware-gateway/pkg/utils"
)

// Middleware 中间件管理器
type Middleware struct {
	rateLimiter    interfaces.RateLimiter
	circuitBreaker interfaces.CircuitBreaker
	errorSampler   interfaces.ErrorSampler
	vectorAgent    interfaces.VectorAgent
	metrics        interfaces.MetricsCollector
}

// NewMiddleware 创建中间件管理器
func NewMiddleware(
	rateLimiter interfaces.RateLimiter,
	circuitBreaker interfaces.CircuitBreaker,
	errorSampler interfaces.ErrorSampler,
	vectorAgent interfaces.VectorAgent,
	metrics interfaces.MetricsCollector,
) *Middleware {
	return &Middleware{
		rateLimiter:    rateLimiter,
		circuitBreaker: circuitBreaker,
		errorSampler:   errorSampler,
		vectorAgent:    vectorAgent,
		metrics:        metrics,
	}
}

// Recovery 恢复中间件
func (m *Middleware) Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.String(http.StatusInternalServerError, fmt.Sprintf("error: %s", err))
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

// Logger 日志中间件
func (m *Middleware) Logger() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
				param.ClientIP,
				param.TimeStamp.Format(time.RFC1123),
				param.Method,
				param.Path,
				param.Request.Proto,
				param.StatusCode,
				param.Latency,
				param.Request.UserAgent(),
				param.ErrorMessage,
			)
		},
	})
}

// Tracing 链路追踪中间件
func (m *Middleware) Tracing() gin.HandlerFunc {
	return otelgin.Middleware("llm-aware-gateway")
}

// Authentication 认证中间件
func (m *Middleware) Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现JWT/OIDC认证逻辑
		// 这里暂时跳过认证
		c.Next()
	}
}

// RateLimit 限流中间件
func (m *Middleware) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.rateLimiter == nil {
			c.Next()
			return
		}

		// 检查是否允许请求
		if !m.rateLimiter.Allow(c) {
			// 记录限流指标
			clusterID := utils.ExtractServiceName(c)
			if m.metrics != nil {
				m.metrics.RecordRateLimitHit(clusterID, "RATE_LIMIT")
			}

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CircuitBreaker 熔断中间件
func (m *Middleware) CircuitBreaker() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.circuitBreaker == nil {
			c.Next()
			return
		}

		// 尝试识别簇ID
		clusterID := ""
		if m.vectorAgent != nil {
			errorSignature := utils.ExtractErrorSignature(c)
			if errorSignature != "" {
				if id, err := m.vectorAgent.IdentifyCluster(errorSignature); err == nil {
					clusterID = id
				}
			}
		}

		// 检查熔断器状态
		if !m.circuitBreaker.Allow(c.Request.Context(), clusterID) {
			// 记录熔断指标
			if m.metrics != nil {
				m.metrics.RecordCircuitBreakerState(clusterID, 1) // 1 = OPEN
			}

			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service temporarily unavailable",
				"code":  "CIRCUIT_BREAKER_OPEN",
			})
			c.Abort()
			return
		}

		// 保存簇ID到上下文，供后续中间件使用
		c.Set("cluster_id", clusterID)

		// 执行请求
		c.Next()

		// 根据请求结果记录成功或失败
		if c.Writer.Status() >= 500 {
			m.circuitBreaker.RecordFailure(clusterID)
		} else {
			m.circuitBreaker.RecordSuccess(clusterID)
		}
	}
}

// ErrorSampling 错误采样中间件
func (m *Middleware) ErrorSampling() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 检查是否有错误
		if len(c.Errors) > 0 || c.Writer.Status() >= 400 {
			if m.errorSampler != nil {
				// 构造错误
				var err error
				if len(c.Errors) > 0 {
					err = c.Errors.Last()
				} else {
					err = errors.New(http.StatusText(c.Writer.Status()))
				}

				// 将错误信息保存到上下文，供工具函数提取
				c.Set("error", err)

				// 采样错误
				if sampErr := m.errorSampler.SampleError(c, err); sampErr != nil {
					log.Printf("Failed to sample error: %v", sampErr)
				}
			}
		}
	}
}

// Metrics 指标收集中间件
func (m *Middleware) Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		// 记录请求指标
		if m.metrics != nil {
			duration := time.Since(start).Seconds()
			clusterID, _ := c.Get("cluster_id")
			clusterIDStr := ""
			if id, ok := clusterID.(string); ok {
				clusterIDStr = id
			}

			status := fmt.Sprintf("%d", c.Writer.Status())
			m.metrics.RecordRequest(c.Request.Method, c.Request.URL.Path, status, clusterIDStr, duration)
		}
	}
}

// CORS 跨域中间件
func (m *Middleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// HealthCheck 健康检查中间件
func (m *Middleware) HealthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
			return
		}

		if c.Request.URL.Path == "/ready" {
			// 检查各组件是否就绪
			ready := true
			components := make(map[string]bool)

			// 检查限流器
			if m.rateLimiter != nil {
				components["rate_limiter"] = true
			} else {
				components["rate_limiter"] = false
				ready = false
			}

			// 检查熔断器
			if m.circuitBreaker != nil {
				components["circuit_breaker"] = true
			} else {
				components["circuit_breaker"] = false
				ready = false
			}

			if ready {
				c.JSON(http.StatusOK, gin.H{
					"status":     "ready",
					"components": components,
					"timestamp":  time.Now().Unix(),
				})
			} else {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status":     "not_ready",
					"components": components,
					"timestamp":  time.Now().Unix(),
				})
			}
			c.Abort()
			return
		}

		c.Next()
	}
}
