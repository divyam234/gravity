package model

import (
	"time"
)

// DownloadStatus represents the current state of a download
// @Description active, waiting, paused, uploading, complete, error
// @enum active,waiting,paused,uploading,complete,error
type DownloadStatus string

const (
	StatusActive     DownloadStatus = "active"
	StatusWaiting    DownloadStatus = "waiting"
	StatusPaused     DownloadStatus = "paused"
	StatusUploading  DownloadStatus = "uploading"
	StatusComplete   DownloadStatus = "complete"
	StatusError      DownloadStatus = "error"
	StatusAllocating DownloadStatus = "allocating"
	StatusProcessing DownloadStatus = "processing"
	StatusResolving  DownloadStatus = "resolving"
)

type ExecutionMode string

const (
	ExecutionModeDirect      ExecutionMode = "direct"       // Standard URL -> engine.Add()
	ExecutionModeMagnet      ExecutionMode = "magnet"       // Magnet/torrent -> engine.AddMagnetWithSelection()
	ExecutionModeDebridFiles ExecutionMode = "debrid-files" // Cached debrid -> parallel file downloads
)

type Proxy struct {
	URL  string `json:"url"`
	Type string `json:"type" enums:"all,downloads,uploads,magnets"`
}

type Download struct {
	ID            string            `json:"id" example:"d_a1b2c3d4" gorm:"primaryKey"  binding:"required"`
	URL           string            `json:"url" example:"http://example.com/file.zip"  binding:"required"`
	ResolvedURL   string            `json:"resolvedUrl,omitempty"`
	Provider      string            `json:"provider,omitempty"`
	Engine        string            `json:"engine,omitempty" enums:"aria2,native"`
	ExecutionMode ExecutionMode     `json:"executionMode,omitempty"`
	Status        DownloadStatus    `json:"status" example:"active" enums:"active,waiting,paused,uploading,complete,error"  binding:"required"`
	Error         string            `json:"error,omitempty"`
	Filename      string            `json:"filename" binding:"required"`
	Dir           string            `json:"dir" binding:"required"`
	Destination   string            `json:"destination,omitempty"`
	UploadStatus  string            `json:"uploadStatus,omitempty" enums:"idle,running,complete,error"`
	Size          int64             `json:"size" example:"10485760" binding:"required"`
	Proxies       []Proxy           `json:"proxies" gorm:"serializer:json"`
	RemoveLocal   *bool             `json:"removeLocal,omitempty"`
	Downloaded    int64             `json:"downloaded" example:"5242880" binding:"required"`
	EngineID      string            `json:"-" gorm:"column:engine_id;index"`
	UploadJobID   string            `json:"-" gorm:"column:upload_job_id;index"`
	Headers       map[string]string `json:"headers,omitempty" gorm:"serializer:json"`
	CreatedAt     time.Time         `json:"createdAt"`
	StartedAt     *time.Time        `json:"startedAt,omitempty"`
	CompletedAt   *time.Time        `json:"completedAt,omitempty"`
	UpdatedAt     time.Time         `json:"updatedAt"`

	IsMagnet      bool           `json:"isMagnet"`
	MagnetHash    string         `json:"magnetHash,omitempty"`
	TorrentData   string         `json:"torrentData,omitempty"`
	SelectedFiles []int          `json:"-" gorm:"serializer:json"`
	Files         []DownloadFile `json:"files" gorm:"serializer:json"`

	// Priority and Retry
	Priority    int        `json:"priority" gorm:"default:5"` // 1 (highest) to 10 (lowest)
	RetryCount  int        `json:"retryCount" gorm:"default:0"`
	NextRetryAt *time.Time `json:"nextRetryAt,omitempty"`
	MaxRetries  int        `json:"maxRetries" gorm:"default:3"`

	//Not Saved in DB
	Split          *int   `json:"split" gorm:"-"`
	UploadProgress int    `json:"uploadProgress" example:"50" gorm:"-"`
	UploadSpeed    int64  `json:"uploadSpeed" example:"512000" gorm:"-"`
	Speed          int64  `json:"speed" example:"1024000" gorm:"-" binding:"required"`
	ETA            int    `json:"eta" example:"10"  gorm:"-" binding:"required"`
	Seeders        int    `json:"seeders" gorm:"-"`
	Peers          int    `json:"peers" gorm:"-"`
	PeerDetails    []Peer `json:"peerDetails" gorm:"-"`
}

// TransitionTo updates the download status if the transition is valid
func (d *Download) TransitionTo(status DownloadStatus) error {
	if err := ValidateTransition(d.Status, status); err != nil {
		return err
	}
	d.Status = status
	d.UpdatedAt = time.Now()
	return nil
}

// Peer represents a network peer in a BitTorrent swarm
type Peer struct {
	IP            string `json:"ip" validate:"required" binding:"required"`
	Port          string `json:"port" validate:"required" binding:"required"`
	DownloadSpeed int64  `json:"downloadSpeed" validate:"required" binding:"required"`
	UploadSpeed   int64  `json:"uploadSpeed" validate:"required" binding:"required"`
	IsSeeder      bool   `json:"isSeeder" validate:"required" binding:"required"`
}

// DownloadFile represents an individual file within a multi-file download (magnet/torrent)
type DownloadFile struct {
	ID         string         `json:"id" gorm:"primaryKey"`
	Name       string         `json:"name"`
	Path       string         `json:"path"` // relative path: "ubuntu/iso/file.iso"
	Size       int64          `json:"size"`
	Downloaded int64          `json:"downloaded"`
	Progress   int            `json:"progress"`
	Status     DownloadStatus `json:"status"`
	Error      string         `json:"error,omitempty"`
	URL        string         `json:"-"`
	Index      int            `json:"-" gorm:"column:file_index"` // 1-indexed file number for aria2c --select-file
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

// TransitionTo updates the file status if the transition is valid
func (f *DownloadFile) TransitionTo(status DownloadStatus) error {
	if err := ValidateTransition(f.Status, status); err != nil {
		return err
	}
	f.Status = status
	f.UpdatedAt = time.Now()
	return nil
}
