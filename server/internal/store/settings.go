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

func (r *SettingsRepo) SetMany(ctx context.Context, settings map[string]string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO settings (key, value, updated_at) 
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET 
			value = excluded.value,
			updated_at = excluded.updated_at
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for k, v := range settings {
		if _, err := stmt.ExecContext(ctx, k, v, now); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SettingsRepo) DeleteAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM settings")
	return err
}
