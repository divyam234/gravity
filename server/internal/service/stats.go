package service

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/store"

	"go.uber.org/zap"
)

type SpeedEntry struct {
	Speed     int64
	UpdatedAt time.Time
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

	// Task-level speed tracking for global aggregation (using sync.Map for lock-free reads)
	taskSpeeds   sync.Map // map[string]SpeedEntry
	uploadSpeeds sync.Map // map[string]SpeedEntry

	// Track accumulated bytes for active downloads to persist incrementally
	activeBytes   map[string]int64
	activeUploads map[string]int64

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
		activeBytes:    make(map[string]int64),
		activeUploads:  make(map[string]int64),
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

	// 1. Listen for events to aggregate speeds and update totals
	ch := s.bus.Subscribe()
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case ev := <-ch:
				switch ev.Type {
				case event.DownloadProgress:
					if data, ok := ev.Data.(map[string]interface{}); ok {
						id := data["id"].(string)
						speed := data["speed"].(int64)
						downloaded := data["downloaded"].(int64)

						s.taskSpeeds.Store(id, SpeedEntry{Speed: speed, UpdatedAt: time.Now()})

						s.mu.Lock()
						prev := s.activeBytes[id]
						if downloaded > prev {
							delta := downloaded - prev
							s.repo.Increment(s.ctx, "total_downloaded", delta)
							s.activeBytes[id] = downloaded
						}
						s.mu.Unlock()
					}
				case event.UploadProgress:
					if data, ok := ev.Data.(map[string]interface{}); ok {
						id := data["id"].(string)
						speed := data["speed"].(int64)
						uploaded := data["uploaded"].(int64)

						s.uploadSpeeds.Store(id, SpeedEntry{Speed: speed, UpdatedAt: time.Now()})

						s.mu.Lock()
						prev := s.activeUploads[id]
						if uploaded > prev {
							delta := uploaded - prev
							s.repo.Increment(s.ctx, "total_uploaded", delta)
							s.activeUploads[id] = uploaded
						}
						s.mu.Unlock()
					}
				case event.DownloadCompleted, event.DownloadError, event.DownloadPaused:
					if d, ok := ev.Data.(*model.Download); ok {
						s.mu.Lock()
						delete(s.activeBytes, d.ID)
						s.mu.Unlock()
					}
					if ev.Type == event.DownloadCompleted {
						s.repo.Increment(s.ctx, "downloads_completed", 1)
					}
				case event.UploadCompleted, event.UploadError:
					if d, ok := ev.Data.(*model.Download); ok {
						s.mu.Lock()
						delete(s.activeUploads, d.ID)
						s.mu.Unlock()
					}
					if ev.Type == event.UploadCompleted {
						s.repo.Increment(s.ctx, "uploads_completed", 1)
					}
				}

				// Trigger broadcast for relevant state changes
				switch ev.Type {
				case event.DownloadCreated, event.DownloadStarted, event.DownloadPaused,
					event.DownloadResumed, event.DownloadCompleted, event.DownloadError,
					event.UploadStarted, event.UploadCompleted, event.UploadError:

					select {
					case s.trigger <- struct{}{}:
					default:
					}
				}
			}
		}
	}()

	// 2. Start broadcast loop
	go s.broadcastLoop()
}

func (s *StatsService) Stop() {
	close(s.done)
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
		s.bus.Publish(event.Event{
			Type:      event.StatsUpdate,
			Timestamp: time.Now(),
			Data:      stats,
		})
	}
}

func (s *StatsService) GetCurrent(ctx context.Context) (*model.Stats, error) {
	historical, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	counts, _ := s.downloadRepo.GetStatusCounts(ctx)

	activeDownloads := counts[model.StatusActive]
	uploadingTasks := counts[model.StatusUploading]
	pendingDownloads := counts[model.StatusWaiting]
	pausedDownloads := counts[model.StatusPaused]
	currentCompleted := counts[model.StatusComplete]
	currentFailed := counts[model.StatusError]

	var downloadSpeed, uploadSpeed int64
	now := time.Now()

	s.taskSpeeds.Range(func(key, value any) bool {
		entry := value.(SpeedEntry)
		if now.Sub(entry.UpdatedAt) < 5*time.Second {
			downloadSpeed += entry.Speed
		} else {
			s.taskSpeeds.Delete(key)
		}
		return true
	})
	s.uploadSpeeds.Range(func(key, value any) bool {
		entry := value.(SpeedEntry)
		if now.Sub(entry.UpdatedAt) < 5*time.Second {
			uploadSpeed += entry.Speed
		} else {
			s.uploadSpeeds.Delete(key)
		}
		return true
	})

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
			TotalDownloaded:   historical["total_downloaded"],
			TotalUploaded:     historical["total_uploaded"],
			SessionDownloaded: historical["total_downloaded"] - s.startDownloaded,
			SessionUploaded:   historical["total_uploaded"] - s.startUploaded,
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