package user

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	passwordSaltBytes = 16
	passwordIters     = 100_000
)

func GenerateSaltHex() (string, error) {
	b := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func HashPassword(password, saltHex string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password is empty")
	}
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return "", fmt.Errorf("invalid salt: %w", err)
	}
	// 简化实现：多轮 SHA256(salt || password || prev)。
	// 说明：生产建议使用 bcrypt/argon2（需要额外依赖与环境支持）。
	var prev [32]byte
	for i := 0; i < passwordIters; i++ {
		h := sha256.New()
		_, _ = h.Write(salt)
		_, _ = h.Write([]byte(password))
		_, _ = h.Write(prev[:])
		copy(prev[:], h.Sum(nil))
	}
	return hex.EncodeToString(prev[:]), nil
}

func VerifyPassword(password, saltHex, wantHashHex string) bool {
	got, err := HashPassword(password, saltHex)
	if err != nil {
		return false
	}
	// 这里使用普通字符串比较（演示环境）；如需更严格可使用 subtle.ConstantTimeCompare。
	return got == wantHashHex
}
