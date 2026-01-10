# SmartLinkDrive 车联网平台

SmartLinkDrive 是一个专注于车联网 + 物联网 + 大数据解决方案的平台，聚焦网约车/物流车等车辆管理领域，支撑亿级用户同时在线的分布式系统。

## 项目结构

```
SmartLinkDrive/
├── cmd/                    # 各微服务入口
│   ├── user-service/
│   ├── vehicle-service/
│   ├── order-service/
│   └── ...
├── internal/               # 内部代码
│   ├── common/            # 公共组件库
│   │   ├── config/        # 配置管理
│   │   ├── logger/        # 日志组件
│   │   ├── discovery/     # 服务发现
│   │   ├── tracing/       # 链路追踪
│   │   ├── middleware/    # 中间件（熔断、限流等）
│   │   └── db/            # 数据库连接池
│   ├── api/               # API定义
│   │   └── proto/         # Protobuf定义文件
│   └── service/           # 业务服务实现
│       ├── user/
│       ├── vehicle/
│       └── ...
├── pkg/                    # 可复用的公共包
│   ├── cache/             # 缓存封装
│   ├── mq/                # 消息队列封装
│   └── utils/             # 工具函数
├── deployments/           # 部署相关文件
│   ├── docker-compose.yml # Docker Compose配置
│   └── configs/           # 配置文件
├── scripts/               # 脚本文件
├── docs/                  # 文档
├── go.mod
└── go.sum
```

## 快速开始

### 环境要求

- Go 1.21+
- Docker & Docker Compose
- Make (可选)

### 开发环境搭建

1. 克隆项目
```bash
git clone https://github.com/SmartLinkDrive/SmartLinkDrive.git
cd SmartLinkDrive
```

2. 启动基础设施服务
```bash
docker-compose -f deployments/docker-compose.yml up -d
```

3. 安装依赖
```bash
go mod download
```

4. 运行服务
```bash
# 运行用户服务
go run cmd/user-service/main.go

# 运行车辆服务
go run cmd/vehicle-service/main.go
```

详细文档请参考：
- [开发环境文档](./docs/开发环境文档.md)
- [部署文档](./docs/部署文档.md)

## 技术栈

- **语言**: Go 1.21+
- **微服务**: gRPC
- **服务发现**: Consul
- **数据库**: MySQL 8.0, ClickHouse
- **缓存**: Redis
- **消息队列**: Kafka
- **监控**: Prometheus + Grafana
- **链路追踪**: Jaeger
- **日志**: ELK Stack

## 许可证

MIT License
