package model

type Settings struct {
	Download DownloadSettings `json:"download"`
	Upload   UploadSettings   `json:"upload"`
	Network  NetworkSettings  `json:"network"`
	Torrent  TorrentSettings  `json:"torrent"`
	Vfs      VfsSettings      `json:"vfs"`
}

type DownloadSettings struct {
	DownloadDir            string `json:"downloadDir"`
	MaxConcurrentDownloads int    `json:"maxConcurrentDownloads"`
	MaxDownloadSpeed       string `json:"maxDownloadSpeed"`
	MaxUploadSpeed         string `json:"maxUploadSpeed"`
	MaxConnectionPerServer int    `json:"maxConnectionPerServer"`
	Split                  int    `json:"split"`
	UserAgent              string `json:"userAgent"`
	ConnectTimeout         int    `json:"connectTimeout"`
	MaxTries               int    `json:"maxTries"`
	CheckCertificate       bool   `json:"checkCertificate"`
}

type UploadSettings struct {
	DefaultRemote string `json:"defaultRemote"`
	AutoUpload    bool   `json:"autoUpload"`
	RemoveLocal   bool   `json:"removeLocal"`
}

type NetworkSettings struct {
	ProxyEnabled  bool   `json:"proxyEnabled"`
	ProxyUrl      string `json:"proxyUrl"`
	ProxyUser     string `json:"proxyUser"`
	ProxyPassword string `json:"proxyPassword"`
}

type TorrentSettings struct {
	SeedRatio     string `json:"seedRatio"`
	SeedTime      int    `json:"seedTime"`
	ListenPort    int    `json:"listenPort"`
	ForceSave     bool   `json:"forceSave"`
	EnablePex     bool   `json:"enablePex"`
	EnableDht     bool   `json:"enableDht"`
	EnableLpd     bool   `json:"enableLpd"`
	Encryption    string `json:"encryption"`
}

type VfsSettings struct {
	CacheMode         string `json:"cacheMode"`
	CacheMaxSize      string `json:"cacheMaxSize"`
	CacheMaxAge       string `json:"cacheMaxAge"`
	WriteBack         string `json:"writeBack"`
	ReadChunkSize     string `json:"readChunkSize"`
	ReadChunkSizeLimit string `json:"readChunkSizeLimit"`
	ReadAhead         string `json:"readAhead"`
	DirCacheTime      string `json:"dirCacheTime"`
	PollInterval      string `json:"pollInterval"`
	ReadChunkStreams  int    `json:"readChunkStreams"`
}
