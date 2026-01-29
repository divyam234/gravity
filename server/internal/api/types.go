package api

import (
	"gravity/internal/engine"
	"gravity/internal/model"
	"gravity/internal/provider"
	"time"
)

// Type aliases for Swag to handle generics with slices
type DownloadList []*model.Download
type ProviderList []model.ProviderSummary
type RemoteList []engine.Remote
type FileInfoList []engine.FileInfo
type IndexedFileList []model.IndexedFile
type RemoteIndexConfigList []model.RemoteIndexConfig

// Concrete response wrappers for Swagger (Flattened to avoid generated names)
// Only include fields that are actually used in the response.

type DownloadListResponse struct {
	Data DownloadList `json:"data" binding:"required"`
	Meta *Meta        `json:"meta,omitempty"`
}

type DownloadResponse struct {
	Data *model.Download `json:"data" binding:"required"`
}

type ProviderListResponse struct {
	Data ProviderList `json:"data" binding:"required"`
}

type AccountInfoResponse struct {
	Data *model.AccountInfo `json:"data" binding:"required"`
}

type ProviderHostsResponse struct {
	Data ProviderHosts `json:"data" binding:"required"`
}

type ProviderSchemaResponse struct {
	Data ProviderSchema `json:"data" binding:"required"`
}

type ResolveResultResponse struct {
	Data ResolveResult `json:"data" binding:"required"`
}

type RemoteListResponse struct {
	Data RemoteList `json:"data" binding:"required"`
}

type FileInfoListResponse struct {
	Data FileInfoList `json:"data" binding:"required"`
}

type FileOperationResponse struct {
	Data FileOperation `json:"data" binding:"required"`
}

type IndexedFileListResponse struct {
	Data IndexedFileList `json:"data" binding:"required"`
	Meta *Meta           `json:"meta,omitempty"`
}

type RemoteIndexConfigListResponse struct {
	Data RemoteIndexConfigList `json:"data" binding:"required"`
}

type SystemVersionResponse struct {
	Data SystemVersion `json:"data" binding:"required"`
}

type SettingsResponse struct {
	Data *model.Settings `json:"data" binding:"required"`
}

type SettingsStatusResponse struct {
	Data SettingsStatus `json:"data" binding:"required"`
}

type SettingsStatusDownloads struct {
	Configured bool `json:"configured" binding:"required"`
}

type SettingsStatusCloud struct {
	RemoteCount int `json:"remoteCount" binding:"required"`
}

type SettingsStatusPremium struct {
	Providers []*model.Provider `json:"providers" binding:"required"`
}

type SettingsStatus struct {
	Downloads SettingsStatusDownloads `json:"downloads" binding:"required"`
	Cloud     SettingsStatusCloud     `json:"cloud" binding:"required"`
	Premium   SettingsStatusPremium   `json:"premium" binding:"required"`
}

type StatsResponse struct {
	Data *model.Stats `json:"data" binding:"required"`
}

// SSE Event Types for Documentation
// These help generate strictly typed TypeScript unions for the EventSource

type ProgressEventData struct {
	ID         string `json:"id" validate:"required" binding:"required"`
	Downloaded int64  `json:"downloaded" validate:"required" binding:"required"`
	Uploaded   int64  `json:"uploaded" validate:"required" binding:"required"`
	Size       int64  `json:"size" validate:"required" binding:"required"`
	Speed      int64  `json:"speed" validate:"required" binding:"required"`
	ETA        int    `json:"eta" validate:"required" binding:"required"`
	Seeders    int    `json:"seeders"`
	Peers      int    `json:"peers"`
}

// EventResponse represents a unified SSE event wrapper
type EventResponse struct {
	Type      string    `json:"type" example:"download.progress" binding:"required" enums:"download.created,download.started,download.progress,download.paused,download.resumed,download.completed,download.error,upload.started,upload.progress,upload.completed,upload.error,settings.updated,stats"`
	Timestamp time.Time `json:"timestamp" binding:"required"`
	// Data payload varies by event type
	Data any `json:"data" binding:"required" swaggertype:"primitive,object" oneOf:"ProgressEventData,model.Stats,model.Download"`
}

type Meta struct {
	Total  int `json:"total" example:"100" binding:"required"`
	Limit  int `json:"limit" example:"10" binding:"required"`
	Offset int `json:"offset" example:"0" binding:"required"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"error message" binding:"required"`
	Code  int    `json:"code" example:"500" binding:"required"`
}

type Proxy struct {
	URL  string `json:"url"`
	Type string `json:"type"`
}

// Downloads
type CreateDownloadRequest struct {
	URL         string            `json:"url" validate:"required" binding:"required" example:"http://example.com/file.zip"`
	Filename    string            `json:"filename" example:"my_file.zip"`
	Dir         string            `json:"dir" example:"/downloads"`
	Destination string            `json:"destination" example:"gdrive:movies"`
	Provider    string            `json:"provider"`
	Engine      string            `json:"engine" enums:"native,aria2"`
	Split       *int              `json:"split"`
	Proxies     []Proxy           `json:"proxies"`
	RemoveLocal *bool             `json:"removeLocal"`
	Headers     map[string]string `json:"headers"`

	//Fields For magnets
	TorrentData   string               `json:"torrentData"`
	Hash          string               `json:"hash"`
	SelectedFiles []int                `json:"selectedFiles" example:"1,2"`
	Files         []model.DownloadFile `json:"files"`
}

// Search
type UpdateConfigRequest struct {
	Interval           int    `json:"interval" binding:"required"`
	ExcludedPatterns   string `json:"excludedPatterns"`
	IncludedExtensions string `json:"includedExtensions"`
	MinSizeBytes       int64  `json:"minSizeBytes" binding:"required"`
}

type BatchUpdateConfigRequest struct {
	Configs map[string]UpdateConfigRequest `json:"configs" binding:"required"`
}

// Files
type MkdirRequest struct {
	Path string `json:"path" validate:"required" binding:"required" example:"/movies"`
}

type DeleteFileRequest struct {
	Path string `json:"path" validate:"required" binding:"required" example:"/movies/old_file.txt"`
}

type FileOperationRequest struct {
	Op  string `json:"op" validate:"required" binding:"required" enums:"copy,move,rename" example:"copy"`
	Src string `json:"src" validate:"required" example:"/downloads/file.txt"`
	Dst string `json:"dst" validate:"required" example:"/movies/file.txt"`
}

type FileOperation struct {
	JobID string `json:"jobId" binding:"required"`
}

// Providers
type ConfigureProviderRequest struct {
	Config  map[string]string `json:"config"`
	Enabled bool              `json:"enabled" binding:"required"`
}

type ResolveURLRequest struct {
	URL           string            `json:"url" validate:"required" binding:"required" example:"http://example.com/file.zip"`
	Headers       map[string]string `json:"headers"`
	TorrentBase64 string            `json:"torrentBase64"`
}

type ProviderHosts struct {
	Hosts []string `json:"hosts" binding:"required"`
}

type ProviderSchema struct {
	Name         string                 `json:"name" binding:"required"`
	ConfigSchema []provider.ConfigField `json:"configSchema" binding:"required"`
}

type ResolveResult struct {
	URL      string                  `json:"url" binding:"required"`
	Provider string                  `json:"provider" binding:"required"`
	Result   *provider.ResolveResult `json:"result" binding:"required"`
}

// Remotes
type CreateRemoteRequest struct {
	Name   string            `json:"name" validate:"required" binding:"required" example:"my-gdrive"`
	Type   string            `json:"type" validate:"required" binding:"required" example:"drive"`
	Config map[string]string `json:"config"`
}

// System
type SystemVersion struct {
	Version string `json:"version" binding:"required"`
	Aria2   string `json:"aria2" binding:"required"`
	Native  string `json:"native" binding:"required"`
	Rclone  string `json:"rclone" binding:"required"`
}
