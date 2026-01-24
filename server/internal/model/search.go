package model

import "time"

type IndexedFile struct {
	ID            string    `json:"id" gorm:"primaryKey"`
	Remote        string    `json:"remote" gorm:"index"`
	Path          string    `json:"path" gorm:"index"`
	Name          string    `json:"name" gorm:"column:filename;index"`
	Size          int64     `json:"size"`
	ModTime       time.Time `json:"modTime"`
	IsDir         bool      `json:"isDir"`
	LastIndexedAt time.Time `json:"lastIndexedAt"`
}
