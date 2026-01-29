package server

import (
	"context"
	"testing"
	"time"

	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/config"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryJWTAuthInterceptorAndRBAC(t *testing.T) {
	authCfg := config.AuthConfig{
		Enabled:   true,
		JWTSecret: "test-secret",
		Issuer:    "smartlinkdrive",
		Audience:  "smartlinkdrive",
		RBAC: map[string][]string{
			"/x.y.Service/AdminOnly": {"admin"},
			"/x.y.Service/Open":      {},
		},
	}

	// 生成一个带 roles 的 token
	now := time.Now()
	claims := struct {
		Roles []string `json:"roles"`
		jwt.RegisteredClaims
	}{
		Roles: []string{"user", "admin"},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "u-1",
			Issuer:    authCfg.Issuer,
			Audience:  jwt.ClaimStrings{authCfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(authCfg.JWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	authIC := UnaryJWTAuthInterceptor(authCfg, nil)
	rbacIC := UnaryRBACInterceptor(authCfg)
	chain := UnaryChain(authIC, rbacIC)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
	info := &grpc.UnaryServerInfo{FullMethod: "/x.y.Service/AdminOnly"}

	_, err = chain(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		ai, ok := AuthFromContext(ctx)
		if !ok {
			t.Fatalf("missing auth info in ctx")
		}
		if ai.Subject != "u-1" {
			t.Fatalf("subject mismatch: %s", ai.Subject)
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("expected allow, got err=%v", err)
	}

	// 换一个只有 user 角色的 token，应被 RBAC 拒绝
	claims.Roles = []string{"user"}
	token2 := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr2, err := token2.SignedString([]byte(authCfg.JWTSecret))
	if err != nil {
		t.Fatalf("sign token2: %v", err)
	}
	ctx2 := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr2))

	_, err = chain(ctx2, nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err == nil {
		t.Fatalf("expected permission denied, got nil")
	}
}

