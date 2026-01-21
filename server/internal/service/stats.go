package service

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/store"
)

type StatsService struct {
	repo           *store.StatsRepo
	settingsRepo   *store.SettingsRepo
	downloadRepo   *store.DownloadRepo
	downloadEngine engine.DownloadEngine
	uploadEngine   engine.UploadEngine
	bus            *event.Bus

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

func NewStatsService(repo *store.StatsRepo, setr *store.SettingsRepo, dr *store.DownloadRepo, de engine.DownloadEngine, ue engine.UploadEngine, bus *event.Bus) *StatsService {
	s := &StatsService{
		repo:           repo,
		settingsRepo:   setr,
		downloadRepo:   dr,
		downloadEngine: de,
		uploadEngine:   ue,
		bus:            bus,
		pollingPaused:  true,
		startTime:      time.Now(),
		done:           make(chan struct{}),
		trigger:        make(chan struct{}, 1),
	}
	s.pollingCond = sync.NewCond(&s.mu)
	return s
}

func (s *StatsService) Start() {
	// Initialize session baselines
	ctx := context.Background()
	if historical, err := s.repo.Get(ctx); err == nil {
		s.startDownloaded = historical["total_downloaded"]
		s.startUploaded = historical["total_uploaded"]
	}

	// 1. Listen for completion events to update totals
	ch := s.bus.Subscribe()
	go func() {
		for ev := range ch {
			ctx := context.Background()
			switch ev.Type {
			case event.DownloadCompleted:
				if d, ok := ev.Data.(*model.Download); ok {
					s.repo.Increment(ctx, "downloads_completed", 1)
					s.repo.Increment(ctx, "total_downloaded", d.Size)
				}
			case event.UploadCompleted:
				if d, ok := ev.Data.(*model.Download); ok {
					s.repo.Increment(ctx, "uploads_completed", 1)
					s.repo.Increment(ctx, "total_uploaded", d.Size)
				}
			case event.DownloadError:
				s.repo.Increment(ctx, "downloads_failed", 1)
			case event.UploadError:
				s.repo.Increment(ctx, "uploads_failed", 1)
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
		log.Println("Stats: Polling paused")
	}
}

func (s *StatsService) ResumePolling() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pollingPaused {
		s.pollingPaused = false
		s.pollingCond.Broadcast()
		log.Println("Stats: Polling resumed")
	}
}

func (s *StatsService) broadcastLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		s.mu.Lock()
		for s.pollingPaused {
			select {
			case <-s.done:
				s.mu.Unlock()
				return
			default:
				s.pollingCond.Wait()
			}
		}
		s.mu.Unlock()

		select {
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
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

	// Fetch all counts from SQLite in one go
	counts, _ := s.downloadRepo.GetStatusCounts(ctx)

	activeDownloads := counts[model.StatusActive]
	uploadingTasks := counts[model.StatusUploading]
	pendingDownloads := counts[model.StatusWaiting]
	pausedDownloads := counts[model.StatusPaused]
	currentCompleted := counts[model.StatusComplete]
	currentFailed := counts[model.StatusError]

	// Fetch real-time speeds from engines
	downloadSpeed := int64(0)
	if activeDownloads > 0 {
		downloads, _ := s.downloadEngine.List(ctx)
		for _, d := range downloads {
			if model.DownloadStatus(d.Status) == model.StatusActive {
				downloadSpeed += d.Speed
			}
		}
	}

	uploadSpeed := int64(0)
	if uploadingTasks > 0 {
		uploadStats, _ := s.uploadEngine.GetGlobalStats(ctx)
		if uploadStats != nil {
			uploadSpeed = uploadStats.Speed
		}
	}

	// Disk stats
	settings, _ := s.settingsRepo.Get(ctx)
	downloadDir := settings["download_dir"]
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
