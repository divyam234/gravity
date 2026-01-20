package store

import (
	"context"
	"database/sql"
	"time"
)

type SearchRepo struct {
	db *sql.DB
}

func NewSearchRepo(db *sql.DB) *SearchRepo {
	return &SearchRepo{db: db}
}

type IndexedFile struct {
	ID            string    `json:"id"`
	Remote        string    `json:"remote"`
	Path          string    `json:"path"`
	Filename      string    `json:"filename"`
	Size          int64     `json:"size"`
	ModTime       time.Time `json:"modTime"`
	IsDir         bool      `json:"isDir"`
	LastIndexedAt time.Time `json:"lastIndexedAt"`
}

type RemoteIndexConfig struct {
	Remote               string     `json:"remote"`
	AutoIndexIntervalMin int        `json:"autoIndexIntervalMin"`
	LastIndexedAt        *time.Time `json:"lastIndexedAt"`
	Status               string     `json:"status"`
	ErrorMsg             string     `json:"errorMsg"`
}

func (r *SearchRepo) SaveFiles(ctx context.Context, remote string, files []IndexedFile) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Delete old files for this remote
	_, err = tx.ExecContext(ctx, "DELETE FROM indexed_files WHERE remote = ?", remote)
	if err != nil {
		return err
	}

	// 2. Insert new files
	query := `
		INSERT INTO indexed_files (id, remote, path, filename, size, mod_time, is_dir, last_indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, f := range files {
		_, err = stmt.ExecContext(ctx, f.ID, remote, f.Path, f.Filename, f.Size, f.ModTime, f.IsDir, now)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SearchRepo) Search(ctx context.Context, query string) ([]IndexedFile, error) {
	sqlQuery := `
		SELECT i.id, i.remote, i.path, i.filename, i.size, i.mod_time, i.is_dir 
		FROM files_search s
		JOIN indexed_files i ON s.rowid = i.id
		WHERE files_search MATCH ?
		ORDER BY rank
		LIMIT 100
	`
	rows, err := r.db.QueryContext(ctx, sqlQuery, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []IndexedFile
	for rows.Next() {
		var f IndexedFile
		if err := rows.Scan(&f.ID, &f.Remote, &f.Path, &f.Filename, &f.Size, &f.ModTime, &f.IsDir); err != nil {
			return nil, err
		}
		results = append(results, f)
	}
	return results, nil
}

func (r *SearchRepo) GetConfigs(ctx context.Context) ([]RemoteIndexConfig, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT remote, auto_index_interval_mins, last_indexed_at, status, error_msg FROM remote_index_config")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []RemoteIndexConfig
	for rows.Next() {
		var c RemoteIndexConfig
		var lastIndexedAt sql.NullTime
		if err := rows.Scan(&c.Remote, &c.AutoIndexIntervalMin, &lastIndexedAt, &c.Status, &c.ErrorMsg); err != nil {
			return nil, err
		}
		if lastIndexedAt.Valid {
			c.LastIndexedAt = &lastIndexedAt.Time
		}
		configs = append(configs, c)
	}
	return configs, nil
}

func (r *SearchRepo) SaveConfig(ctx context.Context, c RemoteIndexConfig) error {
	query := `
		INSERT INTO remote_index_config (remote, auto_index_interval_mins, last_indexed_at, status, error_msg)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(remote) DO UPDATE SET
			auto_index_interval_mins = excluded.auto_index_interval_mins,
			last_indexed_at = excluded.last_indexed_at,
			status = excluded.status,
			error_msg = excluded.error_msg
	`
	_, err := r.db.ExecContext(ctx, query, c.Remote, c.AutoIndexIntervalMin, c.LastIndexedAt, c.Status, c.ErrorMsg)
	return err
}

func (r *SearchRepo) UpdateStatus(ctx context.Context, remote, status, errorMsg string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE remote_index_config SET status = ?, error_msg = ? WHERE remote = ?", status, errorMsg, remote)
	return err
}

func (r *SearchRepo) UpdateLastIndexed(ctx context.Context, remote string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE remote_index_config SET last_indexed_at = ?, status = 'idle', error_msg = '' WHERE remote = ?", time.Now(), remote)
	return err
}
