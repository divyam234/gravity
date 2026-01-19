package model

import (
	"time"
)

type DownloadStatus string

const (
	StatusPending     DownloadStatus = "pending"
	StatusDownloading DownloadStatus = "downloading"
	StatusPaused      DownloadStatus = "paused"
	StatusUploading   DownloadStatus = "uploading"
	StatusComplete    DownloadStatus = "complete"
	StatusError       DownloadStatus = "error"
)

type Download struct {
	ID             string         `json:"id"`
	URL            string         `json:"url"`
	ResolvedURL    string         `json:"resolvedUrl,omitempty"`
	Provider       string         `json:"provider,omitempty"`
	Status         DownloadStatus `json:"status"`
	Error          string         `json:"error,omitempty"`
	Filename       string         `json:"filename,omitempty"`
	Size           int64          `json:"size"`
	Downloaded     int64          `json:"downloaded"`
	Speed          int64          `json:"speed"`
	ETA            int            `json:"eta"`
	Destination    string         `json:"destination,omitempty"`
	UploadStatus   string         `json:"uploadStatus,omitempty"`
	UploadProgress int            `json:"uploadProgress"`
	UploadSpeed    int64          `json:"uploadSpeed"`
	Category       string         `json:"category,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	EngineID       string         `json:"-"`
	UploadJobID    string         `json:"-"`
	CreatedAt      time.Time      `json:"createdAt"`
	StartedAt      *time.Time     `json:"startedAt,omitempty"`
	CompletedAt    *time.Time     `json:"completedAt,omitempty"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}
