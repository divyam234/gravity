package engine

import (
	"context"
)

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

type UploadEngine interface {
	// Lifecycle
	Start(ctx context.Context) error
	Stop() error

	// Operations
	Upload(ctx context.Context, src, dst string, opts UploadOptions) (string, error)
	Cancel(ctx context.Context, jobID string) error

	// Status
	Status(ctx context.Context, jobID string) (*UploadStatus, error)
	GetGlobalStats(ctx context.Context) (*GlobalStats, error)

	// Meta
	Version(ctx context.Context) (string, error)

	// Events
	OnProgress(handler func(jobID string, progress UploadProgress))
	OnComplete(handler func(jobID string))
	OnError(handler func(jobID string, err error))

	// Remotes
	ListRemotes(ctx context.Context) ([]Remote, error)
	CreateRemote(ctx context.Context, name, rtype string, config map[string]string) error
	DeleteRemote(ctx context.Context, name string) error
	TestRemote(ctx context.Context, name string) error

	// Config
	Configure(ctx context.Context, options map[string]string) error
}
