package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/store"
	"syscall"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type DownloadService struct {
	repo         *store.DownloadRepo
	settingsRepo *store.SettingsRepo
	engine       engine.DownloadEngine
	uploadEngine engine.UploadEngine
	bus          *event.Bus
	provider     *ProviderService
	logger       *zap.Logger

	// Lifecycle context
	ctx context.Context

	// Throttling for database writes
	lastPersistMap map[string]time.Time
	mu             sync.RWMutex
	stop           chan struct{}
}

func NewDownloadService(repo *store.DownloadRepo, settingsRepo *store.SettingsRepo, eng engine.DownloadEngine, ue engine.UploadEngine, bus *event.Bus, provider *ProviderService, l *zap.Logger) *DownloadService {
	s := &DownloadService{
		repo:           repo,
		settingsRepo:   settingsRepo,
		engine:         eng,
		uploadEngine:   ue,
		bus:            bus,
		provider:       provider,
		logger:         l.With(zap.String("service", "download")),
		lastPersistMap: make(map[string]time.Time),
		stop:           make(chan struct{}),
	}

	// Wire up engine events
	eng.OnProgress(s.handleProgress)
	eng.OnComplete(s.handleComplete)
	eng.OnError(s.handleError)

	return s
}

func (s *DownloadService) Create(ctx context.Context, url string, filename string, downloadDir string, destination string, options model.TaskOptions) (*model.Download, error) {
	// 1. Resolve URL through providers
	res, providerName, err := s.provider.Resolve(ctx, url, options.Headers)
	if err != nil {
		s.logger.Warn("failed to resolve URL", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("failed to resolve URL: %w", err)
	}

	if filename == "" {
		filename = res.Filename
	}

	// 2. Determine paths
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

	// If downloadDir is provided, use it (absolute or relative to default)
	if downloadDir == "" {
		localPath = defaultDir
	} else if filepath.IsAbs(downloadDir) {
		localPath = downloadDir
	} else {
		localPath = filepath.Join(defaultDir, downloadDir)
	}

	d := &model.Download{
		ID:          "d_" + uuid.New().String()[:8],
		URL:         url,
		ResolvedURL: res.URL,
		Headers:     res.Headers,
		Provider:    providerName,
		Filename:    filename,
		Size:        res.Size,
		Destination: destination, // Remote upload destination
		DownloadDir: localPath,   // Actual local download directory
		Options:     options,
		Status:      model.StatusWaiting,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Merge user provided headers
	if options.Headers != nil {
		if d.Headers == nil {
			d.Headers = make(map[string]string)
		}
		for k, v := range options.Headers {
			d.Headers[k] = v
		}
	}

	if err := s.repo.Create(ctx, d); err != nil {
		s.logger.Error("failed to save download to DB", zap.String("id", d.ID), zap.Error(err))
		return nil, err
	}

	s.logger.Info("download created", 
		zap.String("id", d.ID), 
		zap.String("filename", d.Filename), 
		zap.String("provider", d.Provider))

	s.bus.Publish(event.Event{
		Type:      event.DownloadCreated,
		Timestamp: time.Now(),
		Data:      d,
	})

	// Trigger queue processing
	go func() {
		if s.ctx != nil {
			s.ProcessQueue(s.ctx)
		} else {
			s.ProcessQueue(context.Background())
		}
	}()

	return d, nil
}

func (s *DownloadService) Get(ctx context.Context, id string) (*model.Download, error) {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Attach files for multi-file downloads
	files, _, err := s.repo.GetFiles(ctx, id, 10000, 0)
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
			d.MetadataFetching = status.MetadataFetching

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

func (s *DownloadService) SettingsRepo() *store.SettingsRepo {
	return s.settingsRepo
}

func (s *DownloadService) List(ctx context.Context, status []string, limit, offset int) ([]*model.Download, int, error) {
	downloads, total, err := s.repo.List(ctx, status, limit, offset, false)
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
					d.MetadataFetching = live.MetadataFetching
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
		s.engine.Remove(ctx, d.EngineID)
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
	go func() {
		if s.ctx != nil {
			s.ProcessQueue(s.ctx)
		} else {
			s.ProcessQueue(context.Background())
		}
	}()

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
	go func() {
		if s.ctx != nil {
			s.ProcessQueue(s.ctx)
		} else {
			s.ProcessQueue(context.Background())
		}
	}()

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
	// Bug Fix: Use DownloadDir as the directory, NOT Destination (which might be a remote string)
	engineID, err := s.engine.Add(ctx, d.ResolvedURL, engine.DownloadOptions{
		Filename: d.Filename,
		Dir:      d.DownloadDir,
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

	// 1. Resolve paths for deletion before stopping engine
	var pathsToDelete []string
	if deleteFiles {
		// If we have a stored DownloadDir, that's the primary target
		if d.DownloadDir != "" {
			pathsToDelete = append(pathsToDelete, d.DownloadDir)
		}

		// If active/paused in engine, query it for current path
		if d.EngineID != "" {
			status, err := s.engine.Status(ctx, d.EngineID)
			if err == nil && len(status.Files) > 0 {
				// Use the logic from handleComplete to find root path
				actualPath := status.Files[0].Path
				if actualPath == "" {
					actualPath = filepath.Join(status.Dir, status.Filename)
				}

				// Add to deletion list if not already there (simple dedup)
				duplicate := false
				for _, p := range pathsToDelete {
					if p == actualPath {
						duplicate = true
						break
					}
				}
				if !duplicate && actualPath != "" {
					pathsToDelete = append(pathsToDelete, actualPath)
				}
			}
		}
	}

	// 2. Stop in engine
	if d.IsMagnet && d.MagnetSource == "alldebrid" {
		files, _, _ := s.repo.GetFiles(ctx, d.ID, 1000, 0)
		for _, f := range files {
			if f.EngineID != "" {
				s.engine.Cancel(ctx, f.EngineID)
				s.engine.Remove(ctx, f.EngineID)
			}
		}
	} else if d.EngineID != "" {
		s.engine.Cancel(ctx, d.EngineID)
		s.engine.Remove(ctx, d.EngineID)
	}

	// 3. Also stop upload if active
	if d.Status == model.StatusUploading && d.UploadJobID != "" {
		s.uploadEngine.Cancel(ctx, d.UploadJobID)
	}

	// 4. Delete files from disk
	if deleteFiles {
		for _, path := range pathsToDelete {
			// Safety check: don't delete root or obviously wrong paths
			if len(path) > 5 { // arbitary safety len
				os.RemoveAll(path)

				// Also try to remove aria2 control file
				os.Remove(path + ".aria2")
			}
		}
	}

	// 5. Clean up database
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *DownloadService) Start(ctx context.Context) {
	s.ctx = ctx

	// 1. Listen for engine events to trigger scheduler
	events := s.bus.Subscribe()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-events:
				switch ev.Type {
				case event.DownloadCreated, event.DownloadCompleted, event.DownloadError:
					// Use background context for queue processing to ensure it finishes even if trigger context is done
					s.ProcessQueue(context.Background())
				}
			}
		}
	}()

	// 2. Initial sync to catch stale tasks from previous run
	if err := s.Sync(ctx); err != nil {
		s.logger.Error("Initial sync failed", zap.Error(err))
	}

	// 3. Periodic scheduler check
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.ProcessQueue(ctx)
			case <-s.stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// 4. Initial processing
	s.ProcessQueue(ctx)
}

func (s *DownloadService) ProcessQueue(ctx context.Context) {
	s.mu.Lock()
	// Get max concurrent settings
	settings, _ := s.settingsRepo.Get(ctx)
	maxConcurrent := 3 // default
	if settings != nil {
		maxConcurrent = settings.Download.MaxConcurrentDownloads
	}

	// 1. Count active tasks in DB (including Allocating)
	dbActive, _, _ := s.repo.List(ctx, []string{string(model.StatusActive), string(model.StatusAllocating)}, 100, 0, false)
	activeCount := len(dbActive)

	if activeCount >= maxConcurrent {
		s.mu.Unlock()
		return
	}

	// 2. Fetch next waiting tasks from DB
	limit := maxConcurrent - activeCount
	waiting, _, err := s.repo.List(ctx, []string{string(model.StatusWaiting)}, limit, 0, true)
	if err != nil || len(waiting) == 0 {
		s.mu.Unlock()
		return
	}

	// Reserve slots by marking as Allocating inside the lock
	for _, d := range waiting {
		d.Status = model.StatusAllocating
		s.repo.Update(ctx, d)
	}
	s.mu.Unlock()

	for _, d := range waiting {
		// Calculate total size...
		var totalSize int64
		files, _, _ := s.repo.GetFiles(ctx, d.ID, 1000, 0)
		if len(files) > 0 {
			for _, f := range files {
				totalSize += f.Size
			}
			d.Size = totalSize
		}

		// Check disk space before submission
		if d.Size > 0 {
			free, err := s.checkDiskSpace(d.DownloadDir)
			if err == nil && free < uint64(d.Size) {
				s.logger.Warn("insufficient disk space", 
					zap.String("id", d.ID), 
					zap.Int64("required", d.Size), 
					zap.Uint64("available", free),
					zap.String("dir", d.DownloadDir))
				
				d.Status = model.StatusError
				d.Error = fmt.Sprintf("Insufficient disk space in %s (required: %d, available: %d)", d.DownloadDir, d.Size, free)
				s.repo.Update(ctx, d)
				continue
			}
		}

		// Submit to engine 
		gid := fmt.Sprintf("%016s", d.ID[2:])

		opts := engine.DownloadOptions{
			Filename:      d.Filename,
			Size:          d.Size,
			Dir:           d.DownloadDir,
			Headers:       d.Headers,
			TorrentData:   d.TorrentData,
			SelectedFiles: d.SelectedFiles,
			ID:            gid,
			MaxSpeed:      d.Options.MaxDownloadSpeed,
			Connections:   d.Options.Connections,
			Split:         d.Options.Split,
			ProxyURL:      d.Options.ProxyURL,
		}

		engineID, err := s.engine.Add(ctx, d.ResolvedURL, opts)

		if err != nil {
			d.Status = model.StatusError
			d.Error = "Queue submission failed: " + err.Error()
			s.repo.Update(ctx, d)
			s.logger.Error("failed to submit download to engine", zap.String("id", d.ID), zap.Error(err))
			continue
		}

		d.EngineID = engineID
		d.Status = model.StatusActive
		s.repo.Update(ctx, d)

		s.logger.Debug("download submitted to engine", 
			zap.String("id", d.ID), 
			zap.String("engine_id", engineID))

		s.bus.Publish(event.Event{
			Type:      event.DownloadStarted,
			Timestamp: time.Now(),
			Data:      d,
		})
	}
}

func (s *DownloadService) Sync(ctx context.Context) error {
	// Use provided context for database and engine calls
	dbActive, _, err := s.repo.List(ctx, []string{string(model.StatusActive), string(model.StatusAllocating)}, 1000, 0, false)
	if err != nil {
		return err
	}

	for _, d := range dbActive {
		s.logger.Info("resetting active/allocating download to waiting on startup", zap.String("id", d.ID))
		d.Status = model.StatusWaiting
		d.EngineID = "" // Clear EngineID to force fresh submission
		s.repo.Update(ctx, d)
	}

	// Trigger queue processing
	go s.ProcessQueue(ctx)

	return nil
}

func (s *DownloadService) handleProgress(engineID string, p engine.Progress) {
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}

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
		// Prevent overwriting terminal states
		if d.Status == model.StatusComplete || d.Status == model.StatusError {
			return
		}

		d.Downloaded = p.Downloaded
		d.Size = p.Size
		d.Speed = p.Speed
		d.ETA = p.ETA
		d.Seeders = p.Seeders
		d.Peers = p.Peers
		d.MetadataFetching = p.MetadataFetching

		s.mu.Lock()
		lastPersist := s.lastPersistMap[d.ID]
		if time.Since(lastPersist) > 10*time.Second {
			s.repo.Update(ctx, d)
			s.lastPersistMap[d.ID] = time.Now()
		}
		s.mu.Unlock()

		s.publishProgress(d)

		// Robust completion check: 
		// 1. If engine reports it's a seeder (for BitTorrent downloads)
		isComplete := p.IsSeeder
		
		// 2. We NO LONGER auto-complete HTTP/Direct downloads based on byte count here.
		//    We must wait for the engine's OnComplete event to ensure buffers are flushed and files finalized.
		//    Only exception: if it's a magnet/torrent and we've reached seeding state.

		if isComplete && d.Status == model.StatusActive {
			s.logger.Debug("auto-completing seeding download", zap.String("id", d.ID))
			go func() {
				s.engine.Cancel(context.Background(), engineID)
				s.handleComplete(engineID, "")
			}()
		}
		return
	}

	// 3. Try file within magnet (AllDebrid)
	file, err := s.repo.GetFileByEngineID(ctx, engineID)
	if err == nil {
		// Prevent overwriting terminal states
		if file.Status == model.StatusComplete || file.Status == model.StatusError {
			return
		}

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
			parent.Speed = p.Speed
			s.updateAggregateProgress(ctx, parent)
		}
		return
	}
}

func (s *DownloadService) updateAggregateProgress(ctx context.Context, d *model.Download) {
	files, _, err := s.repo.GetFiles(ctx, d.ID, 10000, 0)
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

	if d.Speed > 0 {
		remaining := totalSize - totalDownloaded
		if remaining > 0 {
			d.ETA = int(remaining / d.Speed)
		} else {
			d.ETA = 0
		}
	} else {
		d.ETA = 0
	}

	if filesComplete == len(files) && len(files) > 0 && d.Status != model.StatusComplete && d.Status != model.StatusUploading {
		if d.Destination != "" {
			d.Status = model.StatusUploading
		} else {
			d.Status = model.StatusComplete
		}
		now := time.Now()
		d.CompletedAt = &now

		s.bus.Publish(event.Event{
			Type:      event.DownloadCompleted,
			Timestamp: time.Now(),
			Data:      d,
		})

		// Force remove from engine (stop seeding)
		if d.EngineID != "" {
			s.engine.Cancel(ctx, d.EngineID)
			s.engine.Remove(ctx, d.EngineID)
		}
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
			"id":               d.ID,
			"downloaded":       d.Downloaded,
			"size":             d.Size,
			"speed":            d.Speed,
			"eta":              d.ETA,
			"seeders":          d.Seeders,
			"peers":            d.Peers,
			"metadataFetching": d.MetadataFetching,
		},
	})
}

func (s *DownloadService) handleComplete(engineID string, filePath string) {
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	d, err := s.repo.GetByEngineID(ctx, engineID)
	if err == nil {
		// Determine the actual path created by aria2
		status, err := s.engine.Status(ctx, engineID)
		if err == nil {
			if len(status.FollowedBy) > 0 {
				newGID := status.FollowedBy[0]
				d.EngineID = newGID
				d.Status = model.StatusActive
				d.Downloaded = 0
				d.UpdatedAt = time.Now()
				s.repo.Update(ctx, d)

				s.engine.Remove(ctx, engineID)
				return
			}

			// Update final stats
			d.Downloaded = status.Downloaded
			d.Size = status.Size

			if len(status.Files) > 0 {
				actualPath := status.Files[0].Path 
				if actualPath == "" {
					actualPath = filePath // fallback
				}

				rel, err := filepath.Rel(status.Dir, actualPath)
				if err == nil {
					parts := strings.Split(filepath.ToSlash(rel), "/")
					if len(parts) > 0 {
						d.DownloadDir = filepath.Join(status.Dir, parts[0])
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

		// Cleanup engine result
		s.engine.Remove(ctx, engineID)
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

		// Cleanup engine result
		s.engine.Remove(ctx, engineID)
		return
	}
}

func (s *DownloadService) GetFiles(ctx context.Context, id string) ([]model.DownloadFile, error) {
	files, _, err := s.repo.GetFiles(ctx, id, 10000, 0)
	return files, err
}

func (s *DownloadService) checkDiskSpace(path string) (uint64, error) {
	// Ensure directory exists or check parent
	dir := path
	for {
		if _, err := os.Stat(dir); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	var stat syscall.Statfs_t
	err := syscall.Statfs(dir, &stat)
	if err != nil {
		return 0, err
	}

	// Available blocks * block size
	return stat.Bavail * uint64(stat.Bsize), nil
}

func (s *DownloadService) handleError(engineID string, err error) {
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}

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

		s.engine.Remove(ctx, engineID)
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

		s.engine.Remove(ctx, engineID)
		return
	}
}
