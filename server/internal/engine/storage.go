package engine

import (
	"context"
	"io"
	"time"

	"gravity/internal/model"
)

type FileType string

const (
	FileTypeFolder FileType = "folder"
	FileTypeFile   FileType = "file"
)

type FileInfo struct {
	Path     string    `json:"path"`         // Full virtual path
	Name     string    `json:"name"`         // Display name
	Size     int64     `json:"size"`         // Bytes
	MimeType string    `json:"mimeType"`     // Mime type
	ModTime  time.Time `json:"modTime"`      // Modification time
	Type     FileType  `json:"type"`         // "file" or "folder"
	IsDir    bool      `json:"isDir"`        // Helper boolean
	ID       string    `json:"id,omitempty"` // Optional ID for some systems
}

type Remote struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Connected bool   `json:"connected"`
}

type UploadProgress struct {
	Uploaded int64 `json:"uploaded"`
	Size     int64 `json:"size"`
	Speed    int64 `json:"speed"`
}

type UploadStatus struct {
	JobID    string `json:"jobId"`
	Status   string `json:"status"` // running, complete, error
	Src      string `json:"src"`
	Dst      string `json:"dst"`
	Size     int64  `json:"size"`
	Uploaded int64  `json:"uploaded"`
	Speed    int64  `json:"speed"`
	Error    string `json:"error,omitempty"`
}

type UploadOptions struct {
	DeleteAfter bool
	TrackingID  string // Download ID to use for progress tracking callbacks
	JobID       int64  // Custom job ID to use (if 0, one will be generated)
}

type GlobalStats struct {
	Speed           int64 `json:"speed"`
	ActiveTransfers int   `json:"activeTransfers"`
}

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type StorageEngine interface {
	// Virtual File System Operations
	List(ctx context.Context, virtualPath string) ([]FileInfo, error)
	Stat(ctx context.Context, virtualPath string) (*FileInfo, error)

	// Remotes
	ListRemotes(ctx context.Context) ([]Remote, error)

	// File Manipulation
	Mkdir(ctx context.Context, virtualPath string) error
	Delete(ctx context.Context, virtualPath string) error
	Rename(ctx context.Context, virtualPath, newName string) error

	// Data
	Open(ctx context.Context, virtualPath string) (ReadSeekCloser, error)
}

type UploadEngine interface {
	StorageEngine

	// Lifecycle
	Start(ctx context.Context) error
	Stop() error

	// Operations
	Upload(ctx context.Context, src, dst string, opts UploadOptions) (string, error)
	Cancel(ctx context.Context, jobID string) error
	Copy(ctx context.Context, srcPath, dstPath string) (string, error)
	Move(ctx context.Context, srcPath, dstPath string) (string, error)

	// Status
	Status(ctx context.Context, jobID string) (*UploadStatus, error)
	GetGlobalStats(ctx context.Context) (*GlobalStats, error)

	// Meta
	Version(ctx context.Context) (string, error)

	// Events
	OnProgress(handler func(jobID string, progress UploadProgress))
	OnComplete(handler func(jobID string))
	OnError(handler func(jobID string, err error))

	// Remotes Admin
	CreateRemote(ctx context.Context, name, rtype string, config map[string]string) error
	DeleteRemote(ctx context.Context, name string) error
	TestRemote(ctx context.Context, name string) error

	// Config
	Configure(ctx context.Context, settings *model.Settings) error
	Restart(ctx context.Context) error
}
