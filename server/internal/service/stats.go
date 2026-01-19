package service

import (
	"context"
	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/store"
)

type StatsService struct {
	repo           *store.StatsRepo
	downloadEngine engine.DownloadEngine
	uploadEngine   engine.UploadEngine
	bus            *event.Bus
}

func NewStatsService(repo *store.StatsRepo, de engine.DownloadEngine, ue engine.UploadEngine, bus *event.Bus) *StatsService {
	return &StatsService{
		repo:           repo,
		downloadEngine: de,
		uploadEngine:   ue,
		bus:            bus,
	}
}

func (s *StatsService) Start() {
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
		}
	}()
}

func (s *StatsService) GetCurrent(ctx context.Context) (map[string]interface{}, error) {
	historical, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	downloads, _ := s.downloadEngine.List(ctx)

	activeDownloads := 0
	pendingDownloads := 0
	pausedDownloads := 0
	downloadSpeed := int64(0)
	for _, d := range downloads {
		if d.Status == "active" || d.Status == "downloading" {
			activeDownloads++
			downloadSpeed += d.Speed
		} else if d.Status == "paused" {
			pausedDownloads++
		} else if d.Status == "waiting" || d.Status == "pending" {
			pendingDownloads++
		}
	}

	uploadStats, _ := s.uploadEngine.GetGlobalStats(ctx)
	activeUploads := 0
	uploadSpeed := int64(0)
	if uploadStats != nil {
		activeUploads = uploadStats.ActiveTransfers
		uploadSpeed = uploadStats.Speed
	}

	return map[string]interface{}{
		"active": map[string]interface{}{
			"downloads":     activeDownloads,
			"downloadSpeed": downloadSpeed,
			"uploads":       activeUploads,
			"uploadSpeed":   uploadSpeed,
		},
		"queue": map[string]interface{}{
			"pending": pendingDownloads,
			"paused":  pausedDownloads,
		},
		"totals": historical,
	}, nil
}
