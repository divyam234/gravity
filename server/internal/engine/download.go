package engine

import (
	"context"
)

type Progress struct {
	Downloaded int64 `json:"downloaded"`
	Size       int64 `json:"size"`
	Speed      int64 `json:"speed"`
	ETA        int   `json:"eta"`
}

type DownloadStatus struct {
	ID          string `json:"id"`
	Status      string `json:"status"` // active, paused, complete, error
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	Dir         string `json:"dir"`
	Size        int64  `json:"size"`
	Downloaded  int64  `json:"downloaded"`
	Speed       int64  `json:"speed"`
	Connections int    `json:"connections"`
	Eta         int    `json:"eta"`
	Error       string `json:"error,omitempty"`
}

type DownloadOptions struct {
	Filename    string
	Dir         string
	Headers     map[string]string
	MaxSpeed    int64
	Connections int
}

type DownloadEngine interface {
	// Lifecycle
	Start(ctx context.Context) error
	Stop() error

	// Operations
	Add(ctx context.Context, url string, opts DownloadOptions) (string, error)
	Pause(ctx context.Context, id string) error
	Resume(ctx context.Context, id string) error
	Cancel(ctx context.Context, id string) error
	Remove(ctx context.Context, id string) error

	// Status
	Status(ctx context.Context, id string) (*DownloadStatus, error)
	List(ctx context.Context) ([]*DownloadStatus, error)

	// Configuration
	Configure(ctx context.Context, options map[string]string) error

	// Meta
	Version(ctx context.Context) (string, error)

	// Events
	OnProgress(handler func(id string, progress Progress))
	OnComplete(handler func(id string, filePath string))
	OnError(handler func(id string, err error))
}
