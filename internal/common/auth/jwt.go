package auth

import (
	"fmt"
	"time"

	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/config"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// GenerateAccessToken 生成 HS256 JWT access token。
func GenerateAccessToken(cfg config.AuthConfig, subject string, roles []string, ttl time.Duration) (token string, expiresAt time.Time, err error) {
	if subject == "" {
		return "", time.Time{}, fmt.Errorf("subject is empty")
	}
	if cfg.JWTSecret == "" {
		return "", time.Time{}, fmt.Errorf("jwt_secret is empty")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	now := time.Now()
	expiresAt = now.Add(ttl)

	c := Claims{
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    cfg.Issuer,
			Audience:  audience(cfg.Audience),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := t.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func audience(aud string) jwt.ClaimStrings {
	if aud == "" {
		return nil
	}
	return jwt.ClaimStrings{aud}
}
