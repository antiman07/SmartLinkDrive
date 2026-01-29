package main

import (
	"flag"
	"fmt"

	vehiclepb "github.com/SmartLinkDrive/SmartLinkDrive/internal/api/proto/vehicle"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/config"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/db"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/logger"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/server"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/tracing"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/vehicle"
	"google.golang.org/grpc"
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

	// 初始化数据库
	gormDB, err := db.NewMySQL(
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Database,
		cfg.Database.MaxIdle,
		cfg.Database.MaxOpen,
	)
	if err != nil {
		log.Fatalf("failed to init mysql: %v", err)
	}
	if err := gormDB.AutoMigrate(&vehicle.Vehicle{}); err != nil {
		log.Fatalf("failed to migrate mysql schema: %v", err)
	}

	// 启动统一的 gRPC 服务模板
	if err := server.RunGRPCServer(cfg, log, func(s *grpc.Server) error {
		vehiclepb.RegisterVehicleServiceServer(s, vehicle.NewGRPCServer(gormDB))
		return nil
	}); err != nil {
		log.Fatalf("vehicle-service exited with error: %v", err)
	}
}
