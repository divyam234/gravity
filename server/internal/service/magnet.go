package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gravity/internal/engine"
	"gravity/internal/engine/aria2"
	"gravity/internal/model"
	"gravity/internal/provider"
	"gravity/internal/store"

	"github.com/google/uuid"
)

type MagnetService struct {
	downloadRepo *store.DownloadRepo
	aria2Engine  *aria2.Engine
	registry     *provider.Registry
}

func NewMagnetService(
	repo *store.DownloadRepo,
	aria2Engine *aria2.Engine,
	registry *provider.Registry,
) *MagnetService {
	return &MagnetService{
		downloadRepo: repo,
		aria2Engine:  aria2Engine,
		registry:     registry,
	}
}

// CheckMagnet checks if a magnet is available and returns file list
func (s *MagnetService) CheckMagnet(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
	// Validate magnet URI
	if !strings.HasPrefix(strings.ToLower(magnet), "magnet:") {
		return nil, fmt.Errorf("invalid magnet URI")
	}

	// 1. Try debrid providers first (AllDebrid)
	for _, p := range s.registry.List() {
		magnetProvider, ok := p.(provider.MagnetProvider)
		if !ok {
			continue
		}

		if !p.IsConfigured() {
			continue
		}

		info, err := magnetProvider.CheckMagnet(ctx, magnet)
		if err != nil {
			// Log error but continue to next provider or fallback
			continue
		}

		if info != nil && info.Cached {
			return info, nil
		}
	}

	// 2. Fall back to raw magnet via aria2
	info, err := s.aria2Engine.GetMagnetFiles(ctx, magnet)
	if err != nil {
		return nil, fmt.Errorf("failed to get magnet files: %w", err)
	}

	return info, nil
}

// MagnetDownloadRequest contains parameters for starting a magnet download
type MagnetDownloadRequest struct {
	Magnet        string             `json:"magnet"`
	Source        string             `json:"source"`   // "alldebrid" or "aria2"
	MagnetID      string             `json:"magnetId"` // AllDebrid magnet ID
	Name          string             `json:"name"`
	SelectedFiles []string           `json:"selectedFiles"`
	Destination   string             `json:"destination"`
	Files         []model.MagnetFile `json:"files"` // Full file list for lookup
}

// DownloadMagnet starts download of selected files from a magnet
func (s *MagnetService) DownloadMagnet(ctx context.Context, req MagnetDownloadRequest) (*model.Download, error) {
	if len(req.SelectedFiles) == 0 {
		return nil, fmt.Errorf("no files selected")
	}

	// Create download record
	d := &model.Download{
		ID:           "d_" + uuid.New().String()[:8],
		URL:          req.Magnet,
		Filename:     req.Name,
		Status:       model.StatusActive,
		Destination:  req.Destination,
		IsMagnet:     true,
		MagnetSource: req.Source,
		MagnetID:     req.MagnetID,
		MagnetHash:   extractHashFromMagnet(req.Magnet),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Build file list from selected files
	selectedMap := make(map[string]bool)
	for _, id := range req.SelectedFiles {
		selectedMap[id] = true
	}

	var totalSize int64
	for _, file := range flattenFiles(req.Files) {
		if !selectedMap[file.ID] {
			continue
		}

		d.Files = append(d.Files, model.DownloadFile{
			ID:     file.ID,
			Name:   file.Name,
			Path:   file.Path,
			Size:   file.Size,
			Status: model.StatusWaiting,
			URL:    file.Link,
			Index:  file.Index,
		})
		totalSize += file.Size
	}

	d.Size = totalSize
	d.TotalFiles = len(d.Files)

	// Save to database
	if err := s.downloadRepo.CreateWithFiles(ctx, d); err != nil {
		return nil, err
	}

	// Start downloads based on source
	if req.Source == "alldebrid" {
		go s.startAllDebridDownload(context.Background(), d)
	} else {
		go s.startAria2Download(context.Background(), d, req.Magnet)
	}

	return d, nil
}

// startAllDebridDownload downloads files via AllDebrid direct links
func (s *MagnetService) startAllDebridDownload(ctx context.Context, d *model.Download) {
	// Download each file in parallel via aria2
	for i := range d.Files {
		file := &d.Files[i]
		if file.URL == "" {
			file.Status = model.StatusError
			file.Error = "no download URL"
			continue
		}

		// Add to aria2
		gid, err := s.aria2Engine.Add(ctx, file.URL, engine.DownloadOptions{
			Dir:      d.LocalPath,
			Filename: file.Path,
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
	s.downloadRepo.UpdateFiles(ctx, d.ID, d.Files)
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
