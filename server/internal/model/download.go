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
	ID               string            `json:"id" example:"d_a1b2c3d4" gorm:"primaryKey"`
	URL              string            `json:"url" example:"http://example.com/file.zip"`
	ResolvedURL      string            `json:"resolvedUrl,omitempty"`
	Provider         string            `json:"provider,omitempty"`
	Engine           string            `json:"engine,omitempty"`
	Options          TaskOptions       `json:"options" gorm:"serializer:json"`
	Status           DownloadStatus    `json:"status" example:"active" enums:"active,waiting,paused,uploading,complete,error"`
	Error            string            `json:"error,omitempty"`
	Filename         string            `json:"filename,omitempty"`
	DownloadDir      string            `json:"downloadDir,omitempty"`
	Size             int64             `json:"size" example:"10485760"`
	Downloaded       int64             `json:"downloaded" example:"5242880"`
	Speed            int64             `json:"speed" example:"1024000"`
	ETA              int               `json:"eta" example:"10"`
	Seeders          int               `json:"seeders,omitempty"`
	Peers            int               `json:"peers,omitempty"`
	MetadataFetching bool              `json:"metadataFetching,omitempty" gorm:"-"`
	Destination      string            `json:"destination,omitempty"`
	UploadStatus     string            `json:"uploadStatus,omitempty" enums:"idle,running,complete,error"`
	UploadProgress   int               `json:"uploadProgress" example:"50"`
	UploadSpeed      int64             `json:"uploadSpeed" example:"512000"`
	Category         string            `json:"category,omitempty"`
	Tags             []string          `json:"tags,omitempty" gorm:"serializer:json"`
	EngineID         string            `json:"-" gorm:"column:engine_id;index"`
	UploadJobID      string            `json:"-" gorm:"column:upload_job_id;index"`
	Headers          map[string]string `json:"-" gorm:"serializer:json"`
	CreatedAt        time.Time         `json:"createdAt"`
	StartedAt        *time.Time        `json:"startedAt,omitempty"`
	CompletedAt      *time.Time        `json:"completedAt,omitempty"`
	UpdatedAt        time.Time         `json:"updatedAt"`

	// Multi-file support for magnets/torrents
	IsMagnet      bool           `json:"isMagnet,omitempty"`
	MagnetHash    string         `json:"magnetHash,omitempty"`
	MagnetSource  string         `json:"magnetSource,omitempty" enums:"alldebrid,aria2"` // "alldebrid" or "aria2"
	MagnetID      string         `json:"-"`                                              // AllDebrid magnet ID
	TorrentData   string         `json:"-"`                                              // Base64 encoded .torrent
	SelectedFiles []int          `json:"-" gorm:"serializer:json"`
	Files         []DownloadFile `json:"files,omitempty" gorm:"foreignKey:DownloadID"`
	PeerDetails   []Peer         `json:"peerDetails,omitempty" gorm:"-"`
	TotalFiles    int            `json:"totalFiles,omitempty"`
	FilesComplete int            `json:"filesComplete,omitempty"`
}

// Peer represents a network peer in a BitTorrent swarm
type Peer struct {
	IP            string `json:"ip"`
	Port          string `json:"port"`
	DownloadSpeed int64  `json:"downloadSpeed"`
	UploadSpeed   int64  `json:"uploadSpeed"`
	IsSeeder      bool   `json:"isSeeder"`
}

type TaskOptions struct {
	MaxDownloadSpeed int64             `json:"maxDownloadSpeed"`
	Connections      int               `json:"connections"`
	Split            int               `json:"split"`
	ProxyURL         string            `json:"proxyUrl"`
	UploadRemote     string            `json:"uploadRemote"`
	Headers          map[string]string `json:"headers"`
}

// DownloadFile represents an individual file within a multi-file download (magnet/torrent)
type DownloadFile struct {
	ID         string         `json:"id" gorm:"primaryKey"`
	DownloadID string         `json:"-" gorm:"index"` // parent download ID
	Name       string         `json:"name"`
	Path       string         `json:"path"` // relative path: "ubuntu/iso/file.iso"
	Size       int64          `json:"size"`
	Downloaded int64          `json:"downloaded"`
	Progress   int            `json:"progress"` // 0-100
	Status     DownloadStatus `json:"status"`
	Error      string         `json:"error,omitempty"`
	EngineID   string         `json:"-" gorm:"index"`             // aria2c GID for this file
	URL        string         `json:"-"`                          // Download URL (AllDebrid direct link)
	Index      int            `json:"-" gorm:"column:file_index"` // 1-indexed file number for aria2c --select-file
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}
