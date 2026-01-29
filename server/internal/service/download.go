package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/store"

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
	ctx    context.Context
	cancel context.CancelFunc

	// Semaphore-based slot management (fixes race condition)
	slotSem   chan struct{}
	queueWake chan struct{}

	// Throttling for database writes
	progressBuffer *progressBuffer
	mu             sync.RWMutex
	stop           chan struct{}
}

// progressBuffer handles batched progress persistence
type progressBuffer struct {
	mu        sync.Mutex
	downloads map[string]*model.Download
	dirty     map[string]time.Time
	repo      *store.DownloadRepo
}

func newProgressBuffer(repo *store.DownloadRepo) *progressBuffer {
	return &progressBuffer{
		downloads: make(map[string]*model.Download),
		dirty:     make(map[string]time.Time),
		repo:      repo,
	}
}

func (pb *progressBuffer) update(d *model.Download) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.downloads[d.ID] = d
	pb.dirty[d.ID] = time.Now()
}

func (pb *progressBuffer) get(id string) *model.Download {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	return pb.downloads[id]
}

func (pb *progressBuffer) remove(id string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	delete(pb.downloads, id)
	delete(pb.dirty, id)
}

func (pb *progressBuffer) flush(ctx context.Context, maxAge time.Duration) {
	pb.mu.Lock()
	now := time.Now()
	toFlush := make([]*model.Download, 0)
	for id, lastDirty := range pb.dirty {
		if now.Sub(lastDirty) >= maxAge {
			if d, ok := pb.downloads[id]; ok {
				toFlush = append(toFlush, d)
			}
			delete(pb.dirty, id)
		}
	}
	pb.mu.Unlock()

	for _, d := range toFlush {
		pb.repo.Update(ctx, d)
	}
}

func (pb *progressBuffer) flushAll(ctx context.Context) {
	pb.mu.Lock()
	toFlush := make([]*model.Download, 0, len(pb.downloads))
	for _, d := range pb.downloads {
		toFlush = append(toFlush, d)
	}
	pb.dirty = make(map[string]time.Time)
	pb.mu.Unlock()

	for _, d := range toFlush {
		pb.repo.Update(ctx, d)
	}
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
		progressBuffer: newProgressBuffer(repo),
		stop:           make(chan struct{}),
		queueWake:      make(chan struct{}, 1),
	}

	// Wire up engine events
	eng.OnProgress(s.handleProgress)
	eng.OnComplete(s.handleComplete)
	eng.OnError(s.handleError)

	return s
}

func (s *DownloadService) Create(ctx context.Context, url string, filename string, options model.TaskOptions) (*model.Download, error) {
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
	downloadDir := options.DownloadDir

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
		Destination: options.Destination, // Remote upload destination
		DownloadDir: localPath,           // Actual local download directory
		Status:      model.StatusWaiting,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Options:     options, // Store all task options in JSON
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

	s.bus.PublishLifecycle(event.LifecycleEvent{
		Type:      event.DownloadCreated,
		ID:        d.ID,
		Data:      d,
		Timestamp: time.Now(),
	})

	// Signal queue to check for work
	s.signalQueueCheck()

	return d, nil
}

func (s *DownloadService) Get(ctx context.Context, id string) (*model.Download, error) {
	// Check progress buffer first for latest data
	if buffered := s.progressBuffer.get(id); buffered != nil {
		return buffered, nil
	}

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
		// Capture latest stats before stopping
		if status, err := s.engine.Status(ctx, d.EngineID); err == nil {
			d.Downloaded = status.Downloaded
			d.Size = status.Size
		}

		s.engine.Cancel(ctx, d.EngineID)
		s.engine.Remove(ctx, d.EngineID)
	}

	d.Status = model.StatusPaused
	d.EngineID = "" // Clear GID as it's no longer in engine
	if err := s.repo.Update(ctx, d); err != nil {
		return err
	}

	s.progressBuffer.remove(id)

	s.bus.PublishLifecycle(event.LifecycleEvent{
		Type:      event.DownloadPaused,
		ID:        id,
		Data:      map[string]string{"id": id},
		Timestamp: time.Now(),
	})

	// Signal queue to fill the slot
	s.signalQueueCheck()

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

	s.bus.PublishLifecycle(event.LifecycleEvent{
		Type:      event.DownloadResumed,
		ID:        id,
		Data:      map[string]string{"id": id},
		Timestamp: time.Now(),
	})

	// Signal queue
	s.signalQueueCheck()

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
	engineOpts := toEngineOptionsFromDownload(d)
	engineOpts.Filename = d.Filename
	engineID, err := s.engine.Add(ctx, d.ResolvedURL, engineOpts)
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

	s.bus.PublishLifecycle(event.LifecycleEvent{
		Type:      event.DownloadStarted,
		ID:        d.ID,
		Data:      d,
		Timestamp: time.Now(),
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
				actualPath := status.Files[0].Path
				if actualPath == "" {
					actualPath = filepath.Join(status.Dir, status.Filename)
				}

				// Add to deletion list if not already there
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
			if len(path) > 5 { // Safety check
				os.RemoveAll(path)
				os.Remove(path + ".aria2")
			}
		}
	}

	// 5. Clean up
	s.progressBuffer.remove(id)

	// 6. Clean up database
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *DownloadService) Start(ctx context.Context) {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Initialize semaphore based on settings
	settings, _ := s.settingsRepo.Get(ctx)
	maxConcurrent := 3
	if settings != nil && settings.Download.MaxConcurrentDownloads > 0 {
		maxConcurrent = settings.Download.MaxConcurrentDownloads
	}
	s.slotSem = make(chan struct{}, maxConcurrent)

	// 1. Listen for lifecycle events to trigger queue processing
	lifecycle := s.bus.SubscribeLifecycle()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("panic in lifecycle listener", zap.Any("panic", r))
			}
			s.bus.UnsubscribeLifecycle(lifecycle)
		}()
		for {
			select {
			case <-s.ctx.Done():
				return
			case ev, ok := <-lifecycle:
				if !ok {
					return
				}
				switch ev.Type {
				case event.DownloadCreated, event.DownloadCompleted, event.DownloadError:
					s.signalQueueCheck()
				}
			}
		}
	}()

	// 2. Queue processor goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("panic in queue processor", zap.Any("panic", r))
			}
		}()
		s.queueProcessor()
	}()

	// 3. Progress flush goroutine (every 10 seconds)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("panic in progress flusher", zap.Any("panic", r))
			}
		}()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				s.progressBuffer.flushAll(context.Background())
				return
			case <-ticker.C:
				s.progressBuffer.flush(s.ctx, 10*time.Second)
			}
		}
	}()

	// 4. Initial sync
	if err := s.Sync(ctx); err != nil {
		s.logger.Error("Initial sync failed", zap.Error(err))
	}

	// 5. Initial queue processing
	s.signalQueueCheck()
}

func (s *DownloadService) signalQueueCheck() {
	select {
	case s.queueWake <- struct{}{}:
	default: // Already signaled
	}
}

func (s *DownloadService) queueProcessor() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.queueWake:
			s.processQueueOnce()
		}
	}
}

func (s *DownloadService) processQueueOnce() {
	for {
		// Try to acquire a slot
		select {
		case s.slotSem <- struct{}{}:
			// Got a slot
		case <-s.ctx.Done():
			return
		default:
			// No slots available
			return
		}

		// Atomically claim next waiting task
		d := s.claimNextWaiting()
		if d == nil {
			<-s.slotSem // Release slot, nothing to do
			return
		}

		// Execute download in goroutine
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("panic in executeDownload", zap.Any("panic", r))
				}
			}()
			s.executeDownload(d)
		}()
	}
}

func (s *DownloadService) claimNextWaiting() *model.Download {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	waiting, _, err := s.repo.List(ctx, []string{string(model.StatusWaiting)}, 1, 0, true)
	if err != nil || len(waiting) == 0 {
		return nil
	}

	d := waiting[0]
	d.Status = model.StatusAllocating
	s.repo.Update(ctx, d)
	return d
}

func (s *DownloadService) executeDownload(d *model.Download) {
	defer func() { <-s.slotSem }() // Release slot on completion

	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// Calculate total size for multi-file downloads
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

			s.bus.PublishLifecycle(event.LifecycleEvent{
				Type:      event.DownloadError,
				ID:        d.ID,
				Error:     d.Error,
				Timestamp: time.Now(),
			})
			return
		}
	}

	// Submit to engine
	gid := fmt.Sprintf("%016s", d.ID[2:])

	opts := toEngineOptionsFromDownload(d)
	opts.Filename = d.Filename
	opts.Size = d.Size
	opts.Headers = d.Headers
	opts.TorrentData = d.TorrentData
	opts.ID = gid

	engineID, err := s.engine.Add(ctx, d.ResolvedURL, opts)

	if err != nil {
		d.Status = model.StatusError
		d.Error = "Queue submission failed: " + err.Error()
		s.repo.Update(ctx, d)
		s.logger.Error("failed to submit download to engine", zap.String("id", d.ID), zap.Error(err))

		s.bus.PublishLifecycle(event.LifecycleEvent{
			Type:      event.DownloadError,
			ID:        d.ID,
			Error:     d.Error,
			Timestamp: time.Now(),
		})
		return
	}

	d.EngineID = engineID
	d.Status = model.StatusActive
	s.repo.Update(ctx, d)

	s.logger.Debug("download submitted to engine",
		zap.String("id", d.ID),
		zap.String("engine_id", engineID))

	s.bus.PublishLifecycle(event.LifecycleEvent{
		Type:      event.DownloadStarted,
		ID:        d.ID,
		Data:      d,
		Timestamp: time.Now(),
	})
}

// ProcessQueue is kept for backward compatibility but delegates to new system
func (s *DownloadService) ProcessQueue(ctx context.Context) {
	s.signalQueueCheck()
}

func (s *DownloadService) Sync(ctx context.Context) error {
	dbActive, _, err := s.repo.List(ctx, []string{string(model.StatusActive), string(model.StatusAllocating)}, 1000, 0, false)
	if err != nil {
		return err
	}

	for _, d := range dbActive {
		s.logger.Info("resetting active/allocating download to waiting on startup", zap.String("id", d.ID))
		d.Status = model.StatusWaiting
		d.EngineID = ""
		s.repo.Update(ctx, d)
	}

	s.signalQueueCheck()

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

					if file.Status != oldStatus {
						s.repo.UpdateFile(ctx, file)
					}

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
		d.ETA = event.CalculateETA(p.Size-p.Downloaded, p.Speed)
		d.Seeders = p.Seeders
		d.Peers = p.Peers
		d.MetadataFetching = p.MetadataFetching

		// Update progress buffer (will be flushed periodically)
		s.progressBuffer.update(d)

		s.publishProgress(d)

		// Robust completion check for BitTorrent downloads
		if p.IsSeeder && d.Status == model.StatusActive {
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

		if file.Status != oldStatus {
			s.repo.UpdateFile(ctx, file)
		}

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
	d.ETA = event.CalculateETA(totalSize-totalDownloaded, d.Speed)

	// Override with accurate engine stats if available
	if d.Status == model.StatusActive && d.EngineID != "" {
		if status, err := s.engine.Status(ctx, d.EngineID); err == nil {
			d.Downloaded = status.Downloaded
			d.Speed = status.Speed
			d.ETA = status.Eta
		}
	}

	if filesComplete == len(files) && len(files) > 0 && d.Status != model.StatusComplete && d.Status != model.StatusUploading {
		if d.Destination != "" {
			d.Status = model.StatusUploading
		} else {
			d.Status = model.StatusComplete
			d.Downloaded = d.Size
		}
		now := time.Now()
		d.CompletedAt = &now

		s.bus.PublishLifecycle(event.LifecycleEvent{
			Type:      event.DownloadCompleted,
			ID:        d.ID,
			Data:      d,
			Timestamp: time.Now(),
		})

		// Force remove from engine (stop seeding)
		if d.EngineID != "" {
			s.engine.Cancel(ctx, d.EngineID)
			s.engine.Remove(ctx, d.EngineID)
		}

		s.repo.Update(ctx, d)
		s.progressBuffer.remove(d.ID)
	} else {
		s.progressBuffer.update(d)
	}

	s.publishProgress(d)
}

func (s *DownloadService) publishProgress(d *model.Download) {
	s.bus.PublishProgress(event.ProgressEvent{
		ID:               d.ID,
		Type:             "download",
		Downloaded:       d.Downloaded,
		Size:             d.Size,
		Speed:            d.Speed,
		ETA:              d.ETA,
		Seeders:          d.Seeders,
		Peers:            d.Peers,
		MetadataFetching: d.MetadataFetching,
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
					actualPath = filePath
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
			// Ensure downloaded amount matches size when complete
			d.Downloaded = d.Size
		}

		now := time.Now()
		d.CompletedAt = &now
		s.repo.Update(ctx, d)
		s.repo.MarkAllFilesComplete(ctx, d.ID)
		s.progressBuffer.remove(d.ID)

		s.bus.PublishLifecycle(event.LifecycleEvent{
			Type:      event.DownloadCompleted,
			ID:        d.ID,
			Data:      d,
			Timestamp: time.Now(),
		})

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

		s.engine.Remove(ctx, engineID)
		return
	}
}

func (s *DownloadService) GetFiles(ctx context.Context, id string) ([]model.DownloadFile, error) {
	files, _, err := s.repo.GetFiles(ctx, id, 10000, 0)
	return files, err
}

func (s *DownloadService) checkDiskSpace(path string) (uint64, error) {
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
		s.progressBuffer.remove(d.ID)

		s.bus.PublishLifecycle(event.LifecycleEvent{
			Type:      event.DownloadError,
			ID:        d.ID,
			Error:     d.Error,
			Data:      map[string]string{"id": d.ID, "error": d.Error},
			Timestamp: time.Now(),
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

// StopGracefully signals background routines to stop
func (s *DownloadService) StopGracefully() {
	if s.cancel != nil {
		s.cancel()
	}
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}

// Stop is an alias for StopGracefully for test compatibility
func (s *DownloadService) Stop() {
	s.StopGracefully()
}
