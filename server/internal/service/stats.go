package service

import (
	"context"
	"log"
	"sync"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/store"
)

type StatsService struct {
	repo           *store.StatsRepo
	downloadRepo   *store.DownloadRepo
	downloadEngine engine.DownloadEngine
	uploadEngine   engine.UploadEngine
	bus            *event.Bus

	mu            sync.Mutex
	pollingPaused bool
	pollingCond   *sync.Cond
	done          chan struct{}
	trigger       chan struct{}
}

func NewStatsService(repo *store.StatsRepo, dr *store.DownloadRepo, de engine.DownloadEngine, ue engine.UploadEngine, bus *event.Bus) *StatsService {
	s := &StatsService{
		repo:           repo,
		downloadRepo:   dr,
		downloadEngine: de,
		uploadEngine:   ue,
		bus:            bus,
		pollingPaused:  true,
		done:           make(chan struct{}),
		trigger:        make(chan struct{}, 1),
	}
	s.pollingCond = sync.NewCond(&s.mu)
	return s
}

func (s *StatsService) Start() {
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

	downloads, _ := s.downloadEngine.List(ctx)

	activeDownloads := 0
	downloadSpeed := int64(0)
	for _, d := range downloads {
		status := model.DownloadStatus(d.Status)
		if status == model.StatusActive {
			activeDownloads++
			downloadSpeed += d.Speed
		}
	}

	// Fetch paused/waiting from DB as they might not be in engine
	pendingDownloads, _ := s.downloadRepo.Count(ctx, []string{string(model.StatusWaiting)})
	pausedDownloads, _ := s.downloadRepo.Count(ctx, []string{string(model.StatusPaused)})

	activeUploads := 0
	uploadSpeed := int64(0)
	uploadStats, _ := s.uploadEngine.GetGlobalStats(ctx)
	if uploadStats != nil {
		activeUploads = uploadStats.ActiveTransfers
		if activeUploads > 0 {
			uploadSpeed = uploadStats.Speed
		}
	}

	// Fetch current counts from DB to reflect deletions
	currentCompleted, _ := s.downloadRepo.Count(ctx, []string{string(model.StatusComplete)})
	currentFailed, _ := s.downloadRepo.Count(ctx, []string{string(model.StatusError)})

	// Override activeUploads with DB count for accurate queue representation
	uploadingCount, _ := s.downloadRepo.Count(ctx, []string{string(model.StatusUploading)})
	if uploadingCount > 0 {
		activeUploads = uploadingCount
	}

	return &model.Stats{

		Active: model.ActiveStats{

			Downloads: activeDownloads,

			DownloadSpeed: downloadSpeed,

			Uploads: activeUploads,

			UploadSpeed: uploadSpeed,
		},

		Queue: model.QueueStats{

			Pending: pendingDownloads,

			Paused: pausedDownloads,
		},

		Totals: model.TotalStats{

			TotalDownloaded: historical["total_downloaded"],

			TotalUploaded: historical["total_uploaded"],

			TasksFinished: int64(currentCompleted),

			TasksFailed: int64(currentFailed),
		},
	}, nil

}
