package model

import (
	"os"
	"path/filepath"
	"time"
)

type Settings struct {
	ID         uint               `gorm:"primaryKey" json:"-"`
	Download   DownloadSettings   `gorm:"serializer:json" json:"download"`
	Upload     UploadSettings     `gorm:"serializer:json" json:"upload"`
	Network    NetworkSettings    `gorm:"serializer:json" json:"network"`
	Torrent    TorrentSettings    `gorm:"serializer:json" json:"torrent"`
	Vfs        VfsSettings        `gorm:"serializer:json" json:"vfs"`
	Advanced   AdvancedSettings   `gorm:"serializer:json" json:"advanced"`
	Automation AutomationSettings `gorm:"serializer:json" json:"automation"`
	Search     SearchSettings     `gorm:"serializer:json" json:"search"`
	UpdatedAt  time.Time          `json:"updatedAt"`
}

type SearchSettings struct {
	Configs []RemoteIndexConfig `json:"configs"`
}

type RemoteIndexConfig struct {
	Remote               string     `json:"remote"`
	AutoIndexIntervalMin int        `json:"autoIndexIntervalMin"`
	LastIndexedAt        *time.Time `json:"lastIndexedAt"`
	Status               string     `json:"status"` // idle, indexing, error
	ErrorMsg             string     `json:"errorMsg"`
	ExcludedPatterns     string     `json:"excludedPatterns"`
	IncludedExtensions   string     `json:"includedExtensions"`
	MinSizeBytes         int64      `json:"minSizeBytes"`
}

type DownloadSettings struct {
	DownloadDir            string `json:"downloadDir" validate:"required"`
	MaxConcurrentDownloads int    `json:"maxConcurrentDownloads" validate:"min=1,max=20"`

	// Engine Preferences
	PreferredEngine       string `json:"preferredEngine" enums:"aria2,native" default:"aria2"`
	PreferredMagnetEngine string `json:"preferredMagnetEngine" enums:"aria2,native" default:"aria2"`

	MaxDownloadSpeed       string `json:"maxDownloadSpeed" example:"0"` // 0 = unlimited
	MaxUploadSpeed         string `json:"maxUploadSpeed" example:"0"`
	MaxConnectionPerServer int    `json:"maxConnectionPerServer" validate:"min=1,max=16"`
	Split                  int    `json:"split" validate:"min=1,max=16"`
	UserAgent              string `json:"userAgent"`
	ConnectTimeout         int    `json:"connectTimeout" validate:"min=1"`
	MaxTries               int    `json:"maxTries" validate:"min=0"`
	CheckCertificate       bool   `json:"checkCertificate"`
	AutoResume             bool   `json:"autoResume"`

	// Professional Enhancements
	PreAllocateSpace bool   `json:"preAllocateSpace"`        // Prevent disk fragmentation
	DiskCache        string `json:"diskCache" example:"32M"` // Reduce disk I/O overhead
	MinSplitSize     string `json:"minSplitSize" example:"1M"`
	LowestSpeedLimit string `json:"lowestSpeedLimit" example:"0"`
}

type UploadSettings struct {
	DefaultRemote     string `json:"defaultRemote"`
	AutoUpload        bool   `json:"autoUpload"`
	RemoveLocal       bool   `json:"removeLocal"`
	ConcurrentUploads int    `json:"concurrentUploads" validate:"min=1"`

	// Professional Enhancements
	UploadBandwidth  string `json:"uploadBandwidth" example:"0"` // Limit specifically for uploads
	MaxRetryAttempts int    `json:"maxRetryAttempts" validate:"min=0"`
	ChunkSize        string `json:"chunkSize" example:"64M"`
}

type ProxyConfig struct {
	Enabled  bool   `json:"enabled"`
	URL      string `json:"url"` // scheme://host:port
	User     string `json:"user"`
	Password string `json:"password"`
}

type NetworkSettings struct {
	ProxyMode string `json:"proxyMode" enums:"global,granular" default:"global"`

	// Proxy Configurations
	GlobalProxy   ProxyConfig `json:"globalProxy"`
	MagnetProxy   ProxyConfig `json:"magnetProxy"`
	DownloadProxy ProxyConfig `json:"downloadProxy"`
	UploadProxy   ProxyConfig `json:"uploadProxy"`

	DNSOverHTTPS string `json:"dnsOverHttps" example:"https://cloudflare-dns.com/dns-query"`

	// Professional Enhancements
	InterfaceBinding string `json:"interfaceBinding" example:"eth0"` // Bind to specific NIC or VPN
	TCPPortRange     string `json:"tcpPortRange" example:"6881-6999"`
}

type TorrentSettings struct {
	SeedRatio  string `json:"seedRatio" example:"1.0"`
	SeedTime   int    `json:"seedTime" example:"60"` // Minutes
	ListenPort int    `json:"listenPort" validate:"min=1024,max=65535"`
	ForceSave  bool   `json:"forceSave"`
	EnablePex  bool   `json:"enablePex"`
	EnableDht  bool   `json:"enableDht"`
	EnableLpd  bool   `json:"enableLpd"`
	Encryption string `json:"encryption" enums:"forced,enabled,disabled"`
	MaxPeers   int    `json:"maxPeers" validate:"min=0"`
}

type VfsSettings struct {
	CacheMode          string `json:"cacheMode" enums:"off,minimal,writes,full"`
	CacheMaxSize       string `json:"cacheMaxSize" example:"10G"`
	CacheMaxAge        string `json:"cacheMaxAge" example:"24h"`
	WriteBack          string `json:"writeBack" example:"5s"`
	ReadChunkSize      string `json:"readChunkSize" example:"128M"`
	ReadChunkSizeLimit string `json:"readChunkSizeLimit" example:"off"`
	ReadAhead          string `json:"readAhead" example:"128M"`
	DirCacheTime       string `json:"dirCacheTime" example:"5m"`
	PollInterval       string `json:"pollInterval" example:"1m"`
	ReadChunkStreams   int    `json:"readChunkStreams" validate:"min=0"`
}

type AutomationSettings struct {
	// Scheduling
	ScheduleEnabled bool           `json:"scheduleEnabled"`
	Rules           []ScheduleRule `json:"rules"`

	// Post-Processing
	OnCompleteAction string `json:"onCompleteAction" enums:"none,shutdown,sleep,run_script"`
	ScriptPath       string `json:"scriptPath" example:"/scripts/notify.sh"`

	// Auto-Organization
	Categories []Category `json:"categories"`
}

type Category struct {
	ID         string   `json:"id" example:"cat_1"`
	Name       string   `json:"name" example:"Movies"`
	Path       string   `json:"path" example:"/downloads/movies"` // Sub-folder or absolute path
	Extensions []string `json:"extensions" example:"mp4,mkv,avi"`
	Icon       string   `json:"icon" example:"video"`
	IsDefault  bool     `json:"isDefault"`
}

type ScheduleRule struct {
	ID        string `json:"id" example:"rule_1"`
	Enabled   bool   `json:"enabled"`
	Label     string `json:"label" example:"Work Hours"`
	Days      []int  `json:"days" example:"1,2,3,4,5"` // 0=Sunday, 1=Monday...
	StartTime string `json:"startTime" example:"09:00"`
	EndTime   string `json:"endTime" example:"17:00"`

	// Speed Limits during this window
	DownloadLimit string `json:"downloadLimit" example:"500K"` // 0 = unlimited, "paused" = pause all
	UploadLimit   string `json:"uploadLimit" example:"100K"`
}

type AdvancedSettings struct {
	LogLevel     string `json:"logLevel" enums:"debug,info,warn,error"`
	DebugMode    bool   `json:"debugMode"`
	SaveInterval int    `json:"saveInterval" example:"60"` // Seconds
}

type SettingsUpdatedEventData struct {
	Changes []string `json:"changes"`
}

func DefaultSettings() *Settings {
	home, _ := os.UserHomeDir()
	defaultDir := filepath.Join(home, ".gravity", "downloads")

	return &Settings{
		Download: DownloadSettings{
			DownloadDir:            defaultDir,
			MaxConcurrentDownloads: 3,
			PreferredEngine:        "aria2",
			PreferredMagnetEngine:  "aria2",
			MaxConnectionPerServer: 8,
			Split:                  8,
			AutoResume:             true,
			PreAllocateSpace:       true,
			MinSplitSize:           "1M",
			ConnectTimeout:         60,
		},
		Upload: UploadSettings{
			ConcurrentUploads: 1,
			MaxRetryAttempts:  3,
			ChunkSize:         "64M",
		},
		Network: NetworkSettings{},
		Torrent: TorrentSettings{
			SeedRatio:  "1.0",
			SeedTime:   1440,
			ListenPort: 6881,
			EnableDht:  true,
			EnablePex:  true,
			EnableLpd:  true,
			Encryption: "enabled",
		},
		Automation: AutomationSettings{
			ScheduleEnabled: false,
			Categories: []Category{
				{ID: "cat_comp", Name: "Compressed", Icon: "archive", Extensions: []string{"zip", "rar", "7z", "tar", "gz"}, IsDefault: true, Path: "Compressed"},
				{ID: "cat_doc", Name: "Documents", Icon: "file-text", Extensions: []string{"pdf", "doc", "docx", "xls", "xlsx", "txt"}, IsDefault: true, Path: "Documents"},
				{ID: "cat_music", Name: "Music", Icon: "music", Extensions: []string{"mp3", "wav", "flac", "ogg", "m4a"}, IsDefault: true, Path: "Music"},
				{ID: "cat_prog", Name: "Programs", Icon: "terminal", Extensions: []string{"exe", "msi", "dmg", "app", "deb", "rpm"}, IsDefault: true, Path: "Programs"},
				{ID: "cat_video", Name: "Video", Icon: "video", Extensions: []string{"mp4", "mkv", "avi", "mov", "wmv"}, IsDefault: true, Path: "Video"},
			},
		},
		Advanced: AdvancedSettings{
			LogLevel:     "info",
			SaveInterval: 60,
		},
	}
}