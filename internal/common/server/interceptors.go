package server

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/logger"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryChain 将多个 unary interceptor 串起来（按传入顺序执行）。
func UnaryChain(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		h := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			ic := interceptors[i]
			if ic == nil {
				continue
			}
			next := h
			h = func(currentCtx context.Context, currentReq any) (any, error) {
				return ic(currentCtx, currentReq, info, next)
			}
		}
		return h(ctx, req)
	}
}

// UnaryRecoveryInterceptor 防止 panic 直接把进程打崩，并记录栈信息。
func UnaryRecoveryInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				if log != nil {
					log.Errorf("panic in grpc method=%s err=%v stack=%s", info.FullMethod, r, string(debug.Stack()))
				}
				err = fmt.Errorf("internal error")
			}
		}()
		return handler(ctx, req)
	}
}

// UnaryAccessLogInterceptor 记录每个 gRPC 请求的耗时/错误。
func UnaryAccessLogInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		cost := time.Since(start)

		if log != nil {
			fields := map[string]interface{}{
				"method": info.FullMethod,
				"cost":   cost.String(),
			}
			if err != nil {
				fields["error"] = err.Error()
				log.WithFields(fields).Warn("grpc request failed")
			} else {
				log.WithFields(fields).Info("grpc request ok")
			}
		}

		return resp, err
	}
}

// UnaryTracingInterceptor 基于 OpenTracing 的最小 server interceptor：
// - 从 metadata 里提取 span context（例如 uber-trace-id / traceparent 等，取决于上游注入格式）
// - 创建 server span，并注入到 ctx，方便业务侧 opentracing.StartSpanFromContext 使用
func UnaryTracingInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		tracer := opentracing.GlobalTracer()

		var parent opentracing.SpanContext
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if sc, err := tracer.Extract(opentracing.TextMap, metadataTextMapCarrier(md)); err == nil {
				parent = sc
			}
		}

		operation := info.FullMethod
		if strings.HasPrefix(operation, "/") {
			operation = operation[1:]
		}

		var span opentracing.Span
		if parent != nil {
			span = tracer.StartSpan(operation, ext.RPCServerOption(parent))
		} else {
			span = tracer.StartSpan(operation)
		}
		defer span.Finish()

		ext.SpanKindRPCServer.Set(span)
		ext.Component.Set(span, "grpc")
		if serviceName != "" {
			span.SetTag("service", serviceName)
		}

		ctx = opentracing.ContextWithSpan(ctx, span)
		return handler(ctx, req)
	}
}

// metadataTextMapCarrier 让 gRPC metadata 适配 OpenTracing 的 TextMap。
type metadataTextMapCarrier metadata.MD

func (c metadataTextMapCarrier) ForeachKey(handler func(key, val string) error) error {
	md := metadata.MD(c)
	for k, vs := range md {
		for _, v := range vs {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c metadataTextMapCarrier) Set(key, val string) {
	// server 侧 Extract 不需要 Set；保留实现便于将来扩展（如向下游注入）。
	md := metadata.MD(c)
	md.Set(key, val)
}

