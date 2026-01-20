package store

import (
	"context"
	"database/sql"
	"time"
)

type CacheRepo struct {
	db *sql.DB
}

func NewCacheRepo(db *sql.DB) *CacheRepo {
	return &CacheRepo{db: db}
}

func (r *CacheRepo) Get(ctx context.Context, key string) ([]byte, error) {
	var value []byte
	var expiresAt time.Time

	query := "SELECT value, expires_at FROM kv_cache WHERE key = ?"
	err := r.db.QueryRowContext(ctx, query, key).Scan(&value, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, err
	}

	if time.Now().After(expiresAt) {
		// Expired, cleanup and return nil
		r.Delete(ctx, key)
		return nil, nil
	}

	return value, nil
}

func (r *CacheRepo) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	query := `
		INSERT INTO kv_cache (key, value, created_at, expires_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			created_at = excluded.created_at,
			expires_at = excluded.expires_at
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, key, value, now, now.Add(ttl))
	return err
}

func (r *CacheRepo) Delete(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM kv_cache WHERE key = ?", key)
	return err
}

func (r *CacheRepo) DeletePrefix(ctx context.Context, prefix string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM kv_cache WHERE key LIKE ?", prefix+"%")
	return err
}
