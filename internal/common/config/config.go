package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
	Consul   ConsulConfig   `json:"consul"`
	Jaeger   JaegerConfig   `json:"jaeger"`
	Kafka    KafkaConfig    `json:"kafka"`
	Log      LogConfig      `json:"log"`
}

// ServerConfig 服务配置
type ServerConfig struct {
	Name     string `json:"name"`      // 服务名称
	Host     string `json:"host"`      // 服务地址
	Port     int    `json:"port"`      // 服务端口
	GRPCPort int    `json:"grpc_port"` // gRPC端口
	HTTPPort int    `json:"http_port"` // HTTP端口
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver   string `json:"driver"`   // 数据库驱动
	Host     string `json:"host"`     // 数据库地址
	Port     int    `json:"port"`     // 数据库端口
	User     string `json:"user"`     // 用户名
	Password string `json:"password"` // 密码
	Database string `json:"database"` // 数据库名
	MaxIdle  int    `json:"max_idle"` // 最大空闲连接
	MaxOpen  int    `json:"max_open"` // 最大打开连接
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	PoolSize int    `json:"pool_size"`
}

// ConsulConfig Consul配置
type ConsulConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// JaegerConfig Jaeger配置
type JaegerConfig struct {
	Endpoint string  `json:"endpoint"`
	Sampler  float64 `json:"sampler"` // 采样率 0.0-1.0
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers []string `json:"brokers"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `json:"level"`  // debug, info, warn, error
	Format string `json:"format"` // json, text
	Output string `json:"output"` // stdout, file
	Path   string `json:"path"`   // 日志文件路径
}

var (
	globalConfig *Config
	configOnce   sync.Once
)

// LoadConfig 加载配置
func LoadConfig(configPath string) (*Config, error) {
	var err error
	configOnce.Do(func() {
		globalConfig = &Config{}
		// 如果配置文件不存在，使用默认配置
		if _, err = os.Stat(configPath); os.IsNotExist(err) {
			logrus.Warnf("Config file not found: %s, using default config", configPath)
			globalConfig = defaultConfig()
			err = nil
			return
		}

		data, readErr := os.ReadFile(configPath)
		if readErr != nil {
			err = fmt.Errorf("failed to read config file: %w", readErr)
			return
		}

		if unmarshalErr := json.Unmarshal(data, globalConfig); unmarshalErr != nil {
			err = fmt.Errorf("failed to parse config file: %w", unmarshalErr)
			return
		}
	})

	if err != nil {
		return nil, err
	}

	return globalConfig, nil
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	if globalConfig == nil {
		return defaultConfig()
	}
	return globalConfig
}

// defaultConfig 默认配置（开发环境）
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Name:     "default-service",
			Host:     "0.0.0.0",
			Port:     8080,
			GRPCPort: 50051,
			HTTPPort: 8080,
		},
		Database: DatabaseConfig{
			Driver:   "mysql",
			Host:     "localhost",
			Port:     3306,
			User:     "root",
			Password: "root",
			Database: "smartlinkdrive",
			MaxIdle:  10,
			MaxOpen:  100,
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			PoolSize: 10,
		},
		Consul: ConsulConfig{
			Host: "localhost",
			Port: 8500,
		},
		Jaeger: JaegerConfig{
			Endpoint: "http://localhost:14268/api/traces",
			Sampler:  1.0,
		},
		Kafka: KafkaConfig{
			Brokers: []string{"localhost:9092"},
		},
		Log: LogConfig{
			Level:  "debug",
			Format: "text",
			Output: "stdout",
			Path:   "logs/app.log",
		},
	}
}
