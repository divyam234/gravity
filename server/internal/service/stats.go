package service

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/store"

	"go.uber.org/zap"
)

// smoothedSpeed tracks speed with exponential moving average and decay
type smoothedSpeed struct {
	current   int64
	smoothed  int64
	alpha     float64
	updatedAt time.Time
	decaying  bool
}

// SpeedTracker manages speed calculations with smoothing and decay
type SpeedTracker struct {
	speeds sync.Map // map[string]*smoothedSpeed
	mu     sync.Mutex
}

func NewSpeedTracker() *SpeedTracker {
	return &SpeedTracker{}
}

func (t *SpeedTracker) Update(id string, speed int64) {
	raw, _ := t.speeds.LoadOrStore(id, &smoothedSpeed{alpha: 0.3})
	s := raw.(*smoothedSpeed)

	t.mu.Lock()
	defer t.mu.Unlock()

	s.decaying = false
	if s.smoothed == 0 {
		s.smoothed = speed
	} else {
		// Exponential moving average
		s.smoothed = int64(s.alpha*float64(speed) + (1-s.alpha)*float64(s.smoothed))
	}
	s.current = speed
	s.updatedAt = time.Now()
}

func (t *SpeedTracker) OnComplete(id string) {
	raw, ok := t.speeds.Load(id)
	if !ok {
		return
	}
	s := raw.(*smoothedSpeed)

	t.mu.Lock()
	s.decaying = true
	t.mu.Unlock()

	// Decay over 5 seconds
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(time.Second)
			t.mu.Lock()
			if !s.decaying {
				t.mu.Unlock()
				return // Task restarted or removed
			}
			s.smoothed = int64(float64(s.smoothed) * 0.5)
			t.mu.Unlock()
		}
		t.speeds.Delete(id)
	}()
}

func (t *SpeedTracker) Remove(id string) {
	t.speeds.Delete(id)
}

func (t *SpeedTracker) GetGlobalSpeed() int64 {
	var total int64
	cutoff := time.Now().Add(-5 * time.Second)

	t.speeds.Range(func(key, value any) bool {
		s := value.(*smoothedSpeed)
		t.mu.Lock()
		if s.updatedAt.After(cutoff) {
			total += s.smoothed
		}
		t.mu.Unlock()
		return true
	})
	return total
}

type StatsService struct {
	repo           *store.StatsRepo
	settingsRepo   *store.SettingsRepo
	downloadRepo   *store.DownloadRepo
	downloadEngine engine.DownloadEngine
	uploadEngine   engine.UploadEngine
	bus            *event.Bus
	ctx            context.Context
	logger         *zap.Logger

	// Speed tracking with smoothing
	downloadSpeeds *SpeedTracker
	uploadSpeeds   *SpeedTracker

	// Batched stats persistence
	pendingDownloaded atomic.Int64
	pendingUploaded   atomic.Int64
	lastDownloaded    sync.Map // map[string]int64
	lastUploaded      sync.Map // map[string]int64

	// Session tracking
	startTime       time.Time
	startDownloaded int64
	startUploaded   int64

	mu            sync.Mutex
	pollingPaused bool
	pollingCond   *sync.Cond
	done          chan struct{}
	trigger       chan struct{}
}

func NewStatsService(repo *store.StatsRepo, setr *store.SettingsRepo, dr *store.DownloadRepo, de engine.DownloadEngine, ue engine.UploadEngine, bus *event.Bus, l *zap.Logger) *StatsService {
	s := &StatsService{
		repo:           repo,
		settingsRepo:   setr,
		downloadRepo:   dr,
		downloadEngine: de,
		uploadEngine:   ue,
		bus:            bus,
		logger:         l.With(zap.String("service", "stats")),
		pollingPaused:  true,
		startTime:      time.Now(),
		done:           make(chan struct{}),
		trigger:        make(chan struct{}, 1),
		downloadSpeeds: NewSpeedTracker(),
		uploadSpeeds:   NewSpeedTracker(),
	}
	s.pollingCond = sync.NewCond(&s.mu)
	return s
}

func (s *StatsService) Start(ctx context.Context) {
	s.ctx = ctx
	// Initialize session baselines
	if historical, err := s.repo.Get(ctx); err == nil {
		s.startDownloaded = historical["total_downloaded"]
		s.startUploaded = historical["total_uploaded"]
	}

	// 1. Listen for progress events to track speeds
	progress := s.bus.SubscribeProgress()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("panic in stats progress listener", zap.Any("panic", r))
			}
			s.bus.UnsubscribeProgress(progress)
		}()
		for {
			select {
			case <-s.ctx.Done():
				return
			case ev, ok := <-progress:
				if !ok {
					return
				}
				s.handleProgressEvent(ev)
			}
		}
	}()

	// 2. Listen for lifecycle events
	lifecycle := s.bus.SubscribeLifecycle()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("panic in stats lifecycle listener", zap.Any("panic", r))
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
				s.handleLifecycleEvent(ev)
			}
		}
	}()

	// 3. Periodic stats flush (every 10 seconds)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("panic in stats flush loop", zap.Any("panic", r))
			}
		}()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				s.flushStats(context.Background()) // Final flush
				return
			case <-s.done:
				s.flushStats(context.Background()) // Final flush
				return
			case <-ticker.C:
				s.flushStats(s.ctx)
			}
		}
	}()

	// 4. Start broadcast loop
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("panic in stats broadcast loop", zap.Any("panic", r))
			}
		}()
		s.broadcastLoop()
	}()
}

func (s *StatsService) handleProgressEvent(ev event.ProgressEvent) {
	if ev.Type == "download" {
		s.downloadSpeeds.Update(ev.ID, ev.Speed)

		// Track bytes for batched persistence
		if ev.Downloaded > 0 {
			prev, _ := s.lastDownloaded.LoadOrStore(ev.ID, int64(0))
			prevVal := prev.(int64)
			if ev.Downloaded > prevVal {
				delta := ev.Downloaded - prevVal
				s.pendingDownloaded.Add(delta)
				s.lastDownloaded.Store(ev.ID, ev.Downloaded)
			}
		}
	} else if ev.Type == "upload" {
		s.uploadSpeeds.Update(ev.ID, ev.Speed)

		// Track bytes for batched persistence
		if ev.Uploaded > 0 {
			prev, _ := s.lastUploaded.LoadOrStore(ev.ID, int64(0))
			prevVal := prev.(int64)
			if ev.Uploaded > prevVal {
				delta := ev.Uploaded - prevVal
				s.pendingUploaded.Add(delta)
				s.lastUploaded.Store(ev.ID, ev.Uploaded)
			}
		}
	}
}

func (s *StatsService) handleLifecycleEvent(ev event.LifecycleEvent) {
	switch ev.Type {
	case event.DownloadCompleted:
		s.downloadSpeeds.OnComplete(ev.ID)
		s.lastDownloaded.Delete(ev.ID)
		s.repo.Increment(s.ctx, "downloads_completed", 1)
		s.triggerBroadcast()

	case event.DownloadError, event.DownloadPaused:
		s.downloadSpeeds.Remove(ev.ID)
		s.lastDownloaded.Delete(ev.ID)
		s.triggerBroadcast()

	case event.UploadCompleted:
		s.uploadSpeeds.OnComplete(ev.ID)
		s.lastUploaded.Delete(ev.ID)
		s.repo.Increment(s.ctx, "uploads_completed", 1)
		s.triggerBroadcast()

	case event.UploadError:
		s.uploadSpeeds.Remove(ev.ID)
		s.lastUploaded.Delete(ev.ID)
		s.triggerBroadcast()

	case event.DownloadCreated, event.DownloadStarted, event.DownloadResumed,
		event.UploadStarted:
		s.triggerBroadcast()
	}
}

func (s *StatsService) triggerBroadcast() {
	select {
	case s.trigger <- struct{}{}:
	default:
	}
}

func (s *StatsService) flushStats(ctx context.Context) {
	dl := s.pendingDownloaded.Swap(0)
	ul := s.pendingUploaded.Swap(0)

	if dl > 0 {
		s.repo.Increment(ctx, "total_downloaded", dl)
	}
	if ul > 0 {
		s.repo.Increment(ctx, "total_uploaded", ul)
	}
}

func (s *StatsService) Stop() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	s.mu.Lock()
	s.pollingCond.Broadcast()
	s.mu.Unlock()
}

func (s *StatsService) PausePolling() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.pollingPaused {
		s.pollingPaused = true
		s.logger.Debug("polling paused")
	}
}

func (s *StatsService) ResumePolling() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pollingPaused {
		s.pollingPaused = false
		s.pollingCond.Broadcast()
		s.logger.Debug("polling resumed")
	}
}

func (s *StatsService) broadcastLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		s.mu.Lock()
		for s.pollingPaused {
			select {
			case <-s.ctx.Done():
				s.mu.Unlock()
				return
			case <-s.done:
				s.mu.Unlock()
				return
			default:
				s.pollingCond.Wait()
			}
		}
		s.mu.Unlock()

		select {
		case <-s.ctx.Done():
			return
		case <-s.done:
			return
		case <-s.trigger:
			s.broadcast()
		case <-ticker.C:
			s.broadcast()
		}
	}
}

func (s *StatsService) broadcast() {
	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Second)
	defer cancel()

	stats, err := s.GetCurrent(ctx)
	if err == nil {
		// Convert to StatsEvent for typed publishing
		se := event.StatsEvent{}
		se.Speeds.Download = stats.Speeds.Download
		se.Speeds.Upload = stats.Speeds.Upload
		se.Tasks.Active = stats.Tasks.Active
		se.Tasks.Uploading = stats.Tasks.Uploading
		se.Tasks.Waiting = stats.Tasks.Waiting
		se.Tasks.Paused = stats.Tasks.Paused
		se.Tasks.Completed = stats.Tasks.Completed
		se.Tasks.Failed = stats.Tasks.Failed
		se.Usage.TotalDownloaded = stats.Usage.TotalDownloaded
		se.Usage.TotalUploaded = stats.Usage.TotalUploaded
		se.Usage.SessionDownloaded = stats.Usage.SessionDownloaded
		se.Usage.SessionUploaded = stats.Usage.SessionUploaded
		se.System.DiskFree = stats.System.DiskFree
		se.System.DiskTotal = stats.System.DiskTotal
		se.System.DiskUsage = stats.System.DiskUsage
		se.System.Uptime = stats.System.Uptime

		s.bus.PublishStats(se)
	}
}

func (s *StatsService) GetCurrent(ctx context.Context) (*model.Stats, error) {
	historical, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	// Add pending (unflushed) bytes to historical
	totalDownloaded := historical["total_downloaded"] + s.pendingDownloaded.Load()
	totalUploaded := historical["total_uploaded"] + s.pendingUploaded.Load()

	counts, _ := s.downloadRepo.GetStatusCounts(ctx)

	activeDownloads := counts[model.StatusActive]
	uploadingTasks := counts[model.StatusUploading]
	pendingDownloads := counts[model.StatusWaiting]
	pausedDownloads := counts[model.StatusPaused]
	currentCompleted := counts[model.StatusComplete]
	currentFailed := counts[model.StatusError]

	// Use smoothed speed calculations
	downloadSpeed := s.downloadSpeeds.GetGlobalSpeed()
	uploadSpeed := s.uploadSpeeds.GetGlobalSpeed()

	settings, _ := s.settingsRepo.Get(ctx)
	downloadDir := ""
	if settings != nil {
		downloadDir = settings.Download.DownloadDir
	}
	if downloadDir == "" {
		home, _ := os.UserHomeDir()
		downloadDir = filepath.Join(home, ".gravity", "downloads")
	}

	disk := s.getDiskStats(downloadDir)

	return &model.Stats{
		Speeds: model.Speeds{
			Download: downloadSpeed,
			Upload:   uploadSpeed,
		},
		Tasks: model.TaskCounts{
			Active:    activeDownloads,
			Uploading: uploadingTasks,
			Waiting:   pendingDownloads,
			Paused:    pausedDownloads,
			Completed: currentCompleted,
			Failed:    currentFailed,
		},
		Usage: model.UsageStats{
			TotalDownloaded:   totalDownloaded,
			TotalUploaded:     totalUploaded,
			SessionDownloaded: totalDownloaded - s.startDownloaded,
			SessionUploaded:   totalUploaded - s.startUploaded,
		},
		System: model.SystemStats{
			DiskFree:  disk.Free,
			DiskTotal: disk.Total,
			DiskUsage: disk.Usage,
			Uptime:    int64(time.Since(s.startTime).Seconds()),
		},
	}, nil
}

type diskInfo struct {
	Free  uint64
	Total uint64
	Usage float64
}

func (s *StatsService) getDiskStats(path string) diskInfo {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return diskInfo{}
	}

	total := fs.Blocks * uint64(fs.Bsize)
	free := fs.Bfree * uint64(fs.Bsize)
	used := total - free

	usage := 0.0
	if total > 0 {
		usage = (float64(used) / float64(total)) * 100
	}

	return diskInfo{
		Total: total,
		Free:  free,
		Usage: usage,
	}
}
