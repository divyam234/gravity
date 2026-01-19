package store

import (
	"context"
	"database/sql"
	"time"
)

type SettingsRepo struct {
	db *sql.DB
}

func NewSettingsRepo(db *sql.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) Get(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, nil
}

func (r *SettingsRepo) Set(ctx context.Context, key, value string) error {
	query := `
		INSERT INTO settings (key, value, updated_at) 
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET 
			value = excluded.value,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(ctx, query, key, value, time.Now())
	return err
}
