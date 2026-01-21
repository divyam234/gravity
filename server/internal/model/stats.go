package model

type Speeds struct {
	Download int64 `json:"download"`
	Upload   int64 `json:"upload"`
}

type TaskCounts struct {
	Active    int `json:"active"`
	Uploading int `json:"uploading"`
	Waiting   int `json:"waiting"`
	Paused    int `json:"paused"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

type UsageStats struct {
	TotalDownloaded   int64 `json:"totalDownloaded"`
	TotalUploaded     int64 `json:"totalUploaded"`
	SessionDownloaded int64 `json:"sessionDownloaded"`
	SessionUploaded   int64 `json:"sessionUploaded"`
}

type SystemStats struct {
	DiskFree   uint64 `json:"diskFree"`
	DiskTotal  uint64 `json:"diskTotal"`
	DiskUsage  float64 `json:"diskUsage"` // Percentage
	Uptime     int64   `json:"uptime"`    // Seconds
}

type Stats struct {
	Speeds  Speeds      `json:"speeds"`
	Tasks   TaskCounts  `json:"tasks"`
	Usage   UsageStats  `json:"usage"`
	System  SystemStats `json:"system"`
}
