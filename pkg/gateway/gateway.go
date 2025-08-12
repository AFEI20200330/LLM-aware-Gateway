package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/llm-aware-gateway/pkg/gateway/breaker"
	"github.com/llm-aware-gateway/pkg/gateway/config"
	"github.com/llm-aware-gateway/pkg/gateway/limiter"
	"github.com/llm-aware-gateway/pkg/gateway/middleware"
	"github.com/llm-aware-gateway/pkg/gateway/sampler"
	"github.com/llm-aware-gateway/pkg/gateway/vector"
	"github.com/llm-aware-gateway/pkg/interfaces"
	"github.com/llm-aware-gateway/pkg/types"
	"github.com/llm-aware-gateway/pkg/utils"
)

// Gateway 网关服务
type Gateway struct {
	config         *types.GatewayConfig
	router         *gin.Engine
	server         *http.Server
	rateLimiter    interfaces.RateLimiter
	circuitBreaker interfaces.CircuitBreaker
	errorSampler   interfaces.ErrorSampler
	vectorAgent    interfaces.VectorAgent
	configWatcher  interfaces.ConfigWatcher
	metrics        interfaces.MetricsCollector
	middleware     *middleware.Middleware
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

// NewGateway 创建网关实例
func NewGateway(config *types.GatewayConfig) (*Gateway, error) {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// 创建缓存
	cache := utils.NewCache(10000)

	// 创建向量代理 (暂时不连接嵌入服务)
	vectorAgent := vector.NewVectorAgent(nil, cache)

	// 创建限流器
	rateLimiter := limiter.NewClusterRateLimiter(&config.Limiter, vectorAgent)

	// 创建熔断器
	circuitBreaker := breaker.NewClusterCircuitBreaker(&config.Breaker)

	// 创建错误采样器
	errorSampler := sampler.NewErrorSampler(&config.Sampler, &config.Kafka)

	// 创建配置监听器
	configWatcher, err := config.NewConfigWatcher(&config.ETCD)
	if err != nil {
		return nil, fmt.Errorf("failed to create config watcher: %v", err)
	}

	// 创建指标收集器
	metricsCollector := NewMetricsCollector()

	// 创建中间件管理器
	middlewareManager := middleware.NewMiddleware(
		rateLimiter,
		circuitBreaker,
		errorSampler,
		vectorAgent,
		metricsCollector,
	)

	gateway := &Gateway{
		config:         config,
		router:         router,
		rateLimiter:    rateLimiter,
		circuitBreaker: circuitBreaker,
		errorSampler:   errorSampler,
		vectorAgent:    vectorAgent,
		configWatcher:  configWatcher,
		metrics:        metricsCollector,
		middleware:     middlewareManager,
		stopCh:         make(chan struct{}),
	}

	// 设置中间件
	gateway.setupMiddleware()

	// 设置路由
	gateway.setupRoutes()

	return gateway, nil
}

// setupMiddleware 设置中间件
func (g *Gateway) setupMiddleware() {
	g.router.Use(
		g.middleware.Recovery(),
		g.middleware.Logger(),
		g.middleware.Tracing(),
		g.middleware.CORS(),
		g.middleware.HealthCheck(),
		g.middleware.Authentication(),
		g.middleware.RateLimit(),
		g.middleware.CircuitBreaker(),
		g.middleware.ErrorSampling(),
		g.middleware.Metrics(),
	)
}

// setupRoutes 设置路由
func (g *Gateway) setupRoutes() {
	// 健康检查路由已在中间件中处理

	// API代理路由
	api := g.router.Group("/api")
	{
		// 通用代理处理器
		api.Any("/*path", g.proxyHandler)
	}

	// 管理API路由
	admin := g.router.Group("/admin")
	{
		admin.GET("/stats", g.getStatsHandler)
		admin.GET("/clusters", g.getClustersHandler)
		admin.GET("/policies", g.getPoliciesHandler)
	}

	// 指标路由
	if g.config.Metrics.Enabled {
		g.router.GET("/metrics", g.metricsHandler)
	}
}

// Start 启动网关服务
func (g *Gateway) Start() error {
	// 启动错误采样器
	if err := g.errorSampler.Start(); err != nil {
		return fmt.Errorf("failed to start error sampler: %v", err)
	}

	// 启动配置监听器
	if err := g.configWatcher.Start(); err != nil {
		return fmt.Errorf("failed to start config watcher: %v", err)
	}

	// 注册策略更新回调
	g.configWatcher.RegisterCallback(g)

	// 创建HTTP服务器
	g.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", g.config.Server.Host, g.config.Server.Port),
		Handler: g.router,
	}

	// 启动HTTP服务器
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		log.Printf("Starting gateway server on %s", g.server.Addr)
		if err := g.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Failed to start server: %v", err)
		}
	}()

	log.Println("Gateway started successfully")
	return nil
}

// Stop 停止网关服务
func (g *Gateway) Stop() error {
	log.Println("Stopping gateway...")

	// 关闭停止信号
	close(g.stopCh)

	// 停止HTTP服务器
	if g.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := g.server.Shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown server gracefully: %v", err)
		}
	}

	// 停止各个组件
	if g.errorSampler != nil {
		g.errorSampler.Stop()
	}

	if g.configWatcher != nil {
		g.configWatcher.Stop()
	}

	if g.rateLimiter != nil {
		g.rateLimiter.Cleanup()
	}

	// 等待所有goroutine结束
	g.wg.Wait()

	log.Println("Gateway stopped")
	return nil
}

// OnPolicyUpdate 策略更新回调
func (g *Gateway) OnPolicyUpdate(clusterID string, policy *types.Policy) error {
	log.Printf("Received policy update for cluster: %s", clusterID)

	// 更新限流器策略
	if err := g.rateLimiter.UpdatePolicy(clusterID, policy); err != nil {
		log.Printf("Failed to update rate limiter policy: %v", err)
	}

	// 更新熔断器策略
	if err := g.circuitBreaker.UpdatePolicy(clusterID, policy); err != nil {
		log.Printf("Failed to update circuit breaker policy: %v", err)
	}

	return nil
}

// OnPolicyDelete 策略删除回调
func (g *Gateway) OnPolicyDelete(clusterID string) error {
	log.Printf("Received policy delete for cluster: %s", clusterID)
	// 这里可以实现策略删除逻辑
	return nil
}

// proxyHandler 代理处理器
func (g *Gateway) proxyHandler(c *gin.Context) {
	// 这里应该实现到下游服务的代理逻辑
	// 为了演示，我们返回一个简单的响应

	// 模拟服务响应
	service := utils.ExtractServiceName(c)

	// 模拟一些错误情况用于测试
	if c.Query("simulate_error") == "true" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Simulated error for testing",
			"service": service,
			"path": c.Request.URL.Path,
		})
		return
	}

	// 正常响应
	c.JSON(http.StatusOK, gin.H{
		"message": "Request processed successfully",
		"service": service,
		"path":    c.Request.URL.Path,
		"method":  c.Request.Method,
		"timestamp": time.Now().Unix(),
	})
}

// getStatsHandler 获取统计信息
func (g *Gateway) getStatsHandler(c *gin.Context) {
	clusterID := c.Query("cluster_id")
	if clusterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "cluster_id parameter is required",
		})
		return
	}

	stats, err := g.rateLimiter.GetStats(clusterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("No stats found for cluster: %s", clusterID),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster_id": clusterID,
		"stats": stats,
		"breaker_state": g.circuitBreaker.GetState(clusterID),
	})
}

// getClustersHandler 获取簇信息
func (g *Gateway) getClustersHandler(c *gin.Context) {
	// 这里应该从向量代理获取簇信息
	c.JSON(http.StatusOK, gin.H{
		"clusters": []string{}, // 简化实现
		"count": 0,
	})
}

// getPoliciesHandler 获取策略信息
func (g *Gateway) getPoliciesHandler(c *gin.Context) {
	clusterID := c.Query("cluster_id")
	if clusterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "cluster_id parameter is required",
		})
		return
	}

	policy, err := g.configWatcher.GetPolicy(clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get policy: %v", err),
		})
		return
	}

	if policy == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("No policy found for cluster: %s", clusterID),
		})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// metricsHandler 指标处理器
func (g *Gateway) metricsHandler(c *gin.Context) {
	// 这里应该返回Prometheus格式的指标
	c.String(http.StatusOK, "# Metrics endpoint placeholder\n")
}

// GetRouter 获取路由器（用于测试）
func (g *Gateway) GetRouter() *gin.Engine {
	return g.router
}
