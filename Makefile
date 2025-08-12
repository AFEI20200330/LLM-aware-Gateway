# LLM-Aware Gateway Makefile

# 项目信息
PROJECT_NAME := llm-aware-gateway
VERSION := 1.0.0
BUILD_TIME := $(shell date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT := $(shell git rev-parse --short HEAD)

# Go相关配置
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# 构建相关
BINARY_NAME := gateway
BINARY_PATH := ./bin/$(BINARY_NAME)
MAIN_PATH := ./cmd/gateway
LDFLAGS := -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)

# Docker相关
DOCKER_IMAGE := $(PROJECT_NAME)
DOCKER_TAG := $(VERSION)

.PHONY: help build test clean run deps docker-build docker-run docker-compose-up docker-compose-down

# 默认目标
.DEFAULT_GOAL := help

# 帮助信息
help: ## 显示帮助信息
	@echo "LLM-Aware Gateway - 语义感知的熔断/限流网关"
	@echo ""
	@echo "可用命令:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# 安装依赖
deps: ## 安装Go模块依赖
	$(GOMOD) download
	$(GOMOD) tidy

# 构建项目
build: deps ## 构建二进制文件
	@echo "构建 $(BINARY_NAME)..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "构建完成: $(BINARY_PATH)"

# 构建本地版本
build-local: deps ## 构建本地二进制文件
	@echo "构建本地版本 $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "构建完成: $(BINARY_PATH)"

# 运行项目
run: build-local ## 运行网关服务
	@echo "启动网关服务..."
	$(BINARY_PATH) -config=configs/gateway.yaml

# 运行测试
test: ## 运行单元测试
	@echo "运行测试..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "生成测试覆盖率报告..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# 运行基准测试
bench: ## 运行基准测试
	@echo "运行基准测试..."
	$(GOTEST) -bench=. -benchmem ./...

# 代码格式化
fmt: ## 格式化代码
	@echo "格式化代码..."
	$(GOCMD) fmt ./...

# 代码检查
vet: ## 运行go vet检查
	@echo "运行代码检查..."
	$(GOCMD) vet ./...

# 代码质量检查
lint: ## 运行golangci-lint检查
	@echo "运行代码质量检查..."
	golangci-lint run

# 清理构建文件
clean: ## 清理构建文件和缓存
	@echo "清理构建文件..."
	$(GOCLEAN)
	rm -rf bin/
	rm -f coverage.out coverage.html

# Docker构建
docker-build: ## 构建Docker镜像
	@echo "构建Docker镜像..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest
	@echo "Docker镜像构建完成: $(DOCKER_IMAGE):$(DOCKER_TAG)"

# Docker运行
docker-run: docker-build ## 运行Docker容器
	@echo "运行Docker容器..."
	docker run --rm -p 8080:8080 --name $(PROJECT_NAME) $(DOCKER_IMAGE):$(DOCKER_TAG)

# Docker Compose启动
docker-compose-up: ## 启动完整环境（Docker Compose）
	@echo "启动完整环境..."
	docker-compose up -d
	@echo "等待服务启动..."
	@sleep 10
	@echo "服务状态:"
	@docker-compose ps
	@echo ""
	@echo "访问地址:"
	@echo "  网关服务: http://localhost:8080"
	@echo "  Grafana: http://localhost:3000 (admin/admin123)"
	@echo "  Prometheus: http://localhost:9090"

# Docker Compose停止
docker-compose-down: ## 停止完整环境
	@echo "停止完整环境..."
	docker-compose down -v

# Docker Compose重启
docker-compose-restart: docker-compose-down docker-compose-up ## 重启完整环境

# 查看日志
logs: ## 查看网关服务日志
	docker-compose logs -f gateway

# 健康检查
health-check: ## 检查服务健康状态
	@echo "检查网关健康状态..."
	@curl -s http://localhost:8080/health | jq . || echo "网关服务不可用"
	@echo ""
	@curl -s http://localhost:8080/ready | jq . || echo "网关服务未就绪"

# 安装工具
install-tools: ## 安装开发工具
	@echo "安装开发工具..."
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint
	$(GOGET) -u github.com/swaggo/swag/cmd/swag

# 生成Go mod sum
mod-verify: ## 验证模块依赖
	$(GOMOD) verify

# 更新依赖
mod-update: ## 更新所有依赖到最新版本
	$(GOMOD) get -u all
	$(GOMOD) tidy

# 创建发布包
release: clean build ## 创建发布包
	@echo "创建发布包..."
	@mkdir -p release
	@tar -czf release/$(PROJECT_NAME)-$(VERSION)-linux-amd64.tar.gz -C bin $(BINARY_NAME) -C ../configs gateway.yaml
	@echo "发布包已创建: release/$(PROJECT_NAME)-$(VERSION)-linux-amd64.tar.gz"

# 显示项目信息
info: ## 显示项目信息
	@echo "项目信息:"
	@echo "  名称: $(PROJECT_NAME)"
	@echo "  版本: $(VERSION)"
	@echo "  构建时间: $(BUILD_TIME)"
	@echo "  Git提交: $(GIT_COMMIT)"
	@echo "  Go版本: $(shell $(GOCMD) version)"

# 快速开发环境设置
dev-setup: install-tools deps ## 设置开发环境
	@echo "开发环境设置完成"

# 完整的CI/CD流程
ci: deps fmt vet lint test build ## CI/CD流程
	@echo "CI/CD流程完成"
