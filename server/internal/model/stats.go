package model

type Speeds struct {
	Download int64 `json:"download" validate:"required"`
	Upload   int64 `json:"upload" validate:"required"`
}

type TaskCounts struct {
	Active    int `json:"active" validate:"required"`
	Uploading int `json:"uploading" validate:"required"`
	Waiting   int `json:"waiting" validate:"required"`
	Paused    int `json:"paused" validate:"required"`
	Completed int `json:"completed" validate:"required"`
	Failed    int `json:"failed" validate:"required"`
}

type UsageStats struct {
	TotalDownloaded   int64 `json:"totalDownloaded" validate:"required"`
	TotalUploaded     int64 `json:"totalUploaded" validate:"required"`
	SessionDownloaded int64 `json:"sessionDownloaded" validate:"required"`
	SessionUploaded   int64 `json:"sessionUploaded" validate:"required"`
}

type SystemStats struct {
	DiskFree   uint64  `json:"diskFree" validate:"required"`
	DiskTotal  uint64  `json:"diskTotal" validate:"required"`
	DiskUsage  float64 `json:"diskUsage" validate:"required"` // Percentage
	Uptime     int64   `json:"uptime" validate:"required"`    // Seconds
}

type Stats struct {
	Speeds  Speeds      `json:"speeds" validate:"required"`
	Tasks   TaskCounts  `json:"tasks" validate:"required"`
	Usage   UsageStats  `json:"usage" validate:"required"`
	System  SystemStats `json:"system" validate:"required"`
}