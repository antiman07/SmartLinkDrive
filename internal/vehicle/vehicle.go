package vehicle

import (
	"time"
)

// Vehicle 是 vehicles 表的 GORM 模型（最小可用）。
type Vehicle struct {
	ID          string    `gorm:"primaryKey;size:36"`
	PlateNumber string    `gorm:"uniqueIndex;size:32;not null"`
	VIN         string    `gorm:"size:64"`
	Model       string    `gorm:"size:64"`
	OwnerID     string    `gorm:"index;size:36"`
	Status      string    `gorm:"size:16;not null"` // available / busy / offline
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}
