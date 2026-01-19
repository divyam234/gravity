package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"gravity/internal/model"
)

type ProviderRepo struct {
	db *sql.DB
}

func NewProviderRepo(db *sql.DB) *ProviderRepo {
	return &ProviderRepo{db: db}
}

func (r *ProviderRepo) Save(ctx context.Context, p *model.Provider) error {
	configJson, _ := json.Marshal(p.Config)
	hostsJson, _ := json.Marshal(p.CachedHosts)
	accountJson, _ := json.Marshal(p.CachedAccount)

	query := `
		INSERT INTO providers (name, enabled, priority, config, cached_hosts, cached_account, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET 
			enabled = excluded.enabled,
			priority = excluded.priority,
			config = excluded.config,
			cached_hosts = excluded.cached_hosts,
			cached_account = excluded.cached_account,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		p.Name, p.Enabled, p.Priority, string(configJson), string(hostsJson), string(accountJson), time.Now(),
	)
	return err
}

func (r *ProviderRepo) Get(ctx context.Context, name string) (*model.Provider, error) {
	query := `SELECT * FROM providers WHERE name = ?`
	var p model.Provider
	var configJson, hostsJson, accountJson string

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&p.Name, &p.Enabled, &p.Priority, &configJson, &hostsJson, &accountJson, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(configJson), &p.Config)
	json.Unmarshal([]byte(hostsJson), &p.CachedHosts)
	json.Unmarshal([]byte(accountJson), &p.CachedAccount)

	return &p, nil
}

func (r *ProviderRepo) List(ctx context.Context) ([]*model.Provider, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT * FROM providers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*model.Provider
	for rows.Next() {
		var p model.Provider
		var configJson, hostsJson, accountJson string
		err := rows.Scan(
			&p.Name, &p.Enabled, &p.Priority, &configJson, &hostsJson, &accountJson, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(configJson), &p.Config)
		json.Unmarshal([]byte(hostsJson), &p.CachedHosts)
		json.Unmarshal([]byte(accountJson), &p.CachedAccount)
		providers = append(providers, &p)
	}
	return providers, nil
}
