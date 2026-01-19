package store

import (
	"context"
	"database/sql"
	"time"
)

type StatsRepo struct {
	db *sql.DB
}

func NewStatsRepo(db *sql.DB) *StatsRepo {
	return &StatsRepo{db: db}
}

func (r *StatsRepo) Increment(ctx context.Context, key string, value int64) error {
	query := `
		INSERT INTO stats (key, value, updated_at) 
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET 
			value = value + excluded.value,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(ctx, query, key, value, time.Now())
	return err
}

func (r *StatsRepo) Get(ctx context.Context) (map[string]int64, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT key, value FROM stats")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int64)
	for rows.Next() {
		var key string
		var value int64
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		stats[key] = value
	}
	return stats, nil
}
