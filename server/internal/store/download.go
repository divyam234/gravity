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

const downloadColumns = `
	id, url, resolved_url, provider, status, error, 
	filename, local_path, size, downloaded, speed, eta, 
	destination, upload_status, upload_progress, upload_speed, 
	category, tags, engine_id, upload_job_id,
	is_magnet, magnet_hash, magnet_source, magnet_id, 
	total_files, files_complete,
	seeders, peers,
	headers,
	torrent_data, selected_files,
	created_at, started_at, completed_at, updated_at
`

type DownloadRepo struct {
	db *sql.DB
}

func NewDownloadRepo(db *sql.DB) *DownloadRepo {
	return &DownloadRepo{db: db}
}

func (r *DownloadRepo) Create(ctx context.Context, d *model.Download) error {
	tagsJson, _ := json.Marshal(d.Tags)
	headersJson, _ := json.Marshal(d.Headers)
	selectedFilesJson, _ := json.Marshal(d.SelectedFiles)
	query := fmt.Sprintf(`
		INSERT INTO downloads (%s) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, downloadColumns)

	_, err := r.db.ExecContext(ctx, query,
		d.ID, d.URL, d.ResolvedURL, d.Provider, d.Status, d.Error,
		d.Filename, d.LocalPath, d.Size, d.Downloaded, d.Speed, d.ETA,
		d.Destination, d.UploadStatus, d.UploadProgress, d.UploadSpeed,
		d.Category, string(tagsJson), d.EngineID, d.UploadJobID,
		d.IsMagnet, d.MagnetHash, d.MagnetSource, d.MagnetID,
		d.TotalFiles, d.FilesComplete,
		d.Seeders, d.Peers,
		string(headersJson),
		d.TorrentData, string(selectedFilesJson),
		d.CreatedAt, d.StartedAt, d.CompletedAt, d.UpdatedAt,
	)
	return err
}

func (r *DownloadRepo) Get(ctx context.Context, id string) (*model.Download, error) {
	query := fmt.Sprintf("SELECT %s FROM downloads WHERE id = ?", downloadColumns)
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanDownload(row)
}

func (r *DownloadRepo) GetByEngineID(ctx context.Context, engineID string) (*model.Download, error) {
	query := fmt.Sprintf("SELECT %s FROM downloads WHERE engine_id = ?", downloadColumns)
	row := r.db.QueryRowContext(ctx, query, engineID)
	return r.scanDownload(row)
}

func (r *DownloadRepo) GetByUploadJobID(ctx context.Context, jobID string) (*model.Download, error) {
	query := fmt.Sprintf("SELECT %s FROM downloads WHERE upload_job_id = ?", downloadColumns)
	row := r.db.QueryRowContext(ctx, query, jobID)
	return r.scanDownload(row)
}

func (r *DownloadRepo) List(ctx context.Context, status []string, limit, offset int, sortAsc bool) ([]*model.Download, int, error) {
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

	order := "DESC"
	if sortAsc {
		order = "ASC"
	}

	query := fmt.Sprintf("SELECT %s FROM downloads %s ORDER BY created_at %s LIMIT ? OFFSET ?", downloadColumns, where, order)
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
	headersJson, _ := json.Marshal(d.Headers)
	selectedFilesJson, _ := json.Marshal(d.SelectedFiles)
	d.UpdatedAt = time.Now()

	query := `
		UPDATE downloads SET
			status = ?, error = ?, filename = ?, local_path = ?, size = ?, downloaded = ?, 
			speed = ?, eta = ?, destination = ?, upload_status = ?, 
			upload_progress = ?, upload_speed = ?, category = ?, tags = ?, 
			engine_id = ?, upload_job_id = ?,
			is_magnet = ?, magnet_hash = ?, magnet_source = ?, magnet_id = ?,
			total_files = ?, files_complete = ?,
			seeders = ?, peers = ?,
			headers = ?,
			torrent_data = ?, selected_files = ?,
			started_at = ?, completed_at = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		d.Status, d.Error, d.Filename, d.LocalPath, d.Size, d.Downloaded,
		d.Speed, d.ETA, d.Destination, d.UploadStatus,
		d.UploadProgress, d.UploadSpeed, d.Category, string(tagsJson),
		d.EngineID, d.UploadJobID,
		d.IsMagnet, d.MagnetHash, d.MagnetSource, d.MagnetID,
		d.TotalFiles, d.FilesComplete,
		d.Seeders, d.Peers,
		string(headersJson),
		d.TorrentData, string(selectedFilesJson),
		d.StartedAt, d.CompletedAt, d.UpdatedAt,
		d.ID,
	)
	return err
}

func (r *DownloadRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM downloads WHERE id = ?`, id)
	return err
}

func (r *DownloadRepo) Count(ctx context.Context, status []string) (int, error) {
	where := ""
	args := []interface{}{}
	if len(status) > 0 {
		where = "WHERE status IN (" + strings.Repeat("?,", len(status)-1) + "?)"
		for _, s := range status {
			args = append(args, s)
		}
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM downloads %s", where)
	var total int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *DownloadRepo) GetStatusCounts(ctx context.Context) (map[model.DownloadStatus]int, error) {
	query := `SELECT status, COUNT(*) FROM downloads GROUP BY status`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[model.DownloadStatus]int)
	for rows.Next() {
		var status model.DownloadStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, nil
}

func (r *DownloadRepo) scanDownload(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Download, error) {
	d := &model.Download{}
	var tags, headers, selectedFiles sql.NullString
	var resolvedURL, provider, errStr, filename, localPath, destination, uploadStatus, category, engineID, uploadJobID sql.NullString
	var magnetHash, magnetSource, magnetID, torrentData sql.NullString
	var startedAt, completedAt sql.NullTime

	err := scanner.Scan(
		&d.ID, &d.URL, &resolvedURL, &provider, &d.Status, &errStr,
		&filename, &localPath, &d.Size, &d.Downloaded, &d.Speed, &d.ETA,
		&destination, &uploadStatus, &d.UploadProgress, &d.UploadSpeed,
		&category, &tags, &engineID, &uploadJobID,
		&d.IsMagnet, &magnetHash, &magnetSource, &magnetID,
		&d.TotalFiles, &d.FilesComplete,
		&d.Seeders, &d.Peers,
		&headers,
		&torrentData, &selectedFiles,
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
	d.MagnetHash = magnetHash.String
	d.MagnetSource = magnetSource.String
	d.MagnetID = magnetID.String
	d.TorrentData = torrentData.String

	if startedAt.Valid {
		d.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		d.CompletedAt = &completedAt.Time
	}

	if tags.Valid && tags.String != "" {
		json.Unmarshal([]byte(tags.String), &d.Tags)
	}
	if headers.Valid && headers.String != "" {
		json.Unmarshal([]byte(headers.String), &d.Headers)
	}
	if selectedFiles.Valid && selectedFiles.String != "" {
		json.Unmarshal([]byte(selectedFiles.String), &d.SelectedFiles)
	}

	return d, nil
}

// CreateWithFiles creates a download with its associated files
func (r *DownloadRepo) CreateWithFiles(ctx context.Context, d *model.Download) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create download
	tagsJson, _ := json.Marshal(d.Tags)
	headersJson, _ := json.Marshal(d.Headers)
	selectedFilesJson, _ := json.Marshal(d.SelectedFiles)
	query := fmt.Sprintf(`
		INSERT INTO downloads (%s) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, downloadColumns)

	_, err = tx.ExecContext(ctx, query,
		d.ID, d.URL, d.ResolvedURL, d.Provider, d.Status, d.Error,
		d.Filename, d.LocalPath, d.Size, d.Downloaded, d.Speed, d.ETA,
		d.Destination, d.UploadStatus, d.UploadProgress, d.UploadSpeed,
		d.Category, string(tagsJson), d.EngineID, d.UploadJobID,
		d.IsMagnet, d.MagnetHash, d.MagnetSource, d.MagnetID,
		d.TotalFiles, d.FilesComplete,
		d.Seeders, d.Peers,
		string(headersJson),
		d.TorrentData, string(selectedFilesJson),
		d.CreatedAt, d.StartedAt, d.CompletedAt, d.UpdatedAt,
	)
	if err != nil {
		return err
	}

	// Create files
	for _, f := range d.Files {
		fileQuery := `
			INSERT INTO download_files (
				id, download_id, name, path, size, downloaded, progress, status, error, engine_id, url, file_index, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err = tx.ExecContext(ctx, fileQuery,
			f.ID, d.ID, f.Name, f.Path, f.Size, f.Downloaded, f.Progress, f.Status, f.Error, f.EngineID, f.URL, f.Index,
			time.Now(), time.Now(),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetFiles returns all files for a download with pagination
func (r *DownloadRepo) GetFiles(ctx context.Context, downloadID string, limit, offset int) ([]model.DownloadFile, int, error) {
	countQuery := `SELECT COUNT(*) FROM download_files WHERE download_id = ?`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, downloadID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, download_id, name, path, size, downloaded, progress, status, error, engine_id, url, file_index FROM download_files WHERE download_id = ? ORDER BY file_index ASC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, downloadID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var files []model.DownloadFile
	for rows.Next() {
		var f model.DownloadFile
		var errStr, engineID, url sql.NullString
		var fileIndex sql.NullInt64
		err := rows.Scan(&f.ID, &f.DownloadID, &f.Name, &f.Path, &f.Size, &f.Downloaded, &f.Progress, &f.Status, &errStr, &engineID, &url, &fileIndex)
		if err != nil {
			return nil, 0, err
		}
		f.Error = errStr.String
		f.EngineID = engineID.String
		f.URL = url.String
		if fileIndex.Valid {
			f.Index = int(fileIndex.Int64)
		}
		files = append(files, f)
	}
	return files, total, nil
}

// UpdateFile updates a single download file
func (r *DownloadRepo) UpdateFile(ctx context.Context, f *model.DownloadFile) error {
	query := `
		UPDATE download_files SET
			downloaded = ?, progress = ?, status = ?, error = ?, engine_id = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, f.Downloaded, f.Progress, f.Status, f.Error, f.EngineID, time.Now(), f.ID)
	return err
}

// UpdateFiles updates all files for a download
func (r *DownloadRepo) UpdateFiles(ctx context.Context, downloadID string, files []model.DownloadFile) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, f := range files {
		query := `
			UPDATE download_files SET
				downloaded = ?, progress = ?, status = ?, error = ?, engine_id = ?, updated_at = ?
			WHERE id = ? AND download_id = ?
		`
		_, err = tx.ExecContext(ctx, query, f.Downloaded, f.Progress, f.Status, f.Error, f.EngineID, time.Now(), f.ID, downloadID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetFileByEngineID finds a download file by its aria2c GID
func (r *DownloadRepo) GetFileByEngineID(ctx context.Context, engineID string) (*model.DownloadFile, error) {
	query := `SELECT id, download_id, name, path, size, downloaded, progress, status, error, engine_id, url, file_index FROM download_files WHERE engine_id = ?`
	row := r.db.QueryRowContext(ctx, query, engineID)

	var f model.DownloadFile
	var errStr, engID, url sql.NullString
	var fileIndex sql.NullInt64
	err := row.Scan(&f.ID, &f.DownloadID, &f.Name, &f.Path, &f.Size, &f.Downloaded, &f.Progress, &f.Status, &errStr, &engID, &url, &fileIndex)
	if err != nil {
		return nil, err
	}
	f.Error = errStr.String
	f.EngineID = engID.String
	f.URL = url.String
	if fileIndex.Valid {
		f.Index = int(fileIndex.Int64)
	}
	return &f, nil
}

// MarkAllFilesComplete marks all files for a download as complete
func (r *DownloadRepo) MarkAllFilesComplete(ctx context.Context, downloadID string) error {
	query := `
		UPDATE download_files SET
			status = ?, progress = 100, downloaded = size, updated_at = ?
		WHERE download_id = ?
	`
	_, err := r.db.ExecContext(ctx, query, model.StatusComplete, time.Now(), downloadID)
	return err
}

// GetFileByDownloadIDAndIndex finds a download file by its parent ID and index
func (r *DownloadRepo) GetFileByDownloadIDAndIndex(ctx context.Context, downloadID string, index int) (*model.DownloadFile, error) {
	query := `SELECT id, download_id, name, path, size, downloaded, progress, status, error, engine_id, url, file_index FROM download_files WHERE download_id = ? AND file_index = ?`
	row := r.db.QueryRowContext(ctx, query, downloadID, index)

	var f model.DownloadFile
	var errStr, engID, url sql.NullString
	var fileIndex sql.NullInt64
	err := row.Scan(&f.ID, &f.DownloadID, &f.Name, &f.Path, &f.Size, &f.Downloaded, &f.Progress, &f.Status, &errStr, &engID, &url, &fileIndex)
	if err != nil {
		return nil, err
	}
	f.Error = errStr.String
	f.EngineID = engID.String
	f.URL = url.String
	if fileIndex.Valid {
		f.Index = int(fileIndex.Int64)
	}
	return &f, nil
}

// DeleteFiles deletes all files for a download
func (r *DownloadRepo) DeleteFiles(ctx context.Context, downloadID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM download_files WHERE download_id = ?`, downloadID)
	return err
}
