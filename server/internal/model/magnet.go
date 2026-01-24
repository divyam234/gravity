package model

// MagnetInfo represents information about a magnet link
type MagnetInfo struct {
	Source   string        `json:"source"`             // "alldebrid" or "aria2"
	Cached   bool          `json:"cached"`             // true if cached on debrid service
	MagnetID string        `json:"magnetId,omitempty"` // debrid service magnet ID
	Name     string        `json:"name"`
	Hash     string        `json:"hash"`
	Size     int64         `json:"size"`
	Files    []*MagnetFile `json:"files"`
}

// MagnetFile represents a file within a magnet/torrent
type MagnetFile struct {
	ID       string       `json:"id"`   // AllDebrid file ID or aria2c 1-indexed file number
	Name     string       `json:"name"` // filename only
	Path     string       `json:"path"` // full path within torrent
	Size     int64        `json:"size"`
	Link     string       `json:"link,omitempty"` // direct download link (AllDebrid only)
	IsFolder bool         `json:"isFolder"`
	Children []*MagnetFile `json:"children,omitempty"`
	Index    int          `json:"index,omitempty"` // 1-indexed file number for aria2c --select-file
}
