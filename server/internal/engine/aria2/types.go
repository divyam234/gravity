package aria2

type VersionResponse struct {
	Version string `json:"version"`
}

type Aria2Peer struct {
	IP            string `json:"ip"`
	Port          string `json:"port"`
	DownloadSpeed string `json:"downloadSpeed"`
	UploadSpeed   string `json:"uploadSpeed"`
	Seeder        string `json:"seeder"`
}
