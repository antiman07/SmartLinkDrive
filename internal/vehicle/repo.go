package vehicle

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

func (r *Repo) Upsert(ctx context.Context, v *Vehicle) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repo db is nil")
	}
	return r.db.WithContext(ctx).Save(v).Error
}

func (r *Repo) FindByID(ctx context.Context, id string) (*Vehicle, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repo db is nil")
	}
	var v Vehicle
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *Repo) List(ctx context.Context, ownerID string, offset, limit int) ([]Vehicle, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("repo db is nil")
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	q := r.db.WithContext(ctx).Model(&Vehicle{})
	if ownerID != "" {
		q = q.Where("owner_id = ?", ownerID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var vehicles []Vehicle
	if err := q.Order("created_at desc").Offset(offset).Limit(limit).Find(&vehicles).Error; err != nil {
		return nil, 0, err
	}
	return vehicles, total, nil
}
