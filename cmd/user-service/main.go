package main

import (
	"flag"
	"fmt"

	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/config"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/logger"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/server"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/tracing"
	"google.golang.org/grpc"
)

var (
	configPath = flag.String("config", "configs/user-service.json", "配置文件路径")
)

func main() {
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// 初始化日志
	log, err := logger.NewLogger(cfg.Log.Level, cfg.Log.Format, cfg.Log.Output, cfg.Log.Path)
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}

	// 初始化链路追踪
	tracer, closer, err := tracing.InitTracer(
		cfg.Server.Name,
		cfg.Jaeger.Endpoint,
		cfg.Jaeger.Sampler,
	)
	if err != nil {
		log.Warnf("failed to init tracer: %v", err)
	} else {
		defer closer.Close()
	}
	_ = tracer

	// 启动统一的 gRPC 服务模板
	if err := server.RunGRPCServer(cfg, log, func(s *grpc.Server) error {
		// TODO: 在这里注册 user-service 的业务 gRPC 服务
		// pb.RegisterUserServiceServer(s, user.NewServer(...))
		return nil
	}); err != nil {
		log.Fatalf("user-service exited with error: %v", err)
	}
}
