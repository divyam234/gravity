package api

// Downloads
type CreateDownloadRequest struct {
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	Destination string `json:"destination"`
}

type ListResponse struct {
	Data interface{} `json:"data"`
	Meta Meta        `json:"meta"`
}

type Meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Magnets
type CheckMagnetRequest struct {
	Magnet string `json:"magnet"`
}

type CheckTorrentRequest struct {
	TorrentBase64 string `json:"torrentBase64"`
}

type MagnetFileRequest struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	Link  string `json:"link"`
	Index int    `json:"index"`
}

type DownloadMagnetRequest struct {
	Magnet        string              `json:"magnet"`
	TorrentBase64 string              `json:"torrentBase64"`
	Source        string              `json:"source"`
	MagnetID      string              `json:"magnetId"`
	Name          string              `json:"name"`
	SelectedFiles []string            `json:"selectedFiles"`
	Destination   string              `json:"destination"`
	Files         []MagnetFileRequest `json:"files"`
}

// Search
type UpdateConfigRequest struct {
	Interval           int    `json:"interval"`
	ExcludedPatterns   string `json:"excludedPatterns"`
	IncludedExtensions string `json:"includedExtensions"`
	MinSizeBytes       int64  `json:"minSizeBytes"`
}

type BatchUpdateConfigRequest struct {
	Configs map[string]UpdateConfigRequest `json:"configs"`
}
