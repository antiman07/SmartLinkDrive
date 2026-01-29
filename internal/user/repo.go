package user

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

func (r *Repo) Create(ctx context.Context, u *User) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("repo db is nil")
	}
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *Repo) FindByUsername(ctx context.Context, username string) (*User, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repo db is nil")
	}
	var u User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repo) FindByID(ctx context.Context, id string) (*User, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("repo db is nil")
	}
	var u User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repo) List(ctx context.Context, offset, limit int) ([]User, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("repo db is nil")
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	var total int64
	if err := r.db.WithContext(ctx).Model(&User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []User
	if err := r.db.WithContext(ctx).Order("created_at desc").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}
