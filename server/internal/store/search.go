package store

import (
	"context"
	"database/sql"
	"strings"
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
	Name          string    `json:"name"`
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
	ExcludedPatterns     string     `json:"excludedPatterns"`   // Comma-separated regex
	IncludedExtensions   string     `json:"includedExtensions"` // Comma-separated extensions
	MinSizeBytes         int64      `json:"minSizeBytes"`
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
		_, err = stmt.ExecContext(ctx, f.ID, remote, f.Path, f.Name, f.Size, f.ModTime, f.IsDir, now)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SearchRepo) Search(ctx context.Context, query string, limit, offset int) ([]IndexedFile, int, error) {
	// Simple escaping and prefix matching
	// Replaces double quotes and appends * to the query
	ftsQuery := strings.ReplaceAll(query, "\"", "") + "*"

	countQuery := `
		SELECT COUNT(*) 
		FROM files_search s
		JOIN indexed_files i ON s.rowid = i.rowid
		WHERE files_search MATCH ?
	`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, ftsQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	sqlQuery := `
		SELECT i.id, i.remote, i.path, i.filename, i.size, i.mod_time, i.is_dir, i.last_indexed_at
		FROM files_search s
		JOIN indexed_files i ON s.rowid = i.rowid
		WHERE files_search MATCH ?
		ORDER BY rank
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.QueryContext(ctx, sqlQuery, ftsQuery, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []IndexedFile
	for rows.Next() {
		var f IndexedFile
		if err := rows.Scan(&f.ID, &f.Remote, &f.Path, &f.Name, &f.Size, &f.ModTime, &f.IsDir, &f.LastIndexedAt); err != nil {
			return nil, 0, err
		}
		results = append(results, f)
	}
	return results, total, nil
}

func (r *SearchRepo) GetConfigs(ctx context.Context) ([]RemoteIndexConfig, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT remote, auto_index_interval_mins, last_indexed_at, status, error_msg, excluded_patterns, included_extensions, min_size_bytes FROM remote_index_config")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []RemoteIndexConfig
	for rows.Next() {
		var c RemoteIndexConfig
		var lastIndexedAt sql.NullTime
		var excluded, included sql.NullString
		var minSize sql.NullInt64
		if err := rows.Scan(&c.Remote, &c.AutoIndexIntervalMin, &lastIndexedAt, &c.Status, &c.ErrorMsg, &excluded, &included, &minSize); err != nil {
			return nil, err
		}
		if lastIndexedAt.Valid {
			c.LastIndexedAt = &lastIndexedAt.Time
		}
		c.ExcludedPatterns = excluded.String
		c.IncludedExtensions = included.String
		c.MinSizeBytes = minSize.Int64
		configs = append(configs, c)
	}
	return configs, nil
}

func (r *SearchRepo) SaveConfig(ctx context.Context, c RemoteIndexConfig) error {
	query := `
		INSERT INTO remote_index_config (remote, auto_index_interval_mins, last_indexed_at, status, error_msg, excluded_patterns, included_extensions, min_size_bytes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(remote) DO UPDATE SET
			auto_index_interval_mins = excluded.auto_index_interval_mins,
			last_indexed_at = excluded.last_indexed_at,
			status = excluded.status,
			error_msg = excluded.error_msg,
			excluded_patterns = excluded.excluded_patterns,
			included_extensions = excluded.included_extensions,
			min_size_bytes = excluded.min_size_bytes
	`
	_, err := r.db.ExecContext(ctx, query, c.Remote, c.AutoIndexIntervalMin, c.LastIndexedAt, c.Status, c.ErrorMsg, c.ExcludedPatterns, c.IncludedExtensions, c.MinSizeBytes)
	return err
}

func (r *SearchRepo) UpdateStatus(ctx context.Context, remote, status, errorMsg string) error {
	query := `
		INSERT INTO remote_index_config (remote, status, error_msg)
		VALUES (?, ?, ?)
		ON CONFLICT(remote) DO UPDATE SET
			status = excluded.status,
			error_msg = excluded.error_msg
	`
	_, err := r.db.ExecContext(ctx, query, remote, status, errorMsg)
	return err
}

func (r *SearchRepo) UpdateLastIndexed(ctx context.Context, remote string) error {
	query := `
		INSERT INTO remote_index_config (remote, last_indexed_at, status, error_msg)
		VALUES (?, ?, 'idle', '')
		ON CONFLICT(remote) DO UPDATE SET
			last_indexed_at = excluded.last_indexed_at,
			status = excluded.status,
			error_msg = excluded.error_msg
	`
	_, err := r.db.ExecContext(ctx, query, remote, time.Now())
	return err
}
