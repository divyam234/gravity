package store

import (
	"context"
	"time"

	"gravity/internal/model"

	"gorm.io/gorm"
)

type DownloadRepo struct {
	db *gorm.DB
}

func NewDownloadRepo(db *gorm.DB) *DownloadRepo {
	return &DownloadRepo{db: db}
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

	order := "created_at DESC"
	if sortAsc {
		order = "created_at ASC"
	}

	err := query.Order(order).Limit(limit).Offset(offset).Find(&downloads).Error
	return downloads, int(total), err
}

func (r *DownloadRepo) Update(ctx context.Context, d *model.Download) error {
	d.UpdatedAt = time.Now()
	// Updates() with struct only updates non-zero fields. 
	// To update all fields, we can use Save() or a map.
	// For Download manager, we usually want to persist everything.
	return r.db.WithContext(ctx).Save(d).Error
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

// CreateWithFiles creates a download with its associated files
func (r *DownloadRepo) CreateWithFiles(ctx context.Context, d *model.Download) error {
	// GORM handles associations automatically with Create
	return r.db.WithContext(ctx).Create(d).Error
}

// GetFiles returns all files for a download with pagination
func (r *DownloadRepo) GetFiles(ctx context.Context, downloadID string, limit, offset int) ([]model.DownloadFile, int, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.DownloadFile{}).Where("download_id = ?", downloadID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var files []model.DownloadFile
	err := r.db.WithContext(ctx).Where("download_id = ?", downloadID).
		Order("file_index ASC").
		Limit(limit).Offset(offset).
		Find(&files).Error

	return files, int(total), err
}

// UpdateFile updates a single download file
func (r *DownloadRepo) UpdateFile(ctx context.Context, f *model.DownloadFile) error {
	// Use Select to ensure we only update specific fields if needed, or just Save
	return r.db.WithContext(ctx).Save(f).Error
}

// UpdateFiles updates all files for a download
func (r *DownloadRepo) UpdateFiles(ctx context.Context, downloadID string, files []model.DownloadFile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, f := range files {
			if err := tx.Save(&f).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetFileByEngineID finds a download file by its aria2c GID
func (r *DownloadRepo) GetFileByEngineID(ctx context.Context, engineID string) (*model.DownloadFile, error) {
	var f model.DownloadFile
	err := r.db.WithContext(ctx).First(&f, "engine_id = ?", engineID).Error
	return &f, err
}

// MarkAllFilesComplete marks all files for a download as complete
func (r *DownloadRepo) MarkAllFilesComplete(ctx context.Context, downloadID string) error {
	return r.db.WithContext(ctx).Model(&model.DownloadFile{}).Where("download_id = ?", downloadID).Updates(map[string]interface{}{
		"status":     model.StatusComplete,
		"progress":   100,
		"downloaded": gorm.Expr("size"),
		"updated_at": time.Now(),
	}).Error
}

// GetFileByDownloadIDAndIndex finds a download file by its parent ID and index
func (r *DownloadRepo) GetFileByDownloadIDAndIndex(ctx context.Context, downloadID string, index int) (*model.DownloadFile, error) {
	var f model.DownloadFile
	err := r.db.WithContext(ctx).First(&f, "download_id = ? AND file_index = ?", downloadID, index).Error
	return &f, err
}

// DeleteFiles deletes all files for a download
func (r *DownloadRepo) DeleteFiles(ctx context.Context, downloadID string) error {
	return r.db.WithContext(ctx).Delete(&model.DownloadFile{}, "download_id = ?", downloadID).Error
}
