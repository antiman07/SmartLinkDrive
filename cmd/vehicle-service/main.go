package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/config"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/discovery"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/logger"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/tracing"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var (
	configPath = flag.String("config", "configs/vehicle-service.json", "配置文件路径")
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

	// 初始化Consul客户端
	consulClient, err := discovery.NewConsulClient(cfg.Consul.Host, cfg.Consul.Port)
	if err != nil {
		log.Warnf("failed to connect to Consul: %v", err)
	}

	// 启动gRPC服务器
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.GRPCPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 创建gRPC服务器
	s := grpc.NewServer()

	// 注册健康检查服务
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// 注册反射服务（用于gRPC调试）
	reflection.Register(s)

	// 注册服务到Consul
	if consulClient != nil {
		serviceID := fmt.Sprintf("%s-%s", cfg.Server.Name, uuid.New().String())
		registry := discovery.NewServiceRegistry(
			consulClient,
			serviceID,
			cfg.Server.Name,
			cfg.Server.Host,
			cfg.Server.GRPCPort,
			[]string{"grpc"},
		)

		if err := registry.Register(); err != nil {
			log.Warnf("failed to register service to Consul: %v", err)
		} else {
			log.Infof("Service registered to Consul: %s", serviceID)
			defer registry.Deregister()
		}
	}

	log.Infof("Vehicle service starting on %s:%d", cfg.Server.Host, cfg.Server.GRPCPort)

	// 启动服务器
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		log.Warn("Server shutdown timeout, forcing stop...")
		s.Stop()
	case <-stopped:
		log.Info("Server stopped gracefully")
	}
}
