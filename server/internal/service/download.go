package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/store"

	"github.com/google/uuid"
)

type DownloadService struct {
	repo         *store.DownloadRepo
	settingsRepo *store.SettingsRepo
	engine       engine.DownloadEngine
	uploadEngine engine.UploadEngine
	bus          *event.Bus
	provider     *ProviderService

	// Throttling for database writes
	lastPersistMap map[string]time.Time
	mu             sync.RWMutex
}

func NewDownloadService(repo *store.DownloadRepo, settingsRepo *store.SettingsRepo, eng engine.DownloadEngine, ue engine.UploadEngine, bus *event.Bus, provider *ProviderService) *DownloadService {
	s := &DownloadService{
		repo:           repo,
		settingsRepo:   settingsRepo,
		engine:         eng,
		uploadEngine:   ue,
		bus:            bus,
		provider:       provider,
		lastPersistMap: make(map[string]time.Time),
	}

	// Wire up engine events
	eng.OnProgress(s.handleProgress)
	eng.OnComplete(s.handleComplete)
	eng.OnError(s.handleError)

	return s
}

func (s *DownloadService) Create(ctx context.Context, url string, filename string, destination string) (*model.Download, error) {
	// 1. Resolve URL through providers
	res, providerName, err := s.provider.Resolve(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve URL: %w", err)
	}

	if filename == "" {
		filename = res.Filename
	}

	d := &model.Download{
		ID:          "d_" + uuid.New().String()[:8],
		URL:         url,
		ResolvedURL: res.URL,
		Headers:     res.Headers,
		Provider:    providerName,
		Filename:    filename,
		Size:        res.Size,
		Destination: destination,
		Status:      model.StatusWaiting,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}

	s.bus.Publish(event.Event{
		Type:      event.DownloadCreated,
		Timestamp: time.Now(),
		Data:      d,
	})

	// Trigger queue processing
	go s.ProcessQueue(context.Background())

	return d, nil
}

func (s *DownloadService) Get(ctx context.Context, id string) (*model.Download, error) {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Attach files for multi-file downloads
	files, err := s.repo.GetFiles(ctx, id)
	if err == nil && len(files) > 0 {
		d.Files = files
	}

	// Merge live stats from engine if active
	if d.Status == model.StatusActive && d.EngineID != "" {
		status, err := s.engine.Status(ctx, d.EngineID)
		if err == nil {
			d.Downloaded = status.Downloaded
			d.Size = status.Size
			d.Speed = status.Speed
			d.ETA = status.Eta
			d.Seeders = status.Seeders
			d.Peers = status.Peers

			// Also fetch detailed peers for active aria2 downloads
			if d.MagnetSource == "aria2" || !d.IsMagnet {
				peers, err := s.engine.GetPeers(ctx, d.EngineID)
				if err == nil {
					d.PeerDetails = make([]model.Peer, 0, len(peers))
					for _, p := range peers {
						d.PeerDetails = append(d.PeerDetails, model.Peer{
							IP:            p.IP,
							Port:          p.Port,
							DownloadSpeed: p.DownloadSpeed,
							UploadSpeed:   p.UploadSpeed,
							IsSeeder:      p.IsSeeder,
						})
					}
				}
			}
		}
	}

	return d, nil
}

func (s *DownloadService) List(ctx context.Context, status []string, limit, offset int) ([]*model.Download, int, error) {
	downloads, total, err := s.repo.List(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Fetch all engine tasks to merge stats
	engineTasks, err := s.engine.List(ctx)
	if err == nil {
		// Map by ID for quick lookup
		liveMap := make(map[string]*engine.DownloadStatus)
		for _, t := range engineTasks {
			liveMap[t.ID] = t
		}

		// Merge live data into results
		for _, d := range downloads {
			if d.EngineID != "" {
				if live, ok := liveMap[d.EngineID]; ok {
					d.Downloaded = live.Downloaded
					d.Size = live.Size
					d.Speed = live.Speed
					d.ETA = live.Eta
					d.Seeders = live.Seeders
					d.Peers = live.Peers
				}
			}
		}
	}

	return downloads, total, nil
}

func (s *DownloadService) Pause(ctx context.Context, id string) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	// Remove from engine if active
	if d.EngineID != "" {
		s.engine.Cancel(ctx, d.EngineID)
	}

	d.Status = model.StatusPaused
	d.EngineID = "" // Clear GID as it's no longer in engine
	if err := s.repo.Update(ctx, d); err != nil {
		return err
	}

	s.bus.Publish(event.Event{
		Type:      event.DownloadPaused,
		Timestamp: time.Now(),
		Data:      map[string]string{"id": id},
	})

	// Trigger queue to fill the slot
	go s.ProcessQueue(context.Background())

	return nil
}

func (s *DownloadService) Resume(ctx context.Context, id string) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	d.Status = model.StatusWaiting
	if err := s.repo.Update(ctx, d); err != nil {
		return err
	}

	s.bus.Publish(event.Event{
		Type:      event.DownloadResumed,
		Timestamp: time.Now(),
		Data:      map[string]string{"id": id},
	})

	// Trigger queue
	go s.ProcessQueue(context.Background())

	return nil
}

func (s *DownloadService) Retry(ctx context.Context, id string) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	// Only retry failed or completed tasks
	if d.Status != model.StatusError && d.Status != model.StatusComplete {
		return fmt.Errorf("cannot retry download in status %s", d.Status)
	}

	// Re-add to engine
	engineID, err := s.engine.Add(ctx, d.ResolvedURL, engine.DownloadOptions{
		Filename: d.Filename,
		Dir:      d.Destination,
	})
	if err != nil {
		return err
	}

	d.EngineID = engineID
	d.Status = model.StatusActive
	d.Error = ""
	d.Downloaded = 0
	d.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, d); err != nil {
		return err
	}

	s.bus.Publish(event.Event{
		Type:      event.DownloadStarted,
		Timestamp: time.Now(),
		Data:      d,
	})

	return nil
}

func (s *DownloadService) Delete(ctx context.Context, id string, deleteFiles bool) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	// 1. Stop in engine
	if d.IsMagnet && d.MagnetSource == "alldebrid" {
		files, _ := s.repo.GetFiles(ctx, d.ID)
		for _, f := range files {
			if f.EngineID != "" {
				s.engine.Cancel(ctx, f.EngineID)
				if deleteFiles {
					s.engine.Remove(ctx, f.EngineID)
				}
			}
		}
	} else if d.EngineID != "" {
		s.engine.Cancel(ctx, d.EngineID)
		if deleteFiles {
			s.engine.Remove(ctx, d.EngineID)
		}
	}

	// 2. Also stop upload if active
	if d.Status == model.StatusUploading && d.UploadJobID != "" {
		s.uploadEngine.Cancel(ctx, d.UploadJobID)
	}

	// 3. Clean up database
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *DownloadService) Start() {
	// 1. Listen for engine events to trigger scheduler
	events := s.bus.Subscribe()
	go func() {
		for ev := range events {
			switch ev.Type {
			case event.DownloadCompleted, event.DownloadError:
				s.ProcessQueue(context.Background())
			}
		}
	}()

	// 2. Periodic scheduler check
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.ProcessQueue(context.Background())
		}
	}()

	// 3. Initial processing
	s.ProcessQueue(context.Background())
}

func (s *DownloadService) ProcessQueue(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get max concurrent settings
	settings, _ := s.settingsRepo.Get(ctx)
	maxConcurrent := 3 // default
	if val, ok := settings["max_concurrent_downloads"]; ok {
		if i, err := strconv.Atoi(val); err == nil {
			maxConcurrent = i
		}
	}

	// 1. Count active tasks in DB (instead of engine to be more robust)
	dbActive, _, _ := s.repo.List(ctx, []string{string(model.StatusActive)}, 100, 0)
	activeCount := len(dbActive)

	if activeCount >= maxConcurrent {
		return
	}

	// 2. Fetch next waiting tasks from DB
	limit := maxConcurrent - activeCount
	waiting, _, err := s.repo.List(ctx, []string{string(model.StatusWaiting)}, limit, 0)
	if err != nil || len(waiting) == 0 {
		return
	}

	for _, d := range waiting {
		// Calculate total size from selected files only if multi-file
		var totalSize int64
		files, _ := s.repo.GetFiles(ctx, d.ID)
		if len(files) > 0 {
			for _, f := range files {
				totalSize += f.Size
			}
			d.Size = totalSize
		}

		// Submit to engine
		opts := engine.DownloadOptions{
			Filename:      d.Filename,
			Dir:           d.LocalPath,
			Headers:       d.Headers,
			TorrentData:   d.TorrentData,
			SelectedFiles: d.SelectedFiles,
		}

		// If it's a magnet, we need special handling if it's already got metadata
		// but for now, simple Add is enough for normal URLs and magnets.
		engineID, err := s.engine.Add(ctx, d.ResolvedURL, opts)
		if err != nil {
			d.Status = model.StatusError
			d.Error = "Queue submission failed: " + err.Error()
			s.repo.Update(ctx, d)
			continue
		}

		d.EngineID = engineID
		d.Status = model.StatusActive
		s.repo.Update(ctx, d)

		s.bus.Publish(event.Event{
			Type:      event.DownloadStarted,
			Timestamp: time.Now(),
			Data:      d,
		})
	}
}

func (s *DownloadService) Sync(ctx context.Context) error {
	engineTasks, err := s.engine.List(ctx)
	if err != nil {
		return err
	}

	gidMap := make(map[string]*engine.DownloadStatus)
	for _, t := range engineTasks {
		gidMap[t.ID] = t
	}

	dbActive, _, err := s.repo.List(ctx, []string{string(model.StatusActive)}, 1000, 0)
	if err != nil {
		return err
	}

	for _, d := range dbActive {
		if d.EngineID == "" {
			d.Status = model.StatusWaiting
			s.repo.Update(ctx, d)
			continue
		}

		if matched, ok := gidMap[d.EngineID]; ok {
			// update status if it changed in engine (e.g. finished while app was off)
			if matched.Status == "complete" {
				s.handleComplete(matched.ID, "")
			} else if matched.Status == "error" {
				s.handleError(matched.ID, fmt.Errorf("engine reported error"))
			} else if matched.Status == "paused" {
				d.Status = model.StatusPaused
				s.repo.Update(ctx, d)
			}
		} else {
			// disappeared from engine entirely, move to waiting so queue can pick it up
			d.Status = model.StatusWaiting
			d.EngineID = ""
			s.repo.Update(ctx, d)
		}
	}

	// Trigger queue processing
	go s.ProcessQueue(context.Background())

	return nil
}

func (s *DownloadService) handleProgress(engineID string, p engine.Progress) {
	ctx := context.Background()

	// 1. Check for per-file progress update (gid:index)
	if strings.Contains(engineID, ":") {
		parts := strings.Split(engineID, ":")
		if len(parts) == 2 {
			parentGID := parts[0]
			fileIndex, _ := strconv.Atoi(parts[1])

			parent, err := s.repo.GetByEngineID(ctx, parentGID)
			if err == nil {
				file, err := s.repo.GetFileByDownloadIDAndIndex(ctx, parent.ID, fileIndex)
				if err == nil {
					file.Downloaded = p.Downloaded
					file.Progress = 0
					if file.Size > 0 {
						file.Progress = int((file.Downloaded * 100) / file.Size)
					}

					oldStatus := file.Status
					if file.Progress == 100 {
						file.Status = model.StatusComplete
					} else {
						file.Status = model.StatusActive
					}

					s.mu.Lock()
					lastPersist := s.lastPersistMap[file.ID]
					if file.Status != oldStatus || time.Since(lastPersist) > 10*time.Second {
						s.repo.UpdateFile(ctx, file)
						s.lastPersistMap[file.ID] = time.Now()
					}
					s.mu.Unlock()

					// Update parent aggregate stats
					s.updateAggregateProgress(ctx, parent)
					return
				}
			}
		}
	}

	// 2. Try single-file download
	d, err := s.repo.GetByEngineID(ctx, engineID)
	if err == nil {
		d.Downloaded = p.Downloaded
		d.Size = p.Size
		d.Speed = p.Speed
		d.ETA = p.ETA
		d.Seeders = p.Seeders
		d.Peers = p.Peers

		s.mu.Lock()
		lastPersist := s.lastPersistMap[d.ID]
		if time.Since(lastPersist) > 10*time.Second {
			s.repo.Update(ctx, d)
			s.lastPersistMap[d.ID] = time.Now()
		}
		s.mu.Unlock()

		s.publishProgress(d)
		return
	}

	// 3. Try file within magnet (AllDebrid)
	file, err := s.repo.GetFileByEngineID(ctx, engineID)
	if err == nil {
		file.Downloaded = p.Downloaded
		file.Progress = 0
		if file.Size > 0 {
			file.Progress = int((file.Downloaded * 100) / file.Size)
		}

		oldStatus := file.Status
		file.Status = model.StatusActive

		s.mu.Lock()
		lastPersist := s.lastPersistMap[file.ID]
		if file.Status != oldStatus || time.Since(lastPersist) > 10*time.Second {
			s.repo.UpdateFile(ctx, file)
			s.lastPersistMap[file.ID] = time.Now()
		}
		s.mu.Unlock()

		parent, err := s.repo.Get(ctx, file.DownloadID)
		if err == nil {
			s.updateAggregateProgress(ctx, parent)
		}
		return
	}
}

func (s *DownloadService) updateAggregateProgress(ctx context.Context, d *model.Download) {
	files, err := s.repo.GetFiles(ctx, d.ID)
	if err != nil {
		return
	}

	var totalDownloaded int64
	var totalSize int64
	filesComplete := 0

	for _, f := range files {
		totalDownloaded += f.Downloaded
		totalSize += f.Size
		if f.Status == model.StatusComplete {
			filesComplete++
		}
	}

	d.Downloaded = totalDownloaded
	d.Size = totalSize
	d.FilesComplete = filesComplete

	if filesComplete == len(files) && len(files) > 0 && d.Status != model.StatusComplete && d.Status != model.StatusUploading {
		if d.Destination != "" {
			d.Status = model.StatusUploading
		} else {
			d.Status = model.StatusComplete
		}
		now := time.Now()
		d.CompletedAt = &now
	}

	s.mu.Lock()
	lastPersist := s.lastPersistMap[d.ID]
	if d.Status == model.StatusComplete || time.Since(lastPersist) > 10*time.Second {
		s.repo.Update(ctx, d)
		s.lastPersistMap[d.ID] = time.Now()
	}
	s.mu.Unlock()

	s.publishProgress(d)
}

func (s *DownloadService) publishProgress(d *model.Download) {
	s.bus.Publish(event.Event{
		Type:      event.DownloadProgress,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"id":         d.ID,
			"downloaded": d.Downloaded,
			"size":       d.Size,
			"speed":      d.Speed,
			"eta":        d.ETA,
			"seeders":    d.Seeders,
			"peers":      d.Peers,
		},
	})
}

func (s *DownloadService) handleComplete(engineID string, filePath string) {
	ctx := context.Background()

	d, err := s.repo.GetByEngineID(ctx, engineID)
	if err == nil {
		// Determine the actual path created by aria2
		status, err := s.engine.Status(ctx, engineID)
		if err == nil {
			// aria2 returns 'dir' (output dir) and 'files' (actual paths)
			// For a torrent, files[0].path will be something like /downloads/ReleaseName/file.mp4
			// if dir is /downloads, we want to upload /downloads/ReleaseName
			if len(status.Files) > 0 {
				actualPath := status.Files[0].Path // Use .Path from the refactored engine status
				if actualPath == "" {
					actualPath = filePath // fallback
				}

				// Calculate relative path to the download directory
				rel, err := filepath.Rel(status.Dir, actualPath)
				if err == nil {
					// Split relative path and take the first component
					// e.g. "Release/file.mp4" -> "Release"
					// e.g. "file.mp4" -> "file.mp4"
					parts := strings.Split(filepath.ToSlash(rel), "/")
					if len(parts) > 0 {
						d.LocalPath = filepath.Join(status.Dir, parts[0])
					}
				}
			}
		}

		if d.Destination != "" {
			d.Status = model.StatusUploading
		} else {
			d.Status = model.StatusComplete
		}

		now := time.Now()
		d.CompletedAt = &now
		s.repo.Update(ctx, d)
		s.repo.MarkAllFilesComplete(ctx, d.ID)

		s.bus.Publish(event.Event{
			Type:      event.DownloadCompleted,
			Timestamp: time.Now(),
			Data:      d,
		})
		return
	}

	file, err := s.repo.GetFileByEngineID(ctx, engineID)
	if err == nil {
		file.Status = model.StatusComplete
		file.Downloaded = file.Size
		file.Progress = 100
		s.repo.UpdateFile(ctx, file)

		parent, err := s.repo.Get(ctx, file.DownloadID)
		if err == nil {
			s.updateAggregateProgress(ctx, parent)
		}
		return
	}
}

func (s *DownloadService) GetFiles(ctx context.Context, id string) ([]model.DownloadFile, error) {
	return s.repo.GetFiles(ctx, id)
}

func (s *DownloadService) handleError(engineID string, err error) {
	ctx := context.Background()

	d, repoErr := s.repo.GetByEngineID(ctx, engineID)
	if repoErr == nil {
		d.Status = model.StatusError
		d.Error = err.Error()
		s.repo.Update(ctx, d)

		s.bus.Publish(event.Event{
			Type:      event.DownloadError,
			Timestamp: time.Now(),
			Data:      map[string]string{"id": d.ID, "error": d.Error},
		})
		return
	}

	file, repoErr := s.repo.GetFileByEngineID(ctx, engineID)
	if repoErr == nil {
		file.Status = model.StatusError
		file.Error = err.Error()
		s.repo.UpdateFile(ctx, file)

		parent, err := s.repo.Get(ctx, file.DownloadID)
		if err == nil {
			s.updateAggregateProgress(ctx, parent)
		}
		return
	}
}
