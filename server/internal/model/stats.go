package model

type ActiveStats struct {
	Downloads     int   `json:"downloads"`
	DownloadSpeed int64 `json:"downloadSpeed"`
	Uploads       int   `json:"uploads"`
	UploadSpeed   int64 `json:"uploadSpeed"`
}

type QueueStats struct {
	Pending int `json:"pending"`
	Paused  int `json:"paused"`
}

type TotalStats struct {
	TotalDownloaded    int64 `json:"totalDownloaded"`
	TotalUploaded      int64 `json:"totalUploaded"`
	TasksFinished      int64 `json:"tasksFinished"`      // Current count of completed tasks in DB
	TasksFailed        int64 `json:"tasksFailed"`        // Current count of failed tasks in DB
}

type Stats struct {
	Active ActiveStats `json:"active"`
	Queue  QueueStats  `json:"queue"`
	Totals TotalStats  `json:"totals"`
}
