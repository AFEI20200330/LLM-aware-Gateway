package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"

	"github.com/llm-aware-gateway/pkg/gateway"
	"github.com/llm-aware-gateway/pkg/types"
)

func main() {
	// 命令行参数
	configFile := flag.String("config", "configs/gateway.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建网关实例
	gw, err := gateway.NewGateway(config)
	if err != nil {
		log.Fatalf("Failed to create gateway: %v", err)
	}

	// 启动网关
	if err := gw.Start(); err != nil {
		log.Fatalf("Failed to start gateway: %v", err)
	}

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gateway...")

	// 优雅停止网关
	if err := gw.Stop(); err != nil {
		log.Printf("Error stopping gateway: %v", err)
	}

	log.Println("Gateway stopped.")
}

// loadConfig 加载配置文件
func loadConfig(configFile string) (*types.GatewayConfig, error) {
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")

	// 设置默认值
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: Failed to read config file %s: %v", configFile, err)
		log.Println("Using default configuration")
	}

	var config types.GatewayConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setDefaults 设置默认配置
func setDefaults() {
	// 服务器配置
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)

	// 限流器配置
	viper.SetDefault("limiter.default_rate", 1000.0)
	viper.SetDefault("limiter.max_rate", 10000.0)
	viper.SetDefault("limiter.cleanup_interval", "5m")

	// 熔断器配置
	viper.SetDefault("breaker.failure_threshold", 10)
	viper.SetDefault("breaker.recovery_timeout", "30s")
	viper.SetDefault("breaker.recovery_increment", 0.2)

	// 采样器配置
	viper.SetDefault("sampler.sampling_rate", 0.05)
	viper.SetDefault("sampler.buffer_size", 1000)

	// Kafka配置
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.topic", "error-events")

	// ETCD配置
	viper.SetDefault("etcd.endpoints", []string{"localhost:2379"})
	viper.SetDefault("etcd.timeout", "5s")

	// 监控配置
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.port", 9090)
	viper.SetDefault("metrics.path", "/metrics")
}
