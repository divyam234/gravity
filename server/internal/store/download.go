package store

import (
	"context"
	"encoding/json"
	"fmt"
	"gravity/internal/errors"
	"gravity/internal/model"
	"time"

	"gorm.io/gorm"
)

type DownloadRepo struct {
	db *gorm.DB
}

func NewDownloadRepo(db *gorm.DB) *DownloadRepo {
	return &DownloadRepo{db: db}
}

func (r *DownloadRepo) UpdateFile(ctx context.Context, downloadID string, f *model.DownloadFile) error {
	fileJSON, err := json.Marshal(f)
	if err != nil {
		return err
	}

	dbName := r.db.Dialector.Name()

	if dbName == "sqlite" {
		// SQLite JSON update
		// We find the index of the element with matching ID and replace it
		query := `
			UPDATE downloads 
			SET files = json_replace(files, 
				'$[' || (
					SELECT key 
					FROM json_each(files) 
					WHERE json_extract(value, '$.id') = ?
				) || ']', 
				json(?)
			) 
			WHERE id = ?`
		return r.db.WithContext(ctx).Exec(query, f.ID, string(fileJSON), downloadID).Error
	} else if dbName == "postgres" {
		// Postgres JSONB update
		// We reconstruct the array replacing the matching element
		query := `
			UPDATE downloads 
			SET files = (
				SELECT jsonb_agg(
					CASE 
						WHEN elem->>'id' = ? THEN ?::jsonb 
						ELSE elem 
					END
				)
				FROM jsonb_array_elements(files) elem
			)
			WHERE id = ?`
		return r.db.WithContext(ctx).Exec(query, f.ID, string(fileJSON), downloadID).Error
	}

	return fmt.Errorf("unsupported database for partial JSON update: %s", dbName)
}

func (r *DownloadRepo) Create(ctx context.Context, d *model.Download) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *DownloadRepo) Get(ctx context.Context, id string) (*model.Download, error) {
	var d model.Download
	err := r.db.WithContext(ctx).First(&d, "id = ?", id).Error
	return &d, err
}

func (r *DownloadRepo) GetByEngineID(ctx context.Context, engineID string) (*model.Download, error) {
	var d model.Download
	err := r.db.WithContext(ctx).First(&d, "engine_id = ?", engineID).Error
	return &d, err
}

func (r *DownloadRepo) GetByUploadJobID(ctx context.Context, jobID string) (*model.Download, error) {
	var d model.Download
	err := r.db.WithContext(ctx).First(&d, "upload_job_id = ?", jobID).Error
	return &d, err
}

func (r *DownloadRepo) List(ctx context.Context, status []string, limit, offset int, sortAsc bool) ([]*model.Download, int, error) {
	var downloads []*model.Download
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Download{})
	if len(status) > 0 {
		query = query.Where("status IN ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "priority ASC, created_at DESC"
	if sortAsc {
		order = "priority ASC, created_at ASC"
	}

	err := query.Order(order).Limit(limit).Offset(offset).Find(&downloads).Error
	return downloads, int(total), err
}

func (r *DownloadRepo) Update(ctx context.Context, d *model.Download) error {
	d.UpdatedAt = time.Now()

	// Optimistic Locking
	currentVersion := d.Version
	d.Version++

	// Use Where to ensure we are updating the version we read
	result := r.db.WithContext(ctx).Where("version = ?", currentVersion).Save(d)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New(errors.CodeInvalidOperation, fmt.Sprintf("concurrent modification of download %s (version mismatch)", d.ID))
	}
	return nil
}

func (r *DownloadRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Download{}, "id = ?", id).Error
}

func (r *DownloadRepo) Count(ctx context.Context, status []string) (int, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Download{})
	if len(status) > 0 {
		query = query.Where("status IN ?", status)
	}
	err := query.Count(&total).Error
	return int(total), err
}

func (r *DownloadRepo) GetStatusCounts(ctx context.Context) (map[model.DownloadStatus]int, error) {
	type result struct {
		Status model.DownloadStatus
		Count  int
	}
	var results []result
	err := r.db.WithContext(ctx).Model(&model.Download{}).Select("status, count(*) as count").Group("status").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[model.DownloadStatus]int)
	for _, res := range results {
		counts[res.Status] = res.Count
	}
	return counts, nil
}
