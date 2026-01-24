package store

import (
	"context"

	"gravity/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProviderRepo struct {
	db *gorm.DB
}

func NewProviderRepo(db *gorm.DB) *ProviderRepo {
	return &ProviderRepo{db: db}
}

func (r *ProviderRepo) Save(ctx context.Context, p *model.Provider) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(p).Error
}

func (r *ProviderRepo) Get(ctx context.Context, name string) (*model.Provider, error) {
	var p model.Provider
	err := r.db.WithContext(ctx).First(&p, "name = ?", name).Error
	return &p, err
}

func (r *ProviderRepo) List(ctx context.Context) ([]*model.Provider, error) {
	var providers []*model.Provider
	err := r.db.WithContext(ctx).Find(&providers).Error
	return providers, err
}
