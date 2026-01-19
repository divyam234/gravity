package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gravity/internal/model"
)

type DownloadRepo struct {
	db *sql.DB
}

func NewDownloadRepo(db *sql.DB) *DownloadRepo {
	return &DownloadRepo{db: db}
}

func (r *DownloadRepo) Create(ctx context.Context, d *model.Download) error {
	tagsJson, _ := json.Marshal(d.Tags)
	query := `
		INSERT INTO downloads (
			id, url, resolved_url, provider, status, error, filename, local_path, size, 
			downloaded, speed, eta, destination, category, tags, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		d.ID, d.URL, d.ResolvedURL, d.Provider, d.Status, d.Error, d.Filename, d.LocalPath, d.Size,
		d.Downloaded, d.Speed, d.ETA, d.Destination, d.Category, string(tagsJson),
		d.CreatedAt, d.UpdatedAt,
	)
	return err
}

func (r *DownloadRepo) Get(ctx context.Context, id string) (*model.Download, error) {
	query := `SELECT * FROM downloads WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanDownload(row)
}

func (r *DownloadRepo) GetByEngineID(ctx context.Context, engineID string) (*model.Download, error) {
	query := `SELECT * FROM downloads WHERE engine_id = ?`
	row := r.db.QueryRowContext(ctx, query, engineID)
	return r.scanDownload(row)
}

func (r *DownloadRepo) GetByUploadJobID(ctx context.Context, jobID string) (*model.Download, error) {
	query := `SELECT * FROM downloads WHERE upload_job_id = ?`
	row := r.db.QueryRowContext(ctx, query, jobID)
	return r.scanDownload(row)
}

func (r *DownloadRepo) List(ctx context.Context, status []string, limit, offset int) ([]*model.Download, int, error) {
	where := ""
	args := []interface{}{}
	if len(status) > 0 {
		where = "WHERE status IN (" + strings.Repeat("?,", len(status)-1) + "?)"
		for _, s := range status {
			args = append(args, s)
		}
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM downloads %s", where)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf("SELECT * FROM downloads %s ORDER BY created_at DESC LIMIT ? OFFSET ?", where)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	downloads := []*model.Download{}
	for rows.Next() {
		d, err := r.scanDownload(rows)
		if err != nil {
			return nil, 0, err
		}
		downloads = append(downloads, d)
	}

	return downloads, total, nil
}

func (r *DownloadRepo) Update(ctx context.Context, d *model.Download) error {
	tagsJson, _ := json.Marshal(d.Tags)
	d.UpdatedAt = time.Now()

	query := `
		UPDATE downloads SET
			status = ?, error = ?, filename = ?, local_path = ?, size = ?, downloaded = ?, 
			speed = ?, eta = ?, destination = ?, upload_status = ?, 
			upload_progress = ?, upload_speed = ?, category = ?, tags = ?, 
			engine_id = ?, upload_job_id = ?, started_at = ?, 
			completed_at = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		d.Status, d.Error, d.Filename, d.LocalPath, d.Size, d.Downloaded,
		d.Speed, d.ETA, d.Destination, d.UploadStatus,
		d.UploadProgress, d.UploadSpeed, d.Category, string(tagsJson),
		d.EngineID, d.UploadJobID, d.StartedAt,
		d.CompletedAt, d.UpdatedAt,
		d.ID,
	)
	return err
}

func (r *DownloadRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM downloads WHERE id = ?`, id)
	return err
}

func (r *DownloadRepo) scanDownload(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Download, error) {
	d := &model.Download{}
	var tags string
	var resolvedURL, provider, errStr, filename, localPath, destination, uploadStatus, category, engineID, uploadJobID sql.NullString
	var startedAt, completedAt sql.NullTime

	err := scanner.Scan(
		&d.ID, &d.URL, &resolvedURL, &provider, &d.Status, &errStr,
		&filename, &localPath, &d.Size, &d.Downloaded, &d.Speed, &d.ETA,
		&destination, &uploadStatus, &d.UploadProgress, &d.UploadSpeed,
		&category, &tags, &engineID, &uploadJobID,
		&d.CreatedAt, &startedAt, &completedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	d.ResolvedURL = resolvedURL.String
	d.Provider = provider.String
	d.Error = errStr.String
	d.Filename = filename.String
	d.LocalPath = localPath.String
	d.Destination = destination.String
	d.UploadStatus = uploadStatus.String
	d.Category = category.String
	d.EngineID = engineID.String
	d.UploadJobID = uploadJobID.String

	if startedAt.Valid {
		d.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		d.CompletedAt = &completedAt.Time
	}

	json.Unmarshal([]byte(tags), &d.Tags)

	return d, nil
}
