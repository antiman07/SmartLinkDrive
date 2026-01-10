package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 日志接口
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	WithFields(fields map[string]interface{}) Logger
	WithField(key string, value interface{}) Logger
}

// LogrusLogger logrus实现
type LogrusLogger struct {
	logger *logrus.Logger
}

// NewLogrusLogger 创建LogrusLogger
func NewLogrusLogger(level, format, output, path string) (*LogrusLogger, error) {
	log := logrus.New()

	// 设置日志级别
	parseLevel, err := logrus.ParseLevel(level)
	if err != nil {
		parseLevel = logrus.DebugLevel
	}
	log.SetLevel(parseLevel)

	// 设置日志格式
	if format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// 设置日志输出
	var writer io.Writer
	if output == "file" {
		// 确保目录存在
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		writer = io.MultiWriter(os.Stdout, file)
	} else {
		writer = os.Stdout
	}
	log.SetOutput(writer)

	return &LogrusLogger{logger: log}, nil
}

func (l *LogrusLogger) Debug(args ...interface{}) {
	l.logger.Debug(args...)
}

func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *LogrusLogger) Info(args ...interface{}) {
	l.logger.Info(args...)
}

func (l *LogrusLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *LogrusLogger) Warn(args ...interface{}) {
	l.logger.Warn(args...)
}

func (l *LogrusLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *LogrusLogger) Error(args ...interface{}) {
	l.logger.Error(args...)
}

func (l *LogrusLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *LogrusLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

func (l *LogrusLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

func (l *LogrusLogger) WithFields(fields map[string]interface{}) Logger {
	entry := l.logger.WithFields(logrus.Fields(fields))
	return &LogrusLogger{logger: entry.Logger}
}

func (l *LogrusLogger) WithField(key string, value interface{}) Logger {
	entry := l.logger.WithField(key, value)
	return &LogrusLogger{logger: entry.Logger}
}

// ZapLogger zap实现
type ZapLogger struct {
	logger *zap.Logger
}

// NewZapLogger 创建ZapLogger
func NewZapLogger(level, format, output, path string) (*ZapLogger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	var encoder zapcore.Encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	var writer zapcore.WriteSyncer
	if output == "file" {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		writer = zapcore.AddSync(io.MultiWriter(os.Stdout, file))
	} else {
		writer = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(encoder, writer, zapLevel)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &ZapLogger{logger: logger}, nil
}

func (l *ZapLogger) Debug(args ...interface{}) {
	l.logger.Sugar().Debug(args...)
}

func (l *ZapLogger) Debugf(format string, args ...interface{}) {
	l.logger.Sugar().Debugf(format, args...)
}

func (l *ZapLogger) Info(args ...interface{}) {
	l.logger.Sugar().Info(args...)
}

func (l *ZapLogger) Infof(format string, args ...interface{}) {
	l.logger.Sugar().Infof(format, args...)
}

func (l *ZapLogger) Warn(args ...interface{}) {
	l.logger.Sugar().Warn(args...)
}

func (l *ZapLogger) Warnf(format string, args ...interface{}) {
	l.logger.Sugar().Warnf(format, args...)
}

func (l *ZapLogger) Error(args ...interface{}) {
	l.logger.Sugar().Error(args...)
}

func (l *ZapLogger) Errorf(format string, args ...interface{}) {
	l.logger.Sugar().Errorf(format, args...)
}

func (l *ZapLogger) Fatal(args ...interface{}) {
	l.logger.Sugar().Fatal(args...)
}

func (l *ZapLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Sugar().Fatalf(format, args...)
}

func (l *ZapLogger) WithFields(fields map[string]interface{}) Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &ZapLogger{logger: l.logger.With(zapFields...)}
}

func (l *ZapLogger) WithField(key string, value interface{}) Logger {
	return &ZapLogger{logger: l.logger.With(zap.Any(key, value))}
}

// NewLogger 创建Logger（默认使用logrus）
func NewLogger(level, format, output, path string) (Logger, error) {
	return NewLogrusLogger(level, format, output, path)
}
