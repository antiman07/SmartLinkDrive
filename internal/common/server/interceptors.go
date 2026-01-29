package server

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/config"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/logger"
	"github.com/golang-jwt/jwt/v5"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

type authContextKey struct{}

// AuthInfo 从 JWT 中解析出的最小用户信息（放入 ctx，供业务侧使用）。
type AuthInfo struct {
	Subject string   // 用户 ID
	Roles   []string // 角色列表（RBAC）
}

// AuthFromContext 从 ctx 中取出鉴权信息。
func AuthFromContext(ctx context.Context) (AuthInfo, bool) {
	v := ctx.Value(authContextKey{})
	if v == nil {
		return AuthInfo{}, false
	}
	ai, ok := v.(AuthInfo)
	return ai, ok
}

// UnaryJWTAuthInterceptor 用于 JWT 鉴权：
// - 从 metadata 中读取 `authorization: Bearer <token>`
// - 校验 HS256 签名、exp/nbf 等标准字段（jwt/v5 默认校验）
// - 可选校验 iss/aud
// - 将解析结果写入 ctx
func UnaryJWTAuthInterceptor(cfg config.AuthConfig, log logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !cfg.Enabled {
			return handler(ctx, req)
		}
		if isPublicMethod(cfg.PublicMethods, info.FullMethod) {
			return handler(ctx, req)
		}
		if strings.TrimSpace(cfg.JWTSecret) == "" {
			if log != nil {
				log.Warn("auth enabled but jwt_secret is empty")
			}
			return nil, status.Error(codes.Unauthenticated, "auth not configured")
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		raw := ""
		if vs := md.Get("authorization"); len(vs) > 0 {
			raw = vs[0]
		}
		if raw == "" {
			return nil, status.Error(codes.Unauthenticated, "missing authorization")
		}

		tokenStr := strings.TrimSpace(raw)
		if strings.HasPrefix(strings.ToLower(tokenStr), "bearer ") {
			tokenStr = strings.TrimSpace(tokenStr[len("bearer "):])
		}
		if tokenStr == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization")
		}

		claims := struct {
			Roles []string `json:"roles"`
			jwt.RegisteredClaims
		}{}

		parsed, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %s", t.Method.Alg())
			}
			return []byte(cfg.JWTSecret), nil
		}, jwt.WithLeeway(30*time.Second))
		if err != nil || parsed == nil || !parsed.Valid {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		if cfg.Issuer != "" && claims.Issuer != cfg.Issuer {
			return nil, status.Error(codes.Unauthenticated, "invalid issuer")
		}
		if cfg.Audience != "" {
			if !audienceContains(claims.Audience, cfg.Audience) {
				return nil, status.Error(codes.Unauthenticated, "invalid audience")
			}
		}

		ctx = context.WithValue(ctx, authContextKey{}, AuthInfo{
			Subject: claims.Subject,
			Roles:   claims.Roles,
		})
		return handler(ctx, req)
	}
}

// UnaryRBACInterceptor 基于 method->roles 的简单 RBAC：
// - 若 cfg.RBAC[info.FullMethod] 存在且非空，则要求 token roles 与之有交集
// - 若该方法未配置要求角色，则默认放行（即“只鉴权，不限权”）
func UnaryRBACInterceptor(cfg config.AuthConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !cfg.Enabled {
			return handler(ctx, req)
		}
		if isPublicMethod(cfg.PublicMethods, info.FullMethod) {
			return handler(ctx, req)
		}

		required := cfg.RBAC[info.FullMethod]
		if len(required) == 0 {
			return handler(ctx, req)
		}

		ai, ok := AuthFromContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing auth context")
		}
		if hasAnyRole(ai.Roles, required) {
			return handler(ctx, req)
		}
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
}

func hasAnyRole(got, required []string) bool {
	if len(got) == 0 || len(required) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(got))
	for _, r := range got {
		r = strings.TrimSpace(strings.ToLower(r))
		if r == "" {
			continue
		}
		set[r] = struct{}{}
	}
	for _, r := range required {
		r = strings.TrimSpace(strings.ToLower(r))
		if r == "" {
			continue
		}
		if _, ok := set[r]; ok {
			return true
		}
	}
	return false
}

func audienceContains(aud jwt.ClaimStrings, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" || len(aud) == 0 {
		return false
	}
	for _, v := range aud {
		if strings.TrimSpace(v) == want {
			return true
		}
	}
	return false
}

func isPublicMethod(public []string, method string) bool {
	if method == "" || len(public) == 0 {
		return false
	}
	for _, m := range public {
		if strings.TrimSpace(m) == method {
			return true
		}
	}
	return false
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
