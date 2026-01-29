package auth

import (
	"testing"
	"time"

	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/config"
	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAccessToken(t *testing.T) {
	cfg := config.AuthConfig{
		Enabled:   true,
		JWTSecret: "test-secret",
		Issuer:    "smartlinkdrive",
		Audience:  "smartlinkdrive",
	}

	token, exp, err := GenerateAccessToken(cfg, "u-1", []string{"user"}, time.Hour)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	if token == "" {
		t.Fatalf("expected token")
	}
	if exp.Before(time.Now()) {
		t.Fatalf("expected exp in future")
	}

	claims := &struct {
		Roles []string `json:"roles"`
		jwt.RegisteredClaims
	}{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		t.Fatalf("parse token: %v", err)
	}
	if claims.Subject != "u-1" {
		t.Fatalf("subject mismatch: %s", claims.Subject)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "user" {
		t.Fatalf("roles mismatch: %#v", claims.Roles)
	}
}
