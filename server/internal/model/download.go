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
)

type Download struct {
	ID               string            `json:"id" example:"d_a1b2c3d4" gorm:"primaryKey" validate:"required" binding:"required"`
	URL              string            `json:"url" example:"http://example.com/file.zip" validate:"required" binding:"required"`
	ResolvedURL      string            `json:"resolvedUrl,omitempty"`
	Provider         string            `json:"provider,omitempty"`
	Engine           string            `json:"engine,omitempty"`
	Status           DownloadStatus    `json:"status" example:"active" enums:"active,waiting,paused,uploading,complete,error" validate:"required" binding:"required"`
	Error            string            `json:"error,omitempty"`
	Filename         string            `json:"filename,omitempty"`
	DownloadDir      string            `json:"downloadDir,omitempty"`
	Size             int64             `json:"size" example:"10485760" validate:"required" binding:"required"`
	Downloaded       int64             `json:"downloaded" example:"5242880" validate:"required" binding:"required"`
	Speed            int64             `json:"speed" example:"1024000" validate:"required" binding:"required"`
	ETA              int               `json:"eta" example:"10" validate:"required" binding:"required"`
	Seeders          int               `json:"seeders"`
	Peers            int               `json:"peers"`
	MetadataFetching bool              `json:"metadataFetching" gorm:"-"`
	Destination      string            `json:"destination,omitempty"`
	UploadStatus     string            `json:"uploadStatus,omitempty" enums:"idle,running,complete,error"`
	UploadProgress   int               `json:"uploadProgress" example:"50" validate:"required" binding:"required"`
	UploadSpeed      int64             `json:"uploadSpeed" example:"512000" validate:"required" binding:"required"`
	Category         string            `json:"category,omitempty"`
	Tags             []string          `json:"tags" gorm:"serializer:json"`
	EngineID         string            `json:"-" gorm:"column:engine_id;index"`
	UploadJobID      string            `json:"-" gorm:"column:upload_job_id;index"`
	Headers          map[string]string `json:"-" gorm:"serializer:json"`
	CreatedAt        time.Time         `json:"createdAt" validate:"required" binding:"required"`
	StartedAt        *time.Time        `json:"startedAt,omitempty"`
	CompletedAt      *time.Time        `json:"completedAt,omitempty"`
	UpdatedAt        time.Time         `json:"updatedAt" validate:"required" binding:"required"`

	// Task Options - stored as JSON for flexibility
	// All download settings are stored here and can be overridden per-download
	Options TaskOptions `json:"options" gorm:"serializer:json"`

	// Multi-file support for magnets/torrents
	IsMagnet      bool           `json:"isMagnet"`
	MagnetHash    string         `json:"magnetHash,omitempty"`
	MagnetSource  string         `json:"magnetSource,omitempty" enums:"alldebrid,aria2"` // "alldebrid" or "aria2"
	MagnetID      string         `json:"-"`                                              // AllDebrid magnet ID
	TorrentData   string         `json:"-"`                                              // Base64 encoded .torrent
	SelectedFiles []int          `json:"-" gorm:"serializer:json"`
	Files         []DownloadFile `json:"files" gorm:"foreignKey:DownloadID" binding:"required"`
	PeerDetails   []Peer         `json:"peerDetails" gorm:"-"`
	TotalFiles    int            `json:"totalFiles"`
	FilesComplete int            `json:"filesComplete"`
}

// Peer represents a network peer in a BitTorrent swarm
type Peer struct {
	IP            string `json:"ip" validate:"required" binding:"required"`
	Port          string `json:"port" validate:"required" binding:"required"`
	DownloadSpeed int64  `json:"downloadSpeed" validate:"required" binding:"required"`
	UploadSpeed   int64  `json:"uploadSpeed" validate:"required" binding:"required"`
	IsSeeder      bool   `json:"isSeeder" validate:"required" binding:"required"`
}

type TaskOptions struct {
	DownloadDir string            `json:"downloadDir"` // Local save path
	Destination string            `json:"destination"` // Remote path (e.g., "gdrive:movies")
	Split       int               `json:"split"`       // File splitting count
	MaxTries    int               `json:"maxTries"`    // Retry attempts
	UserAgent   string            `json:"userAgent"`   // Custom user agent
	ProxyURL    string            `json:"proxyUrl"`    // Full proxy URL
	RemoveLocal *bool             `json:"removeLocal"` // Remove local file after upload (nullable)
	Headers     map[string]string `json:"headers"`     // Custom HTTP headers
}

// DownloadFile represents an individual file within a multi-file download (magnet/torrent)
type DownloadFile struct {
	ID         string         `json:"id" gorm:"primaryKey" validate:"required"`
	DownloadID string         `json:"-" gorm:"index"` // parent download ID
	Name       string         `json:"name" validate:"required"`
	Path       string         `json:"path" validate:"required"` // relative path: "ubuntu/iso/file.iso"
	Size       int64          `json:"size" validate:"required"`
	Downloaded int64          `json:"downloaded" validate:"required"`
	Progress   int            `json:"progress" validate:"required"` // 0-100
	Status     DownloadStatus `json:"status" validate:"required"`
	Error      string         `json:"error,omitempty"`
	EngineID   string         `json:"-" gorm:"index"`             // aria2c GID for this file
	URL        string         `json:"-"`                          // Download URL (AllDebrid direct link)
	Index      int            `json:"-" gorm:"column:file_index"` // 1-indexed file number for aria2c --select-file
	CreatedAt  time.Time      `json:"createdAt" validate:"required"`
	UpdatedAt  time.Time      `json:"updatedAt" validate:"required"`
}
