# LLM-Aware Gateway

> 语义感知的熔断/限流网关 - 基于大语言模型的智能错误聚类与精准流控

## 📋 项目概述

LLM-Aware Gateway是一个基于语义分析的智能网关系统，通过向量化和聚类技术识别同类错误，实现精确熔断和限流，避免传统粗粒度策略的误杀问题。

### 🌟 核心特性

- **🧠 语义感知**: 使用BGE嵌入模型理解错误语义，实现智能错误分类
- **🔗 精准聚类**: 基于向量相似度的增量聚类算法，动态识别错误模式
- **⚡ 智能限流**: 基于错误簇的精确限流，避免全局误杀
- **🔒 智能熔断**: 针对特定错误类型的精准熔断策略
- **📊 实时监控**: 完整的Prometheus + Grafana监控体系
- **🚀 高性能**: 毫秒级响应，支持万级并发
- **🔧 易扩展**: 模块化架构，支持插件式扩展

## 🏗️ 架构设计

### 整体架构

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   客户端请求     │───▶│   数据面(Gateway)  │───▶│   下游服务       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │   错误采样器      │
                       └──────────────────┘
                                │
                                ▼ (Kafka)
                       ┌──────────────────┐
                       │  控制面(分析引擎)  │
                       └──────────────────┘
                                │
                                ▼ (ETCD)
                       ┌──────────────────┐
                       │    策略存储       │
                       └──────────────────┘
```

### 核心组件

#### 数据面 (Data Plane)
- **限流器**: 基于令牌桶的精准限流
- **熔断器**: 多状态智能熔断器
- **错误采样器**: 高效的错误事件采样
- **向量代理**: 连接数据面与控制面的桥梁

#### 控制面 (Control Plane)
- **嵌入服务**: BGE模型文本向量化
- **聚类引擎**: 增量K-means错误聚类
- **策略引擎**: 基于规则的策略生成
- **配置存储**: 分布式配置管理

## 🚀 快速开始

### 环境要求

- Go 1.21+
- Docker & Docker Compose
- Make工具

### 一键启动

```bash
# 克隆项目
git clone https://github.com/your-org/llm-aware-gateway.git
cd llm-aware-gateway

# 启动完整环境
make docker-compose-up

# 等待服务启动完成，然后访问:
# - 网关服务: http://localhost:8080
# - Grafana监控: http://localhost:3000 (admin/admin123)
# - Prometheus: http://localhost:9090
```

### 本地开发

```bash
# 安装依赖
make deps

# 构建项目
make build-local

# 运行服务
make run

# 运行测试
make test
```

## 📖 使用指南

### 基本请求

```bash
# 健康检查
curl http://localhost:8080/health

# 就绪检查
curl http://localhost:8080/ready

# API请求
curl http://localhost:8080/api/your-service/endpoint

# 模拟错误（测试用）
curl http://localhost:8080/api/test?simulate_error=true
```

### 管理API

```bash
# 查看簇统计
curl "http://localhost:8080/admin/stats?cluster_id=cluster_xxx"

# 查看所有簇
curl http://localhost:8080/admin/clusters

# 查看策略配置
curl "http://localhost:8080/admin/policies?cluster_id=cluster_xxx"
```

### 配置说明

网关配置文件 `configs/gateway.yaml`:

```yaml
# 服务器配置
server:
  host: "0.0.0.0"
  port: 8080

# 限流器配置
limiter:
  default_rate: 1000.0      # 默认限流速率(req/s)
  max_rate: 10000.0         # 最大限流速率
  cleanup_interval: "5m"    # 清理间隔

# 熔断器配置
breaker:
  failure_threshold: 10     # 失败阈值
  recovery_timeout: "30s"   # 恢复超时
  recovery_increment: 0.2   # 恢复步长

# 错误采样配置
sampler:
  sampling_rate: 0.05       # 采样率(5%)
  buffer_size: 1000         # 缓冲区大小

# 消息队列配置
kafka:
  brokers: ["localhost:9092"]
  topic: "error-events"

# 配置存储
etcd:
  endpoints: ["localhost:2379"]
  timeout: "5s"

# 监控配置
metrics:
  enabled: true
  port: 9090
  path: "/metrics"
```

## 🔧 开发指南

### 项目结构

```
.
├── cmd/                    # 应用入口
│   └── gateway/           # 网关服务入口
├── pkg/                   # 核心包
│   ├── interfaces/        # 接口定义
│   ├── types/             # 类型定义
│   ├── gateway/           # 数据面实现
│   ├── controlplane/      # 控制面实现
│   └── utils/             # 工具函数
├── configs/               # 配置文件
├── test/                  # 测试文件
├── monitoring/            # 监控配置
├── docker-compose.yml     # 环境编排
├── Dockerfile             # 容器化
└── Makefile              # 构建脚本
```

### 核心接口

#### 限流器接口
```go
type RateLimiter interface {
    Allow(ctx *gin.Context) bool
    UpdatePolicy(clusterID string, policy *Policy) error
    GetStats(clusterID string) (*ClusterStats, error)
    Cleanup() error
}
```

#### 熔断器接口
```go
type CircuitBreaker interface {
    Allow(ctx context.Context, clusterID string) bool
    RecordSuccess(clusterID string) error
    RecordFailure(clusterID string) error
    GetState(clusterID string) BreakerState
}
```

#### 聚类引擎接口
```go
type ClusteringEngine interface {
    ProcessErrorEvent(event *ErrorEvent) error
    GetCluster(clusterID string) (*Cluster, error)
    GetAllClusters() (map[string]*Cluster, error)
    ReCluster() error
}
```

### 扩展开发

#### 添加新的限流策略

1. 实现 `RateLimiter` 接口
2. 在 `limiter` 包中添加实现
3. 在网关中注册新的限流器

#### 添加新的嵌入模型

1. 实现 `EmbeddingService` 接口
2. 在 `embedding` 包中添加实现
3. 更新配置支持新模型

## 📊 监控与告警

### 关键指标

- `gateway_requests_total`: 总请求数
- `gateway_request_duration_seconds`: 请求延迟
- `gateway_rate_limit_hits_total`: 限流命中次数
- `gateway_circuit_breaker_state`: 熔断器状态
- `gateway_cluster_size`: 错误簇大小
- `gateway_cluster_severity`: 簇严重度

### Grafana仪表盘

系统提供预配置的Grafana仪表盘，包括:

- 📈 请求流量监控
- ⚡ 性能指标分析
- 🚨 错误率趋势
- 🔒 熔断/限流状态
- 🧠 聚类效果分析

### 告警规则

基于Prometheus的告警规则:

- 错误率异常告警
- 响应延迟异常告警
- 服务不可用告警
- 资源使用率告警

## 🧪 测试

### 运行测试

```bash
# 单元测试
make test

# 基准测试
make bench

# 代码覆盖率
make test
open coverage.html
```

### 性能测试

```bash
# 压力测试
wrk -t12 -c400 -d30s http://localhost:8080/api/test

# 错误注入测试
wrk -t12 -c400 -d30s http://localhost:8080/api/test?simulate_error=true
```

## 📦 部署

### Docker部署

```bash
# 构建镜像
make docker-build

# 运行容器
make docker-run
```

### Kubernetes部署

```yaml
# 示例K8s配置
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-aware-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: llm-aware-gateway
  template:
    metadata:
      labels:
        app: llm-aware-gateway
    spec:
      containers:
      - name: gateway
        image: llm-aware-gateway:latest
        ports:
        - containerPort: 8080
        env:
        - name: ETCD_ENDPOINTS
          value: "etcd-service:2379"
        - name: KAFKA_BROKERS
          value: "kafka-service:9092"
```

### 生产环境建议

1. **高可用部署**: 至少3个实例，使用负载均衡
2. **资源配置**: CPU 2核，内存 4GB起
3. **存储配置**: 持久化PostgreSQL和ETCD数据
4. **网络配置**: 配置适当的网络策略和防火墙规则
5. **监控告警**: 部署完整的监控告警体系

## 🤝 贡献指南

### 开发环境设置

```bash
# 设置开发环境
make dev-setup

# 代码格式化
make fmt

# 代码检查
make vet lint
```

### 提交规范

- 🐛 fix: 修复bug
- ✨ feat: 新功能
- 📝 docs: 文档更新
- 🎨 style: 代码格式
- ♻️ refactor: 重构
- ⚡ perf: 性能优化
- ✅ test: 测试相关

### Pull Request流程

1. Fork项目
2. 创建特性分支
3. 提交代码变更
4. 确保所有测试通过
5. 提交Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 🙏 致谢

感谢以下开源项目的支持:

- [Gin](https://github.com/gin-gonic/gin) - HTTP Web框架
- [ETCD](https://github.com/etcd-io/etcd) - 分布式配置存储
- [Kafka](https://kafka.apache.org/) - 消息队列
- [Prometheus](https://prometheus.io/) - 监控系统
- [Grafana](https://grafana.com/) - 可视化平台

## 📞 联系方式

- 项目负责人: [您的邮箱]
- 技术讨论: [GitHub Discussions](https://github.com/your-org/llm-aware-gateway/discussions)
- 问题反馈: [GitHub Issues](https://github.com/your-org/llm-aware-gateway/issues)

---

⭐ 如果这个项目对你有帮助，请给我们一个Star！
