package store

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StatsRepo struct {
	db *gorm.DB
}

func NewStatsRepo(db *gorm.DB) *StatsRepo {
	return &StatsRepo{db: db}
}

func (r *StatsRepo) Increment(ctx context.Context, key string, value int64) error {
	item := StatsKV{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"value":      gorm.Expr("stats.value + excluded.value"),
			"updated_at": gorm.Expr("excluded.updated_at"),
		}),
	}).Create(&item).Error
}

func (r *StatsRepo) Get(ctx context.Context) (map[string]int64, error) {
	var results []StatsKV
	err := r.db.WithContext(ctx).Find(&results).Error
	if err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, res := range results {
		stats[res.Key] = res.Value
	}
	return stats, nil
}
