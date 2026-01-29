package user

import (
	"strings"
	"time"
)

// User 是 users 表的 GORM 模型（最小可用）。
type User struct {
	ID           string    `gorm:"primaryKey;size:36"`
	Username     string    `gorm:"uniqueIndex;size:64;not null"`
	PasswordHash string    `gorm:"size:128;not null"`
	PasswordSalt string    `gorm:"size:64;not null"`
	Nickname     string    `gorm:"size:64"`
	Phone        string    `gorm:"size:32"`
	Email        string    `gorm:"size:128"`
	Roles        string    `gorm:"size:256;not null"` // 逗号分隔，例如 "user,admin"
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (u User) RolesSlice() []string {
	if strings.TrimSpace(u.Roles) == "" {
		return nil
	}
	parts := strings.Split(u.Roles, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func RolesJoin(roles []string) string {
	if len(roles) == 0 {
		return ""
	}
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		out = append(out, r)
	}
	return strings.Join(out, ",")
}
