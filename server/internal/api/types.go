package api

type TaskOptions struct {
	MaxDownloadSpeed int64             `json:"maxDownloadSpeed"`
	Connections      int               `json:"connections"`
	Split            int               `json:"split"`
	ProxyURL         string            `json:"proxyUrl"`
	UploadRemote     string            `json:"uploadRemote"`
	Headers          map[string]string `json:"headers"`
}

// Downloads
type CreateDownloadRequest struct {
	URL         string      `json:"url" validate:"required" example:"http://example.com/file.zip"`
	Filename    string      `json:"filename" example:"my_file.zip"`
	DownloadDir string      `json:"downloadDir" example:"/downloads"`
	Destination string      `json:"destination" example:"gdrive:movies"`
	Options     TaskOptions `json:"options"`
}

type ListResponse struct {
	Data interface{} `json:"data"`
	Meta Meta        `json:"meta"`
}

type Meta struct {
	Total  int `json:"total" example:"100"`
	Limit  int `json:"limit" example:"10"`
	Offset int `json:"offset" example:"0"`
}

// Magnets
type CheckMagnetRequest struct {
	Magnet string `json:"magnet" validate:"required" example:"magnet:?xt=urn:btih:..."`
}

type CheckTorrentRequest struct {
	TorrentBase64 string `json:"torrentBase64" validate:"required"`
}

type MagnetFileRequest struct {
	ID    string `json:"id" example:"1"`
	Name  string `json:"name" example:"movie.mp4"`
	Path  string `json:"path" example:"Release/movie.mp4"`
	Size  int64  `json:"size" example:"1048576"`
	Link  string `json:"link,omitempty"`
	Index int    `json:"index" example:"1"`
}

type DownloadMagnetRequest struct {
	Magnet        string              `json:"magnet" example:"magnet:?xt=urn:btih:..."`
	TorrentBase64 string              `json:"torrentBase64"`
	Source        string              `json:"source" validate:"required" enums:"alldebrid,aria2"`
	MagnetID      string              `json:"magnetId"`
	Name          string              `json:"name" example:"My Movie"`
	SelectedFiles []string            `json:"selectedFiles" example:"1,2"`
	DownloadDir   string              `json:"downloadDir" example:"/downloads"`
	Destination   string              `json:"destination" example:"gdrive:movies"`
	Files         []MagnetFileRequest `json:"files"`
	Options       TaskOptions         `json:"options"`
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

// Files
type MkdirRequest struct {
	Path string `json:"path" validate:"required" example:"/movies"`
}

type DeleteFileRequest struct {
	Path string `json:"path" validate:"required" example:"/movies/old_file.txt"`
}

type FileOperationRequest struct {
	Op  string `json:"op" validate:"required" enums:"copy,move,rename" example:"copy"`
	Src string `json:"src" validate:"required" example:"/downloads/file.txt"`
	Dst string `json:"dst" validate:"required" example:"/movies/file.txt"`
}

// Providers
type ConfigureProviderRequest struct {
	Config  map[string]string `json:"config"`
	Enabled bool              `json:"enabled"`
}

type ResolveURLRequest struct {
	URL     string            `json:"url" validate:"required" example:"http://example.com/file.zip"`
	Headers map[string]string `json:"headers"`
}

// Remotes
type CreateRemoteRequest struct {
	Name   string            `json:"name" validate:"required" example:"my-gdrive"`
	Type   string            `json:"type" validate:"required" example:"drive"`
	Config map[string]string `json:"config"`
}
