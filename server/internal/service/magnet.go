package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/provider/alldebrid"
	"gravity/internal/store"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type MagnetService struct {
	downloadRepo   *store.DownloadRepo
	settingsRepo   *store.SettingsRepo
	downloadEngine engine.DownloadEngine
	allDebrid      *alldebrid.AllDebridProvider
	uploadEngine   engine.UploadEngine
	bus            *event.Bus
	ctx            context.Context
	logger         *zap.Logger
}

func NewMagnetService(
	repo *store.DownloadRepo,
	settingsRepo *store.SettingsRepo,
	de engine.DownloadEngine,
	allDebrid *alldebrid.AllDebridProvider,
	uploadEngine engine.UploadEngine,
	bus *event.Bus,
	l *zap.Logger,
) *MagnetService {
	return &MagnetService{
		downloadRepo:   repo,
		settingsRepo:   settingsRepo,
		downloadEngine: de,
		allDebrid:      allDebrid,
		uploadEngine:   uploadEngine,
		bus:            bus,
		logger:         l.With(zap.String("service", "magnet")),
	}
}

func (s *MagnetService) Start(ctx context.Context) {
	s.ctx = ctx
}

// CheckMagnet checks if a magnet is available and returns file list
func (s *MagnetService) CheckMagnet(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
	s.logger.Debug("checking magnet", zap.String("magnet", magnet))

	// Set a reasonable timeout for the whole check process
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// 1. Try AllDebrid first (if configured)
	if s.allDebrid != nil && s.allDebrid.IsConfigured() {
		s.logger.Debug("trying AllDebrid cache check")
		info, err := s.allDebrid.CheckMagnet(ctx, magnet)
		if err == nil && info != nil && info.Cached {
			s.logger.Debug("found cached magnet on AllDebrid", zap.String("name", info.Name))
			return info, nil
		}
		s.logger.Debug("AllDebrid check failed or not cached", zap.Error(err))
	}

	// 2. Fall back to raw magnet via aria2
	s.logger.Debug("falling back to aria2 metadata fetch")
	info, err := s.downloadEngine.GetMagnetFiles(ctx, magnet)
	if err != nil {
		s.logger.Error("aria2 metadata fetch failed", zap.Error(err))
		return nil, fmt.Errorf("failed to get magnet files: %w", err)
	}

	s.logger.Debug("successfully fetched metadata via aria2", zap.String("name", info.Name))
	return info, nil
}

// CheckTorrent checks if a .torrent file's content is available and returns file list
func (s *MagnetService) CheckTorrent(ctx context.Context, torrentBase64 string) (*model.MagnetInfo, error) {
	s.logger.Debug("checking torrent file")

	// 1. Get metadata from aria2 first to get the hash
	info, err := s.downloadEngine.GetTorrentFiles(ctx, torrentBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse torrent: %w", err)
	}

	// 2. Try AllDebrid cache check with the hash
	if s.allDebrid != nil && s.allDebrid.IsConfigured() && info.Hash != "" {
		magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s", info.Hash)
		cached, err := s.allDebrid.CheckMagnet(ctx, magnet)
		if err == nil && cached != nil && cached.Cached {
			s.logger.Debug("torrent found in AllDebrid cache", zap.String("name", cached.Name))
			return cached, nil
		}
	}

	return info, nil
}

// MagnetDownloadRequest contains parameters for starting a magnet download
type MagnetDownloadRequest struct {
	Magnet        string              `json:"magnet"`
	TorrentBase64 string              `json:"torrentBase64"`
	Source        string              `json:"source"`
	MagnetID      string              `json:"magnetId"`
	Name          string              `json:"name"`
	SelectedFiles []string            `json:"selectedFiles"`
	AllFiles      []*model.MagnetFile `json:"allFiles"`
	
	// Flattened options
	DownloadDir string            `json:"downloadDir"`
	Destination string            `json:"destination"`
	Split       *int              `json:"split"`
	MaxTries    *int              `json:"maxTries"`
	UserAgent   *string           `json:"userAgent"`
	ProxyURL    *string           `json:"proxyUrl"`
	RemoveLocal *bool             `json:"removeLocal"`
	Headers     map[string]string `json:"headers"`
}

// DownloadMagnet starts download of selected files from a magnet
func (s *MagnetService) DownloadMagnet(ctx context.Context, req MagnetDownloadRequest) (*model.Download, error) {
	// Get download directory from settings
	settings, _ := s.settingsRepo.Get(ctx)
	defaultDir := ""
	if settings != nil {
		defaultDir = settings.Download.DownloadDir
	}
	if defaultDir == "" {
		home, _ := os.UserHomeDir()
		defaultDir = filepath.Join(home, ".gravity", "downloads")
	}

	var localPath string
	downloadDir := req.DownloadDir

	// If downloadDir is provided, use it (absolute or relative to default)
	if downloadDir == "" {
		localPath = defaultDir
	} else if filepath.IsAbs(downloadDir) {
		localPath = downloadDir
	} else {
		localPath = filepath.Join(defaultDir, downloadDir)
	}

	// Create download record
	d := &model.Download{
		ID:           "d_" + uuid.New().String()[:8],
		URL:          req.Magnet,
		Status:       model.StatusWaiting,
		Destination:  req.Destination,
		DownloadDir:  localPath,
		IsMagnet:     true,
		MagnetSource: req.Source,
		MagnetID:     req.MagnetID,
		MagnetHash:   extractHashFromMagnet(req.Magnet),
		TorrentData:  req.TorrentBase64,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		// Flattened options
		Split:       req.Split,
		MaxTries:    req.MaxTries,
		UserAgent:   req.UserAgent,
		ProxyURL:    req.ProxyURL,
		RemoveLocal: req.RemoveLocal,
		Headers:     req.Headers,
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

	s.bus.PublishLifecycle(event.LifecycleEvent{
		Type:      event.DownloadCreated,
		ID:        d.ID,
		Timestamp: time.Now(),
		Data:      d,
	})

	// Start downloads based on source
	if req.Source == "alldebrid" {
		d.Status = model.StatusActive
		s.downloadRepo.Update(ctx, d)

		// Use service lifecycle context for background task
		bgCtx := s.ctx
		if bgCtx == nil {
			bgCtx = context.Background()
		}
		go s.startAllDebridDownload(bgCtx, d)
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

		// Unlock the link first
		resolved, err := s.allDebrid.Resolve(ctx, file.URL, nil)
		if err != nil {
			file.Status = model.StatusError
			file.Error = "Link unlock failed: " + err.Error()
			s.downloadRepo.UpdateFile(ctx, file)
			continue
		}

		// Add to aria2
		opts := toEngineOptionsFromDownload(d)
		opts.DownloadDir = d.DownloadDir // Base directory
		opts.Filename = file.Path        // Preserve path structure
		gid, err := s.downloadEngine.Add(ctx, resolved.URL, opts)

		if err != nil {
			file.Status = model.StatusError
			file.Error = err.Error()
			s.downloadRepo.UpdateFile(ctx, file)
			continue
		}

		file.EngineID = gid
		file.Status = model.StatusActive
		s.downloadRepo.UpdateFile(ctx, file)
	}
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
	opts := toEngineOptionsFromDownload(d)
	opts.DownloadDir = d.DownloadDir
	gid, err := s.downloadEngine.AddMagnetWithSelection(ctx, magnet, indexes, opts)

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
	var indexes []int
	for _, file := range d.Files {
		if file.Index > 0 {
			indexes = append(indexes, file.Index)
		}
	}

	opts := toEngineOptionsFromDownload(d)
	opts.DownloadDir = d.DownloadDir
	opts.TorrentData = torrentBase64
	opts.SelectedFiles = indexes
	gid, err := s.downloadEngine.Add(ctx, "", opts)

	if err != nil {
		d.Status = model.StatusError
		d.Error = err.Error()
		s.downloadRepo.Update(ctx, d)
		return
	}

	d.EngineID = gid
	s.downloadRepo.Update(ctx, d)
}

func (r *MagnetDownloadRequest) FindFile(id string) *model.MagnetFile {
	return findFileRecursive(r.AllFiles, id)
}

func findFileRecursive(files []*model.MagnetFile, id string) *model.MagnetFile {
	for i := range files {
		if files[i].ID == id {
			return files[i]
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
func flattenFiles(files []*model.MagnetFile) []*model.MagnetFile {
	var result []*model.MagnetFile

	var traverse func(items []*model.MagnetFile)
	traverse = func(items []*model.MagnetFile) {
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
