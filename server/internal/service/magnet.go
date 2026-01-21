package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gravity/internal/engine"
	"gravity/internal/engine/aria2"
	"gravity/internal/model"
	"gravity/internal/provider/alldebrid"
	"gravity/internal/store"

	"github.com/google/uuid"
)

type MagnetService struct {
	downloadRepo *store.DownloadRepo
	settingsRepo *store.SettingsRepo
	aria2Engine  *aria2.Engine
	allDebrid    *alldebrid.AllDebridProvider
	uploadEngine engine.UploadEngine
}

func NewMagnetService(
	repo *store.DownloadRepo,
	settingsRepo *store.SettingsRepo,
	aria2 *aria2.Engine,
	allDebrid *alldebrid.AllDebridProvider,
	uploadEngine engine.UploadEngine,
) *MagnetService {
	return &MagnetService{
		downloadRepo: repo,
		settingsRepo: settingsRepo,
		aria2Engine:  aria2,
		allDebrid:    allDebrid,
		uploadEngine: uploadEngine,
	}
}

// CheckMagnet checks if a magnet is available and returns file list
func (s *MagnetService) CheckMagnet(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
	log.Printf("[MagnetService] Checking magnet: %s", magnet)

	// Set a reasonable timeout for the whole check process
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// 1. Try AllDebrid first (if configured)
	if s.allDebrid != nil && s.allDebrid.IsConfigured() {
		log.Printf("[MagnetService] Trying AllDebrid...")
		info, err := s.allDebrid.CheckMagnet(ctx, magnet)
		if err == nil && info != nil && info.Cached {
			log.Printf("[MagnetService] Found cached magnet on AllDebrid: %s", info.Name)
			return info, nil
		}
		log.Printf("[MagnetService] AllDebrid check failed or not cached: %v", err)
	}

	// 2. Fall back to raw magnet via aria2
	log.Printf("[MagnetService] Falling back to aria2 metadata fetch...")
	info, err := s.aria2Engine.GetMagnetFiles(ctx, magnet)
	if err != nil {
		log.Printf("[MagnetService] aria2 fetch failed: %v", err)
		return nil, fmt.Errorf("failed to get magnet files: %w", err)
	}

	log.Printf("[MagnetService] Successfully fetched files via aria2: %s", info.Name)
	return info, nil
}

// CheckTorrent checks if a .torrent file's content is available and returns file list
func (s *MagnetService) CheckTorrent(ctx context.Context, torrentBase64 string) (*model.MagnetInfo, error) {
	log.Printf("[MagnetService] Checking torrent file")

	// 1. Get metadata from aria2 first to get the hash
	info, err := s.aria2Engine.GetTorrentFiles(ctx, torrentBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse torrent: %w", err)
	}

	// 2. Try AllDebrid cache check with the hash
	if s.allDebrid != nil && s.allDebrid.IsConfigured() && info.Hash != "" {
		magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s", info.Hash)
		cached, err := s.allDebrid.CheckMagnet(ctx, magnet)
		if err == nil && cached != nil && cached.Cached {
			log.Printf("[MagnetService] Torrent found in AllDebrid cache: %s", info.Name)
			return cached, nil
		}
	}

	return info, nil
}

// MagnetDownloadRequest contains parameters for starting a magnet download
type MagnetDownloadRequest struct {
	Magnet        string             `json:"magnet"`
	TorrentBase64 string             `json:"torrentBase64"`
	Source        string             `json:"source"`
	MagnetID      string             `json:"magnetId"`
	Name          string             `json:"name"`
	SelectedFiles []string           `json:"selectedFiles"`
	Destination   string             `json:"destination"`
	AllFiles      []model.MagnetFile `json:"allFiles"`
}

// DownloadMagnet starts download of selected files from a magnet
func (s *MagnetService) DownloadMagnet(ctx context.Context, req MagnetDownloadRequest) (*model.Download, error) {
	// Get download directory from settings
	settings, _ := s.settingsRepo.Get(ctx)
	defaultDir := settings["download_dir"]
	if defaultDir == "" {
		home, _ := os.UserHomeDir()
		defaultDir = filepath.Join(home, ".gravity", "downloads")
	}

	var localPath, uploadDest string

	// Smart Destination Logic (Same as DownloadService)
	isRemote := strings.Contains(req.Destination, ":") && !filepath.IsAbs(req.Destination)

	if isRemote {
		localPath = defaultDir
		uploadDest = req.Destination
	} else {
		if req.Destination == "" {
			localPath = defaultDir
		} else if filepath.IsAbs(req.Destination) {
			localPath = req.Destination
		} else {
			localPath = filepath.Join(defaultDir, req.Destination)
		}
		uploadDest = ""
	}

	// Create download record
	d := &model.Download{
		ID:           "d_" + uuid.New().String()[:8],
		URL:          req.Magnet,
		Status:       model.StatusWaiting,
		Destination:  uploadDest,
		LocalPath:    localPath,
		IsMagnet:     true,
		MagnetSource: req.Source,
		MagnetID:     req.MagnetID,
		MagnetHash:   extractHashFromMagnet(req.Magnet),
		TorrentData:  req.TorrentBase64,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Build file list and calculate total size
	var totalSize int64
	var selectedIndexes []int
	for _, fileID := range req.SelectedFiles {
		file := req.FindFile(fileID)
		if file == nil {
			continue
		}

		d.Files = append(d.Files, model.DownloadFile{
			ID:     "df_" + uuid.New().String()[:8],
			Name:   file.Name,
			Path:   file.Path,
			Size:   file.Size,
			Status: model.StatusWaiting,
			URL:    file.Link,  // Only for AllDebrid
			Index:  file.Index, // Only for aria2
		})
		totalSize += file.Size
		if file.Index > 0 {
			selectedIndexes = append(selectedIndexes, file.Index)
		}
	}

	d.Size = totalSize
	d.TotalFiles = len(d.Files)
	d.Filename = req.Name // Torrent name
	d.SelectedFiles = selectedIndexes

	// Save to database
	if err := s.downloadRepo.CreateWithFiles(ctx, d); err != nil {
		return nil, err
	}

	// Start downloads based on source
	if req.Source == "alldebrid" {
		go s.startAllDebridDownload(context.Background(), d)
	} else {
		// Native torrents/magnets go to the queue first
		d.Status = model.StatusWaiting
		s.downloadRepo.Update(ctx, d)
	}

	return d, nil
}

// startAllDebridDownload downloads files via AllDebrid direct links
func (s *MagnetService) startAllDebridDownload(ctx context.Context, d *model.Download) {
	// Download each file in parallel via aria2
	for i := range d.Files {
		file := &d.Files[i]
		if file.URL == "" {
			continue
		}

		// Add to aria2
		gid, err := s.aria2Engine.Add(ctx, file.URL, engine.DownloadOptions{
			Dir:      d.LocalPath, // Base directory
			Filename: file.Path,   // Preserve path structure
		})

		if err != nil {
			file.Status = model.StatusError
			file.Error = err.Error()
			continue
		}

		file.EngineID = gid
		file.Status = model.StatusActive
	}

	// Update database
	s.downloadRepo.UpdateFiles(context.Background(), d.ID, d.Files)
}

// startAria2Download downloads magnet via native aria2 BitTorrent
func (s *MagnetService) startAria2Download(ctx context.Context, d *model.Download, magnet string) {
	// Collect selected file indexes
	var indexes []string
	for _, file := range d.Files {
		if file.Index > 0 {
			indexes = append(indexes, fmt.Sprintf("%d", file.Index))
		}
	}

	// Add magnet with file selection
	gid, err := s.aria2Engine.AddMagnetWithSelection(ctx, magnet, indexes, engine.DownloadOptions{
		Dir: d.LocalPath,
	})

	if err != nil {
		d.Status = model.StatusError
		d.Error = err.Error()
		s.downloadRepo.Update(ctx, d)
		return
	}

	d.EngineID = gid
	s.downloadRepo.Update(ctx, d)
}

// startTorrentDownload downloads torrent via native aria2 BitTorrent from file
func (s *MagnetService) startTorrentDownload(ctx context.Context, d *model.Download, torrentBase64 string) {
	// Collect selected file indexes
	var indexes []string
	for _, file := range d.Files {
		if file.Index > 0 {
			indexes = append(indexes, fmt.Sprintf("%d", file.Index))
		}
	}

	// Build select-file string
	options := map[string]interface{}{
		"paused": "false",
	}
	if len(indexes) > 0 {
		options["select-file"] = strings.Join(indexes, ",")
	}
	if d.LocalPath != "" {
		options["dir"] = d.LocalPath
	}

	// aria2.addTorrent expects: [base64_torrent, urls, options]
	res, err := s.aria2Engine.GetClient().Call(ctx, "aria2.addTorrent", torrentBase64, []interface{}{}, options)
	if err != nil {
		d.Status = model.StatusError
		d.Error = err.Error()
		s.downloadRepo.Update(ctx, d)
		return
	}

	var gid string
	json.Unmarshal(res, &gid)

	d.EngineID = gid
	s.downloadRepo.Update(ctx, d)
}

func (r *MagnetDownloadRequest) FindFile(id string) *model.MagnetFile {
	return findFileRecursive(r.AllFiles, id)
}

func findFileRecursive(files []model.MagnetFile, id string) *model.MagnetFile {
	for i := range files {
		if files[i].ID == id {
			return &files[i]
		}
		if len(files[i].Children) > 0 {
			if found := findFileRecursive(files[i].Children, id); found != nil {
				return found
			}
		}
	}
	return nil
}

// flattenFiles converts nested MagnetFile tree to flat array
func flattenFiles(files []model.MagnetFile) []model.MagnetFile {
	var result []model.MagnetFile

	var traverse func(items []model.MagnetFile)
	traverse = func(items []model.MagnetFile) {
		for _, file := range items {
			if !file.IsFolder {
				result = append(result, file)
			}
			if len(file.Children) > 0 {
				traverse(file.Children)
			}
		}
	}

	traverse(files)
	return result
}

// extractHashFromMagnet extracts btih hash from magnet URI
func extractHashFromMagnet(magnet string) string {
	lower := strings.ToLower(magnet)
	if idx := strings.Index(lower, "btih:"); idx >= 0 {
		hash := magnet[idx+5:]
		if ampIdx := strings.Index(hash, "&"); ampIdx >= 0 {
			hash = hash[:ampIdx]
		}
		return strings.ToLower(hash)
	}
	return ""
}
