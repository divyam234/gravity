package model

import (
	"time"
)

type DownloadStatus string

const (
	StatusActive    DownloadStatus = "active"
	StatusWaiting   DownloadStatus = "waiting"
	StatusPaused    DownloadStatus = "paused"
	StatusUploading DownloadStatus = "uploading"
	StatusComplete  DownloadStatus = "complete"
	StatusError     DownloadStatus = "error"
)

type Download struct {
	ID             string         `json:"id"`
	URL            string         `json:"url"`
	ResolvedURL    string         `json:"resolvedUrl,omitempty"`
	Provider       string         `json:"provider,omitempty"`
	Status         DownloadStatus `json:"status"`
	Error          string         `json:"error,omitempty"`
	Filename       string         `json:"filename,omitempty"`
	LocalPath      string         `json:"localPath,omitempty"`
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

	// Multi-file support for magnets/torrents
	IsMagnet      bool           `json:"isMagnet,omitempty"`
	MagnetHash    string         `json:"magnetHash,omitempty"`
	MagnetSource  string         `json:"magnetSource,omitempty"` // "alldebrid" or "aria2"
	MagnetID      string         `json:"-"`                      // AllDebrid magnet ID
	Files         []DownloadFile `json:"files,omitempty"`
	TotalFiles    int            `json:"totalFiles,omitempty"`
	FilesComplete int            `json:"filesComplete,omitempty"`
}

// DownloadFile represents an individual file within a multi-file download (magnet/torrent)
type DownloadFile struct {
	ID         string         `json:"id"`
	DownloadID string         `json:"-"` // parent download ID
	Name       string         `json:"name"`
	Path       string         `json:"path"` // relative path: "ubuntu/iso/file.iso"
	Size       int64          `json:"size"`
	Downloaded int64          `json:"downloaded"`
	Progress   int            `json:"progress"` // 0-100
	Status     DownloadStatus `json:"status"`
	Error      string         `json:"error,omitempty"`
	EngineID   string         `json:"-"` // aria2c GID for this file
	URL        string         `json:"-"` // Download URL (AllDebrid direct link)
	Index      int            `json:"-"` // 1-indexed file number for aria2c --select-file
}
