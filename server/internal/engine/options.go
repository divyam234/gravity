package engine

import (
	"fmt"
	"gravity/internal/model"
	"strings"
	"time"
)

// DownloadOptions represents all possible download configuration options
// All pointer fields are optional - nil means "use global setting"
type DownloadOptions struct {
	// Core identification
	ID       string `json:"id,omitempty"`
	Filename string `json:"filename,omitempty"`
	URL      string `json:"url,omitempty"`

	// Paths
	DownloadDir string `json:"downloadDir,omitempty"` // Local download directory
	Destination string `json:"destination,omitempty"` // Remote upload destination (rclone format)

	// HTTP headers and metadata
	Headers     map[string]string `json:"headers,omitempty"`
	ContentType string            `json:"contentType,omitempty"`
	Referer     *string           `json:"referer,omitempty"`
	UserAgent   *string           `json:"userAgent,omitempty"`

	// Multi-file support (torrents/magnets)
	TorrentData   string `json:"torrentData,omitempty"`   // Base64 encoded
	MagnetHash    string `json:"magnetHash,omitempty"`    // For magnet links
	SelectedFiles []int  `json:"selectedFiles,omitempty"` // 1-indexed file numbers

	// Size (if known upfront)
	Size int64 `json:"size,omitempty"`

	// === All DownloadSettings fields (overridable per-download) ===

	// Connection settings
	Split                  *int  `json:"split,omitempty"`                  // File splitting count (1-16)
	MaxConnectionPerServer *int  `json:"maxConnectionPerServer,omitempty"` // Connections per server
	ConnectTimeout         *int  `json:"connectTimeout,omitempty"`         // Seconds
	MaxTries               *int  `json:"maxTries,omitempty"`               // Retry attempts
	CheckCertificate       *bool `json:"checkCertificate,omitempty"`       // SSL verification

	// Bandwidth limits (string format: "10M", "1G", "0" for unlimited)
	MaxDownloadSpeed *string `json:"maxDownloadSpeed,omitempty"`
	MaxUploadSpeed   *string `json:"maxUploadSpeed,omitempty"`
	LowestSpeedLimit *string `json:"lowestSpeedLimit,omitempty"` // Minimum acceptable speed

	// Performance
	DiskCache        *string `json:"diskCache,omitempty"`        // Buffer size (e.g., "32M")
	MinSplitSize     *string `json:"minSplitSize,omitempty"`     // Minimum file size to split
	PreAllocateSpace *bool   `json:"preAllocateSpace,omitempty"` // Pre-allocate disk space

	// Upload settings
	AutoUpload        *bool `json:"autoUpload,omitempty"`        // Auto-upload when complete
	RemoveLocal       *bool `json:"removeLocal,omitempty"`       // Remove local file after upload
	ConcurrentUploads *int  `json:"concurrentUploads,omitempty"` // Parallel uploads

	// Proxies
	Proxies []model.Proxy `json:"proxies,omitempty"`

	// Engine selection
	Engine string `json:"engine,omitempty"`
}

// EffectiveOptions holds the resolved/final values after merging with global settings
type EffectiveOptions struct {
	DownloadOptions

	// Non-overrideable computed fields
	LocalPath    string    // Fully resolved local download path
	RemotePath   string    // Normalized remote path
	ResolvedAt   time.Time // Track when options were resolved
	SettingsHash string    // Detect if settings changed
}

// OptionResolver handles merging per-download options with global settings
type OptionResolver struct {
	settings *model.Settings
}

// NewOptionResolver creates a new resolver with global settings
func NewOptionResolver(settings *model.Settings) *OptionResolver {
	return &OptionResolver{
		settings: settings,
	}
}

// FromModel creates DownloadOptions from a stored Download model
// This maps only what is present in the DB, without applying defaults.
func FromModel(d *model.Download) DownloadOptions {
	opts := DownloadOptions{
		ID:            d.ID,
		Filename:      d.Filename,
		URL:           d.URL,
		DownloadDir:   d.Dir,
		Destination:   d.Destination,
		Headers:       d.Headers,
		TorrentData:   d.TorrentData,
		MagnetHash:    d.MagnetHash,
		SelectedFiles: d.SelectedFiles,
		Size:          d.Size,
		Engine:        d.Engine,
		Split:         d.Split,
		RemoveLocal:   d.RemoveLocal,

		// Per-download overrides
		MaxDownloadSpeed: d.MaxDownloadSpeed,
		ConnectTimeout:   d.ConnectTimeout,
		MaxTries:         d.MaxTries,
	}

	if len(d.Proxies) > 0 {
		opts.Proxies = make([]model.Proxy, len(d.Proxies))
		copy(opts.Proxies, d.Proxies)
	}

	return opts
}

// Resolve merges per-download options with global settings
// Per-download options take precedence over global settings
func (r *OptionResolver) Resolve(opts DownloadOptions) EffectiveOptions {
	if r.settings == nil {
		r.settings = model.DefaultSettings()
	}

	ds := r.settings.Download
	us := r.settings.Upload

	effective := EffectiveOptions{
		DownloadOptions: DownloadOptions{
			// Copy provided options
			ID:            opts.ID,
			Filename:      opts.Filename,
			URL:           opts.URL,
			DownloadDir:   opts.DownloadDir,
			Destination:   opts.Destination,
			Headers:       opts.Headers,
			ContentType:   opts.ContentType,
			TorrentData:   opts.TorrentData,
			MagnetHash:    opts.MagnetHash,
			SelectedFiles: opts.SelectedFiles,
			Size:          opts.Size,
			Engine:        opts.Engine,

			// Resolve all overrideable fields
			Split:                  derefInt(opts.Split, ds.Split, 8),
			MaxConnectionPerServer: derefInt(opts.MaxConnectionPerServer, ds.MaxConnectionPerServer, 8),
			ConnectTimeout:         derefInt(opts.ConnectTimeout, ds.ConnectTimeout, 60),
			MaxTries:               derefInt(opts.MaxTries, ds.MaxTries, 5),
			CheckCertificate:       derefBool(opts.CheckCertificate, ds.CheckCertificate, true),

			MaxDownloadSpeed: derefString(opts.MaxDownloadSpeed, ds.MaxDownloadSpeed, "0"),
			MaxUploadSpeed:   derefString(opts.MaxUploadSpeed, ds.MaxUploadSpeed, "0"),
			LowestSpeedLimit: derefString(opts.LowestSpeedLimit, ds.LowestSpeedLimit, "0"),

			DiskCache:        derefString(opts.DiskCache, ds.DiskCache, "32M"),
			MinSplitSize:     derefString(opts.MinSplitSize, ds.MinSplitSize, "1M"),
			PreAllocateSpace: derefBool(opts.PreAllocateSpace, ds.PreAllocateSpace, false),

			AutoUpload:        derefBool(opts.AutoUpload, us.AutoUpload, false),
			RemoveLocal:       derefBool(opts.RemoveLocal, us.RemoveLocal, false),
			ConcurrentUploads: derefInt(opts.ConcurrentUploads, us.ConcurrentUploads, 1),

			// Identity
			Referer:   derefString(opts.Referer, "", ""),
			UserAgent: derefString(opts.UserAgent, ds.UserAgent, ""),

			// Proxies
			Proxies: opts.Proxies,
		},
	}

	// Resolve local path
	if effective.DownloadDir == "" {
		effective.LocalPath = ds.DownloadDir
	} else {
		effective.LocalPath = effective.DownloadDir
	}

	// Normalize remote path
	if effective.Destination != "" && !strings.Contains(effective.Destination, ":") {
		// Convert bare path to local rclone format
		effective.RemotePath = "local:" + effective.Destination
	} else {
		effective.RemotePath = effective.Destination
	}

	return effective
}

// Validate checks if the resolved options are valid
func (r *OptionResolver) Validate(opts EffectiveOptions) error {
	return opts.Validate()
}

func (e *EffectiveOptions) Validate() error {
	if e.URL == "" && e.TorrentData == "" && e.MagnetHash == "" {
		return fmt.Errorf("URL, TorrentData, or MagnetHash is required")
	}

	if e.LocalPath == "" {
		return fmt.Errorf("download directory is required")
	}

	if e.Split != nil && (*e.Split < 1 || *e.Split > 32) {
		return fmt.Errorf("split must be between 1 and 32")
	}

	if e.ConnectTimeout != nil && *e.ConnectTimeout < 1 {
		return fmt.Errorf("connect timeout must be at least 1 second")
	}

	if e.MaxTries != nil && *e.MaxTries < 0 {
		return fmt.Errorf("max tries cannot be negative")
	}

	return nil
}

// UpdateSettings updates the global settings used for resolution
func (r *OptionResolver) UpdateSettings(settings *model.Settings) {
	r.settings = settings
}

// Helper functions for dereferencing with defaults

func derefInt(override *int, global int, defaultVal int) *int {
	if override != nil {
		return override
	}
	if global != 0 {
		return &global
	}
	return &defaultVal
}

func derefBool(override *bool, global bool, defaultVal bool) *bool {
	if override != nil {
		return override
	}
	return &global
}

func derefString(override *string, global string, defaultVal string) *string {
	if override != nil {
		return override
	}
	if global != "" {
		return &global
	}
	return &defaultVal
}
