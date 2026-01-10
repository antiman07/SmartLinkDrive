.PHONY: help build run test clean docker-up docker-down proto

help: ## 显示帮助信息
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## 构建所有服务
	@echo "Building all services..."
	@go build -o bin/user-service ./cmd/user-service
	@go build -o bin/vehicle-service ./cmd/vehicle-service

run-user: ## 运行用户服务
	@go run ./cmd/user-service

run-vehicle: ## 运行车辆服务
	@go run ./cmd/vehicle-service

test: ## 运行测试
	@go test -v ./...

clean: ## 清理构建文件
	@rm -rf bin/
	@rm -rf logs/

docker-up: ## 启动Docker服务
	@docker-compose -f deployments/docker-compose.yml up -d

docker-down: ## 停止Docker服务
	@docker-compose -f deployments/docker-compose.yml down

docker-logs: ## 查看Docker日志
	@docker-compose -f deployments/docker-compose.yml logs -f

proto: ## 生成Protobuf代码
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
		internal/api/proto/**/*.proto

install-tools: ## 安装开发工具
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest

mod-download: ## 下载依赖
	@go mod download
	@go mod tidy

mod-vendor: ## 生成vendor目录
	@go mod vendor
