package store

import (
	"context"
	"strings"
	"time"

	"gravity/internal/logger"
	"gravity/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"go.uber.org/zap"
)

type SearchRepo struct {
	db *gorm.DB
}

func NewSearchRepo(db *gorm.DB) *SearchRepo {
	return &SearchRepo{db: db}
}

func (r *SearchRepo) SaveFiles(ctx context.Context, remote string, files []model.IndexedFile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("indexed_files").Where("remote = ?", remote).Delete(nil).Error; err != nil {
			return err
		}

		if len(files) == 0 {
			return nil
		}

		// Batch insert
		return tx.Table("indexed_files").Create(&files).Error
	})
}

func (r *SearchRepo) Search(ctx context.Context, query string, limit, offset int) ([]model.IndexedFile, int, error) {
	dbType := r.db.Dialector.Name()
	var results []model.IndexedFile
	var total int64

	if dbType == "sqlite" {
		// Check if FTS table exists
		var tableName string
		r.db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name='files_search'").Scan(&tableName)
		
		if tableName == "files_search" {
			ftsQuery := strings.ReplaceAll(query, "\"", "") + "*"
			
			countErr := r.db.WithContext(ctx).Table("indexed_files").
				Joins("JOIN files_search ON files_search.rowid = indexed_files.rowid").
				Where("files_search MATCH ?", ftsQuery).
				Count(&total).Error
			if countErr == nil {
				err := r.db.WithContext(ctx).Table("indexed_files").
					Select("indexed_files.* ").
					Joins("JOIN files_search ON files_search.rowid = indexed_files.rowid").
					Where("files_search MATCH ?", ftsQuery).
					Order("rank").
					Limit(limit).Offset(offset).
					Find(&results).Error
				
				return results, int(total), err
			}
			logger.L.Warn("SQLite FTS query failed, falling back to LIKE", zap.Error(countErr))
		}
	}

	if dbType == "postgres" {
		// PostgreSQL Full Text Search
		// websearch_to_tsquery provides a Google-like search syntax (quotes for phrases, - for exclusion)
		tsQuery := "websearch_to_tsquery('english', ?)"
		ftsCondition := "to_tsvector('english', filename || ' ' || path) @@ " + tsQuery

		countErr := r.db.WithContext(ctx).Table("indexed_files").
			Where(ftsCondition, query).
			Count(&total).Error
		if countErr != nil {
			return nil, 0, countErr
		}

		err := r.db.WithContext(ctx).Table("indexed_files").
			Where(ftsCondition, query).
			Order(gorm.Expr("ts_rank(to_tsvector('english', filename || ' ' || path), "+tsQuery+") DESC", query)).
			Limit(limit).Offset(offset).
			Find(&results).Error

		return results, int(total), err
	}

	// Fallback for others using LIKE
	likeQuery := "%" + query + "%"
	
	countErr := r.db.WithContext(ctx).Table("indexed_files").
		Where("filename LIKE ? OR path LIKE ?", likeQuery, likeQuery).
		Count(&total).Error
	if countErr != nil {
		return nil, 0, countErr
	}

	err := r.db.WithContext(ctx).Table("indexed_files").
		Where("filename LIKE ? OR path LIKE ?", likeQuery, likeQuery).
		Limit(limit).Offset(offset).
		Find(&results).Error

	return results, int(total), err
}

func (r *SearchRepo) GetConfigs(ctx context.Context) ([]model.RemoteIndexConfig, error) {
	var configs []model.RemoteIndexConfig
	err := r.db.WithContext(ctx).Table("remote_index_config").Find(&configs).Error
	return configs, err
}

func (r *SearchRepo) SaveConfig(ctx context.Context, c model.RemoteIndexConfig) error {
	return r.db.WithContext(ctx).Table("remote_index_config").Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "remote"}},
		UpdateAll: true,
	}).Create(&c).Error
}

func (r *SearchRepo) UpdateStatus(ctx context.Context, remote, status, errorMsg string) error {
	return r.db.WithContext(ctx).Table("remote_index_config").Where("remote = ?", remote).Updates(map[string]interface{}{
		"status":    status,
		"error_msg": errorMsg,
	}).Error
}

func (r *SearchRepo) UpdateLastIndexed(ctx context.Context, remote string) error {
	return r.db.WithContext(ctx).Table("remote_index_config").Where("remote = ?", remote).Updates(map[string]interface{}{
		"last_indexed_at": time.Now(),
		"status":          "idle",
		"error_msg":       "",
	}).Error
}
