package order

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) withCtx(ctx context.Context) *gorm.DB {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.WithContext(ctx)
}

func (r *Repo) Create(ctx context.Context, o *Order) error {
	db := r.withCtx(ctx)
	if db == nil {
		return fmt.Errorf("repo db is nil")
	}
	return db.Create(o).Error
}

func (r *Repo) Update(ctx context.Context, o *Order) error {
	db := r.withCtx(ctx)
	if db == nil {
		return fmt.Errorf("repo db is nil")
	}
	return db.Save(o).Error
}

func (r *Repo) GetByID(ctx context.Context, id string) (*Order, error) {
	db := r.withCtx(ctx)
	if db == nil {
		return nil, fmt.Errorf("repo db is nil")
	}
	var o Order
	if err := db.Where("id = ?", id).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// List 支持按 user_id / status 过滤 + 分页。
func (r *Repo) List(ctx context.Context, userID string, status Status, offset, limit int) ([]Order, int64, error) {
	db := r.withCtx(ctx)
	if db == nil {
		return nil, 0, fmt.Errorf("repo db is nil")
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	q := db.Model(&Order{})
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var orders []Order
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}
