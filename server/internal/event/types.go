package event

import (
	"time"
)

// Channel represents a typed event channel
type Channel string

const (
	// ChannelProgress handles high-frequency progress updates (download/upload)
	ChannelProgress Channel = "progress"
	// ChannelLifecycle handles state changes (created, started, paused, completed, error)
	ChannelLifecycle Channel = "lifecycle"
	// ChannelStats handles periodic global stats broadcasts
	ChannelStats Channel = "stats"
)

// EventType represents the type of event
type EventType string

const (
	// Download lifecycle events
	DownloadCreated   EventType = "download.created"
	DownloadStarted   EventType = "download.started"
	DownloadProgress  EventType = "download.progress"
	DownloadPaused    EventType = "download.paused"
	DownloadResumed   EventType = "download.resumed"
	DownloadCompleted EventType = "download.completed"
	DownloadError     EventType = "download.error"

	// Upload lifecycle events
	UploadStarted   EventType = "upload.started"
	UploadProgress  EventType = "upload.progress"
	UploadCompleted EventType = "upload.completed"
	UploadError     EventType = "upload.error"

	// System events
	SettingsUpdated EventType = "settings.updated"
	StatsUpdate     EventType = "stats"
)

// ETAUnknown represents an unknown ETA (when speed is 0)
const ETAUnknown = -1

// ProgressEvent represents a high-frequency progress update
type ProgressEvent struct {
	ID         string `json:"id"`
	Type       string `json:"type"` // "download" or "upload"
	Downloaded int64  `json:"downloaded,omitempty"`
	Uploaded   int64  `json:"uploaded,omitempty"`
	Size       int64  `json:"size"`
	Speed      int64  `json:"speed"`
	ETA        int    `json:"eta"` // -1 = unknown, 0 = done, >0 = seconds remaining
	Seeders    int    `json:"seeders,omitempty"`
	Peers      int    `json:"peers,omitempty"`
}

// LifecycleEvent represents a state change event
type LifecycleEvent struct {
	Type      EventType `json:"type"`
	ID        string    `json:"id"`
	Data      any       `json:"data,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// StatsEvent represents periodic stats broadcast
type StatsEvent struct {
	Speeds struct {
		Download int64 `json:"download"`
		Upload   int64 `json:"upload"`
	} `json:"speeds"`
	Tasks struct {
		Active    int `json:"active"`
		Uploading int `json:"uploading"`
		Waiting   int `json:"waiting"`
		Paused    int `json:"paused"`
		Completed int `json:"completed"`
		Failed    int `json:"failed"`
	} `json:"tasks"`
	Usage struct {
		TotalDownloaded   int64 `json:"totalDownloaded"`
		TotalUploaded     int64 `json:"totalUploaded"`
		SessionDownloaded int64 `json:"sessionDownloaded"`
		SessionUploaded   int64 `json:"sessionUploaded"`
	} `json:"usage"`
	System struct {
		DiskFree  uint64  `json:"diskFree"`
		DiskTotal uint64  `json:"diskTotal"`
		DiskUsage float64 `json:"diskUsage"`
		Uptime    int64   `json:"uptime"`
	} `json:"system"`
}

// Event is the legacy unified event type (for SSE compatibility)
type Event struct {
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// CalculateETA computes ETA from remaining bytes and speed
// Returns ETAUnknown (-1) when speed is 0
func CalculateETA(remaining, speed int64) int {
	if remaining <= 0 {
		return 0 // Done
	}
	if speed <= 0 {
		return ETAUnknown // Unknown
	}
	return int(remaining / speed)
}
