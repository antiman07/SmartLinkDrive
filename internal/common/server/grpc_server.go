package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/config"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/discovery"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/logger"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// GRPCRegisterFunc 用于注册业务 gRPC 服务（pb.RegisterXxxServer 等）。
type GRPCRegisterFunc func(s *grpc.Server) error

type RunGRPCOptions struct {
	EnableReflection bool
	ShutdownTimeout  time.Duration
}

func defaultRunGRPCOptions() RunGRPCOptions {
	return RunGRPCOptions{
		EnableReflection: true,
		ShutdownTimeout:  5 * time.Second,
	}
}

// RunGRPCServer 统一的 gRPC 服务启动模板：
// - 初始化 listener + grpc server（含拦截器）
// - 注册 health / reflection
// - 注册业务服务
// - 注册到 Consul（gRPC check）
// - 优雅退出
func RunGRPCServer(cfg *config.Config, log logger.Logger, register GRPCRegisterFunc, opts ...func(*RunGRPCOptions)) error {
	if cfg == nil {
		return fmt.Errorf("cfg is nil")
	}
	if log == nil {
		return fmt.Errorf("log is nil")
	}

	o := defaultRunGRPCOptions()
	for _, apply := range opts {
		if apply != nil {
			apply(&o)
		}
	}

	// 初始化 Consul 客户端（失败不阻塞服务启动）
	consulClient, err := discovery.NewConsulClient(cfg.Consul.Host, cfg.Consul.Port)
	if err != nil {
		log.Warnf("failed to connect to Consul: %v", err)
		consulClient = nil
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// 构建统一的 Unary 拦截器链（按顺序执行）
	unaryInterceptors := UnaryChain(
		UnaryRecoveryInterceptor(log),            // 异常恢复，避免服务崩溃
		UnaryTracingInterceptor(cfg.Server.Name), // 链路追踪
		UnaryAccessLogInterceptor(log),           // 访问日志
	)

	// 创建 gRPC Server，并注入拦截器
	s := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptors),
	)

	// gRPC 健康检查（供 Consul 的 GRPC check 探测）
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	if o.EnableReflection {
		reflection.Register(s)
	}

	if register != nil {
		if err := register(s); err != nil {
			return fmt.Errorf("failed to register grpc services: %w", err)
		}
	}

	// 注册到 Consul（成功才 defer 注销）
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
			defer func() {
				if err := registry.Deregister(); err != nil {
					log.Warnf("failed to deregister service from Consul: %v", err)
				}
			}()
		}
	}

	log.Infof("%s starting on %s:%d", cfg.Server.Name, cfg.Server.Host, cfg.Server.GRPCPort)

	serveErr := make(chan error, 1)
	go func() {
		if err := s.Serve(lis); err != nil {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Infof("received signal %v, shutting down...", sig)
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("grpc serve failed: %w", err)
		}
		return nil
	}

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), o.ShutdownTimeout)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		log.Warn("grpc shutdown timeout, forcing stop...")
		s.Stop()
	case <-stopped:
		log.Info("grpc server stopped gracefully")
	}

	return nil
}

// WithShutdownTimeout 修改优雅退出等待时间。
func WithShutdownTimeout(d time.Duration) func(*RunGRPCOptions) {
	return func(o *RunGRPCOptions) {
		if d > 0 {
			o.ShutdownTimeout = d
		}
	}
}

// WithReflection 控制是否启用 gRPC reflection。
func WithReflection(enable bool) func(*RunGRPCOptions) {
	return func(o *RunGRPCOptions) {
		o.EnableReflection = enable
	}
}
