package store

import (
	"context"
	"time"

	"gravity/internal/model"

	"gorm.io/gorm"
)

type SettingsRepo struct {
	db *gorm.DB
}

func NewSettingsRepo(db *gorm.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) Get(ctx context.Context) (*model.Settings, error) {
	var s model.Settings
	// We always use ID 1 for the global settings record
	err := r.db.WithContext(ctx).First(&s, 1).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &s, err
}

func (r *SettingsRepo) Save(ctx context.Context, s *model.Settings) error {
	s.ID = 1
	s.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *SettingsRepo) DeleteAll(ctx context.Context) error {
	return r.db.WithContext(ctx).Delete(&model.Settings{}, 1).Error
}