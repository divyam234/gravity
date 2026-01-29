package service

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/logger"
	"gravity/internal/model"
	"gravity/internal/store"
	"gravity/internal/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	SettingsWatchInterval = 30 * time.Second
	RetryBackoffBase      = 30 * time.Second
	RetryBackoffCap       = 30 * time.Minute
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

func NewDownloadService(repo *store.DownloadRepo, settingsRepo *store.SettingsRepo, eng engine.DownloadEngine, ue engine.UploadEngine, bus *event.Bus, provider *ProviderService) *DownloadService {
	s := &DownloadService{
		repo:           repo,
		settingsRepo:   settingsRepo,
		engine:         eng,
		uploadEngine:   ue,
		bus:            bus,
		provider:       provider,
		logger:         logger.Component("DOWNLOAD"),
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

func (s *DownloadService) Create(ctx context.Context, d *model.Download) (*model.Download, error) {
	if err := d.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	res, providerName, err := s.provider.Resolve(ctx, d.URL, d.Headers, d.TorrentData)
	if err != nil {
		s.logger.Warn("failed to resolve URL", zap.String("url", d.URL), zap.Error(err))
		return nil, fmt.Errorf("failed to resolve URL: %w", err)
	}

	// Apply identity and basic fields
	d.ID = "d_" + uuid.New().String()[:8]
	d.Provider = providerName
	if err := d.TransitionTo(model.StatusWaiting); err != nil {
		return nil, err
	}
	d.CreatedAt = time.Now()
	d.UpdatedAt = time.Now()

	// Do NOT merge global settings here. d.Dir, d.Split, etc., should remain
	// as provided in the request (zero/nil if using defaults).

	if d.Filename == "" {
		d.Filename = res.Name
	} else if !utils.IsSafeFilename(d.Filename) {
		return nil, fmt.Errorf("invalid filename")
	}

	d.ResolvedURL = res.URL
	d.Headers = res.Headers
	d.Size = res.Size
	d.MagnetHash = res.Hash
	d.IsMagnet = res.IsMagnet
	d.ExecutionMode = res.ExecutionMode

	if res.IsMagnet {
		if len(res.Files) > 0 {
			var totalSize int64
			var selectedIndexes []int
			for _, f := range res.Files {
				if len(d.SelectedFiles) > 0 {
					found := slices.Contains(d.SelectedFiles, f.Index)
					// Allow index 0
					if !found {
						continue
					}
				}
				d.Files = append(d.Files, model.DownloadFile{
					ID:     "df_" + uuid.New().String()[:8],
					Name:   f.Name,
					Path:   f.Path,
					Size:   f.Size,
					Status: model.StatusWaiting,
					URL:    f.URL,
					Index:  f.Index,
				})
				totalSize += f.Size

				selectedIndexes = append(selectedIndexes, f.Index)
			}
			d.Size = totalSize
			d.SelectedFiles = selectedIndexes
		}
	}

	if err := s.repo.Create(ctx, d); err != nil {
		s.logger.Error("failed to save download to DB", zap.String("id", d.ID), zap.Error(err))
		return nil, err
	}

	s.logger.Info("download created",
		zap.String("id", d.ID),
		zap.String("filename", d.Filename),
		zap.String("provider", d.Provider),
		zap.String("mode", string(res.ExecutionMode)))

	s.bus.PublishLifecycle(event.LifecycleEvent{
		Type:      event.DownloadCreated,
		ID:        d.ID,
		Data:      d,
		Timestamp: time.Now(),
	})

	switch res.ExecutionMode {
	case model.ExecutionModeDebridFiles:
		if err := d.TransitionTo(model.StatusActive); err != nil {
			s.logger.Error("failed to activate debrid download", zap.Error(err))
		}
		s.repo.Update(ctx, d)

		// Use service lifecycle context for background task
		if s.ctx != nil {
			go s.startDebridDownload(s.ctx, d)
		} else {
			s.logger.Warn("service context not initialized, skipping debrid download")
			_ = d.TransitionTo(model.StatusError)
			d.Error = "Service not ready"
			s.repo.Update(ctx, d)
		}
	default: // Direct or Magnet (handled by engine)
		// Signal queue to check for work (Standard Flow)
		s.signalQueueCheck()
	}

	return d, nil
}

// startDebridDownload downloads files via Provider direct links
func (s *DownloadService) startDebridDownload(ctx context.Context, d *model.Download) {
	// Resolve options to get the effective directory
	settings, _ := s.settingsRepo.Get(ctx)
	resolver := engine.NewOptionResolver(settings)
	effectiveOpts := resolver.Resolve(engine.FromModel(d))

	// Download each file in parallel via aria2
	for i := range d.Files {
		file := &d.Files[i]
		if file.URL == "" {
			continue
		}

		// Unlock the link first (if needed, or re-resolve)
		// We use provider service to resolve/unlock
		resolved, _, err := s.provider.Resolve(ctx, file.URL, nil, "")
		if err != nil {
			_ = d.Files[i].TransitionTo(model.StatusError)
			d.Files[i].Error = "Link unlock failed: " + err.Error()
			s.repo.UpdateFile(ctx, d.ID, &d.Files[i])
			continue
		}

		// Add to aria2
		// Prepare execution options
		execOpts := effectiveOpts.DownloadOptions
		execOpts.DownloadDir = effectiveOpts.LocalPath // Enforce resolved path
		execOpts.Filename = file.Path                  // Preserve structure

		// Use gid:index format so handleProgress can attribute progress to the correct file
		parentGID := fmt.Sprintf("%016s", d.ID[2:])
		execOpts.ID = fmt.Sprintf("%s:%d", parentGID, file.Index)

		_, err = s.engine.Add(ctx, resolved.URL, execOpts)

		if err != nil {
			_ = d.Files[i].TransitionTo(model.StatusError)
			d.Files[i].Error = err.Error()
			s.repo.UpdateFile(ctx, d.ID, &d.Files[i])
			continue
		}
	}
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

			if status.Status == "resolving" {
				if err := d.TransitionTo(model.StatusResolving); err != nil {
					// log warning
				}
			}

			// Also fetch detailed peers for active aria2 downloads
			if d.Engine == "aria2" || d.IsMagnet {
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

					if live.Status == "resolving" {
						_ = d.TransitionTo(model.StatusResolving)
					}
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

	if err := d.TransitionTo(model.StatusPaused); err != nil {
		return err
	}
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

	if err := d.TransitionTo(model.StatusWaiting); err != nil {
		return err
	}
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

	// Re-add to engine using resolved options
	settings, _ := s.settingsRepo.Get(ctx)
	resolver := engine.NewOptionResolver(settings)
	effectiveOpts := resolver.Resolve(engine.FromModel(d))

	execOpts := effectiveOpts.DownloadOptions
	execOpts.DownloadDir = effectiveOpts.LocalPath
	execOpts.Filename = d.Filename

	engineID, err := s.engine.Add(ctx, d.ResolvedURL, execOpts)
	if err != nil {
		return err
	}

	d.EngineID = engineID
	if err := d.TransitionTo(model.StatusActive); err != nil {
		return err
	}
	d.Error = ""
	d.Downloaded = 0
	d.UpdatedAt = time.Now()

	// Reset file statuses
	for i := range d.Files {
		_ = d.Files[i].TransitionTo(model.StatusWaiting)
		d.Files[i].Error = ""
		d.Files[i].Downloaded = 0
	}

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

func (s *DownloadService) Batch(ctx context.Context, ids []string, action string) error {
	var errs []error
	for _, id := range ids {
		var err error
		switch action {
		case "pause":
			err = s.Pause(ctx, id)
		case "resume":
			err = s.Resume(ctx, id)
		case "delete":
			err = s.Delete(ctx, id, false)
		case "retry":
			err = s.Retry(ctx, id)
		default:
			return fmt.Errorf("unknown action: %s", action)
		}
		if err != nil {
			s.logger.Warn("batch operation failed for item", zap.String("id", id), zap.String("action", action), zap.Error(err))
			errs = append(errs, fmt.Errorf("%s: %w", id, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("batch operation failed for %d/%d items", len(errs), len(ids))
	}
	return nil
}

func (s *DownloadService) UpdatePriority(ctx context.Context, id string, priority int) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	d.Priority = priority
	d.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, d); err != nil {
		return err
	}

	s.logger.Info("download priority updated",
		zap.String("id", id),
		zap.Int("priority", priority))

	return nil
}

func (s *DownloadService) Update(ctx context.Context, id string, filename, destination *string, priority, maxRetries *int) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if (filename != nil || destination != nil) && d.Status == model.StatusActive {
		return fmt.Errorf("cannot update filename or destination while download is active")
	}

	if filename != nil {
		if !utils.IsSafeFilename(*filename) {
			return fmt.Errorf("invalid filename")
		}
		d.Filename = *filename
	}
	if destination != nil {
		d.Destination = *destination
	}
	if priority != nil {
		d.Priority = *priority
	}
	if maxRetries != nil {
		d.MaxRetries = *maxRetries
	}

	d.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, d); err != nil {
		return err
	}

	s.logger.Info("download updated", zap.String("id", id))
	return nil
}

func (s *DownloadService) Delete(ctx context.Context, id string, deleteFiles bool) error {
	s.progressBuffer.remove(id)

	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	// 1. Resolve paths for deletion before stopping engine
	var pathsToDelete []string
	if deleteFiles {
		// We must resolve the effective directory if it was default
		// in order to delete the correct path.
		if d.Dir == "" {
			settings, _ := s.settingsRepo.Get(ctx)
			resolver := engine.NewOptionResolver(settings)
			eff := resolver.Resolve(engine.FromModel(d))
			pathsToDelete = append(pathsToDelete, eff.LocalPath)
		} else {
			pathsToDelete = append(pathsToDelete, d.Dir)
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
				duplicate := slices.Contains(pathsToDelete, actualPath)
				if !duplicate && actualPath != "" {
					pathsToDelete = append(pathsToDelete, actualPath)
				}
			}
		}
	}

	// 2. Stop in engine
	if d.EngineID != "" {
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

	// 6. Settings watcher
	go s.watchSettings()
}

func (s *DownloadService) watchSettings() {
	ticker := time.NewTicker(SettingsWatchInterval)
	defer ticker.Stop()

	var lastMaxConcurrent int

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			settings, err := s.settingsRepo.Get(s.ctx)
			if err != nil || settings == nil {
				continue
			}

			newMax := settings.Download.MaxConcurrentDownloads
			if newMax != lastMaxConcurrent && newMax > 0 {
				s.resizeSemaphore(newMax)
				lastMaxConcurrent = newMax
			}
		}
	}
}

func (s *DownloadService) resizeSemaphore(newSize int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create new semaphore
	newSem := make(chan struct{}, newSize)

	// Transfer existing slots (non-blocking)
	for {
		select {
		case <-s.slotSem:
			select {
			case newSem <- struct{}{}:
			default:
				// New semaphore full, put back
				s.slotSem <- struct{}{}
				goto done
			}
		default:
			goto done
		}
	}
done:
	s.slotSem = newSem
	s.logger.Info("queue size updated", zap.Int("max_concurrent", newSize))
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
	if err := d.TransitionTo(model.StatusAllocating); err != nil {
		s.logger.Error("failed to transition to allocating", zap.Error(err))
		return nil
	}
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
	if len(d.Files) > 0 {
		for _, f := range d.Files {
			totalSize += f.Size
		}
		d.Size = totalSize
	}

	// Prepare options using runtime resolution
	settings, _ := s.settingsRepo.Get(ctx)
	resolver := engine.NewOptionResolver(settings)
	effectiveOpts := resolver.Resolve(engine.FromModel(d))

	execOpts := effectiveOpts.DownloadOptions
	execOpts.DownloadDir = effectiveOpts.LocalPath // Enforce resolved path

	// Check disk space before submission
	if d.Size > 0 {
		free, err := s.checkDiskSpace(execOpts.DownloadDir)
		if err == nil && free < uint64(d.Size) {
			s.logger.Warn("insufficient disk space",
				zap.String("id", d.ID),
				zap.Int64("required", d.Size),
				zap.Uint64("available", free),
				zap.String("dir", execOpts.DownloadDir))

			if err := d.TransitionTo(model.StatusError); err != nil {
				s.logger.Error("failed to transition to error", zap.Error(err))
			}
			d.Error = fmt.Sprintf("Insufficient disk space in %s (required: %d, available: %d)", execOpts.DownloadDir, d.Size, free)
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

	// Double-check status before submission to handle race conditions (e.g. user paused during allocation)
	if fresh, err := s.repo.Get(ctx, d.ID); err == nil {
		if fresh.Status == model.StatusPaused {
			s.logger.Info("download paused during allocation, aborting execution", zap.String("id", d.ID))
			return
		}
		if fresh.Status == model.StatusError {
			s.logger.Info("download error during allocation, aborting execution", zap.String("id", d.ID))
			return
		}
	}

	// Route based on ExecutionMode
	if d.ExecutionMode == model.ExecutionModeDebridFiles {
		// Transition to Active before starting background task
		if err := d.TransitionTo(model.StatusActive); err != nil {
			s.logger.Error("failed to activate debrid download", zap.Error(err))
			// Continue anyway? Or abort? If we can't transition to Active, something is wrong.
			// But startDebridDownload will try to update files.
		}
		s.repo.Update(ctx, d)

		go s.startDebridDownload(ctx, d)
		return
	}

	// Submit to engine
	gid := "00000000" + d.ID[2:]

	execOpts.Filename = d.Filename
	execOpts.Size = d.Size
	execOpts.ID = gid

	engineID, err := s.engine.Add(ctx, d.ResolvedURL, execOpts)

	if err != nil {
		if err := d.TransitionTo(model.StatusError); err != nil {
			s.logger.Error("failed to transition to error", zap.Error(err))
		}
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
	if err := d.TransitionTo(model.StatusActive); err != nil {
		s.logger.Error("failed to transition to active", zap.Error(err))
	}
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
		if err := d.TransitionTo(model.StatusWaiting); err != nil {
			s.logger.Warn("failed to transition download state", zap.String("id", d.ID), zap.Error(err))
			// Force update if transition fails? Or just continue?
			// Since this is startup recovery, we might want to force it or handle the error.
			// For now, let's log it. Ideally we should force it if the state machine is out of sync.
			d.Status = model.StatusWaiting
		}
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
				found := false
				for i := range parent.Files {
					if parent.Files[i].Index == fileIndex {
						file := &parent.Files[i]
						file.Downloaded = p.Downloaded
						file.Progress = 0
						if file.Size > 0 {
							file.Progress = int((file.Downloaded * 100) / file.Size)
						}

						if file.Progress == 100 {
							if file.Status != model.StatusComplete {
								_ = file.TransitionTo(model.StatusComplete)
							}
						} else {
							if file.Status != model.StatusActive {
								_ = file.TransitionTo(model.StatusActive)
							}
						}
						found = true
						break
					}
				}

				if found {
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

	// 3. Try file within magnet (embedded in another download)
	// Usually magnets are handled by 1. if engineID is gid:index
	// But some engines might return just a sub-GID.
}

func (s *DownloadService) updateAggregateProgress(ctx context.Context, d *model.Download) {
	var totalDownloaded int64
	var totalSize int64
	filesComplete := 0

	for _, f := range d.Files {
		totalDownloaded += f.Downloaded
		totalSize += f.Size
		if f.Status == model.StatusComplete {
			filesComplete++
		}
	}

	d.Downloaded = totalDownloaded
	d.Size = totalSize
	d.ETA = event.CalculateETA(totalSize-totalDownloaded, d.Speed)

	// Override with accurate engine stats if available
	if d.Status == model.StatusActive && d.EngineID != "" {
		if status, err := s.engine.Status(ctx, d.EngineID); err == nil {
			d.Downloaded = status.Downloaded
			d.Speed = status.Speed
			d.ETA = status.Eta
		}
	}

	if filesComplete == len(d.Files) && len(d.Files) > 0 && d.Status != model.StatusComplete && d.Status != model.StatusUploading {
		if d.Destination != "" {
			d.TransitionTo(model.StatusUploading)
		} else {
			d.TransitionTo(model.StatusComplete)
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
		ID:         d.ID,
		Type:       "download",
		Downloaded: d.Downloaded,
		Size:       d.Size,
		Speed:      d.Speed,
		ETA:        d.ETA,
		Seeders:    d.Seeders,
		Peers:      d.Peers,
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
				if err := d.TransitionTo(model.StatusActive); err != nil {
					s.logger.Warn("transition failed", zap.Error(err))
				}
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
						d.Dir = filepath.Join(status.Dir, parts[0])
					}
				}
			}
		}

		if d.Destination != "" {
			d.TransitionTo(model.StatusUploading)
		} else {
			d.TransitionTo(model.StatusComplete)
			// Ensure downloaded amount matches size when complete
			d.Downloaded = d.Size
		}

		now := time.Now()
		d.CompletedAt = &now
		for i := range d.Files {
			if d.Files[i].Status != model.StatusComplete {
				_ = d.Files[i].TransitionTo(model.StatusComplete)
			}
			d.Files[i].Downloaded = d.Files[i].Size
			d.Files[i].Progress = 100
		}
		s.repo.Update(ctx, d)
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

	// Sub-files are handled in handleProgress via gid:index usually
}

func (s *DownloadService) GetFiles(ctx context.Context, id string) ([]model.DownloadFile, error) {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return d.Files, nil
}

func (s *DownloadService) checkDiskSpace(path string) (uint64, error) {
	dir := path
	if dir == "" {
		// If path is empty (not yet resolved or root), check current working dir or similar
		// But in executeDownload it should be resolved.
		// If we are checking before resolution, default to "."
		dir = "."
	}
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
		if isRetryableError(err) && d.RetryCount < d.MaxRetries {
			s.scheduleRetry(ctx, d)
			s.engine.Remove(ctx, engineID)
			return
		}

		if err := d.TransitionTo(model.StatusError); err != nil {
			s.logger.Warn("transition failed", zap.Error(err))
		}
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

	// Handle sub-file error
	if strings.Contains(engineID, ":") {
		parts := strings.Split(engineID, ":")
		if len(parts) == 2 {
			parentGID := parts[0]
			fileIndex, _ := strconv.Atoi(parts[1])

			parent, parentErr := s.repo.GetByEngineID(ctx, parentGID)
			if parentErr == nil {
				for i := range parent.Files {
					if parent.Files[i].Index == fileIndex {
						_ = parent.Files[i].TransitionTo(model.StatusError)
						parent.Files[i].Error = err.Error()
						break
					}
				}
				s.updateAggregateProgress(ctx, parent)
				s.engine.Remove(ctx, engineID)
				return
			}
		}
	}
}

func calculateBackoff(retryCount int) time.Duration {
	backoff := time.Duration(float64(RetryBackoffBase) * math.Pow(2, float64(retryCount)))
	if backoff > RetryBackoffCap {
		backoff = RetryBackoffCap
	}
	return backoff
}

func (s *DownloadService) scheduleRetry(ctx context.Context, d *model.Download) {
	if d.RetryCount >= d.MaxRetries {
		s.logger.Info("max retries reached", zap.String("id", d.ID))
		return
	}

	// Exponential backoff: 30s, 1m, 2m, 4m, 8m...
	backoff := calculateBackoff(d.RetryCount)

	nextRetry := time.Now().Add(backoff)
	d.RetryCount++
	d.NextRetryAt = &nextRetry
	if err := d.TransitionTo(model.StatusWaiting); err != nil {
		s.logger.Error("failed to transition to waiting for retry", zap.String("id", d.ID), zap.Error(err))
		// If transition fails, we probably shouldn't proceed with retry update,
		// but since we want to recover, maybe force it?
		d.Status = model.StatusWaiting
	}
	// Reset file statuses so they can be picked up again
	for i := range d.Files {
		_ = d.Files[i].TransitionTo(model.StatusWaiting)
		d.Files[i].Error = ""
	}
	s.repo.Update(ctx, d)

	s.logger.Info("scheduled retry",
		zap.String("id", d.ID),
		zap.Int("attempt", d.RetryCount),
		zap.Time("next_retry", nextRetry))

	// Schedule wake-up
	go func() {
		select {
		case <-time.After(backoff):
			s.signalQueueCheck()
		case <-s.ctx.Done():
		}
	}()
}

func isRetryableError(err error) bool {
	errStr := err.Error()
	retryablePatterns := []string{
		"timeout", "timed out", "connection refused", "connection reset", "temporary failure",
		"503", "502", "429", "rate limit",
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}
	return false
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
