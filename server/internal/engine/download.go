package engine

import (
	"context"
)

type Progress struct {
	Downloaded int64 `json:"downloaded"`
	Size       int64 `json:"size"`
	Speed      int64 `json:"speed"`
	ETA        int   `json:"eta"`
	Seeders    int   `json:"seeders"`
	Peers      int   `json:"peers"`
}

type DownloadStatus struct {
	ID          string               `json:"id"`
	Status      string               `json:"status"` // active, paused, complete, error
	URL         string               `json:"url"`
	Filename    string               `json:"filename"`
	Dir         string               `json:"dir"`
	Size        int64                `json:"size"`
	Downloaded  int64                `json:"downloaded"`
	Speed       int64                `json:"speed"`
	Connections int                  `json:"connections"`
	Seeders     int                  `json:"seeders"`
	Peers       int                  `json:"peers"`
	Eta         int                  `json:"eta"`
	Error       string               `json:"error,omitempty"`
	Files       []DownloadFileStatus `json:"files,omitempty"`
}

type DownloadFileStatus struct {
	Index    int    `json:"index"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Selected bool   `json:"selected"`
}

type DownloadOptions struct {
	Filename      string
	Dir           string
	Headers       map[string]string
	MaxSpeed      int64
	Connections   int
	TorrentData   string // Base64 encoded .torrent
	SelectedFiles []int  // 1-indexed file numbers
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
	GetPeers(ctx context.Context, id string) ([]DownloadPeer, error)
	List(ctx context.Context) ([]*DownloadStatus, error)
	Sync(ctx context.Context) error

	// Configuration
	Configure(ctx context.Context, options map[string]string) error

	// Meta
	Version(ctx context.Context) (string, error)

	// Events
	OnProgress(handler func(id string, progress Progress))
	OnComplete(handler func(id string, filePath string))
	OnError(handler func(id string, err error))
}

type DownloadPeer struct {
	IP            string `json:"ip"`
	Port          string `json:"port"`
	DownloadSpeed int64  `json:"downloadSpeed"`
	UploadSpeed   int64  `json:"uploadSpeed"`
	IsSeeder      bool   `json:"isSeeder"`
}
