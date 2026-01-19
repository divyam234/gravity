package event

import (
	"time"
)

type EventType string

const (
	DownloadCreated   EventType = "download.created"
	DownloadStarted   EventType = "download.started"
	DownloadProgress  EventType = "download.progress"
	DownloadPaused    EventType = "download.paused"
	DownloadResumed   EventType = "download.resumed"
	DownloadCompleted EventType = "download.completed"
	DownloadError     EventType = "download.error"

	UploadStarted   EventType = "upload.started"
	UploadProgress  EventType = "upload.progress"
	UploadCompleted EventType = "upload.completed"
	UploadError     EventType = "upload.error"

	StatsUpdate EventType = "stats"
)

type Event struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}
