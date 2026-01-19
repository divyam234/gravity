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
	downloadRepo   *store.DownloadRepo
	downloadEngine engine.DownloadEngine
	uploadEngine   engine.UploadEngine
	bus            *event.Bus
}

func NewStatsService(repo *store.StatsRepo, dr *store.DownloadRepo, de engine.DownloadEngine, ue engine.UploadEngine, bus *event.Bus) *StatsService {
	return &StatsService{
		repo:           repo,
		downloadRepo:   dr,
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

func (s *StatsService) GetCurrent(ctx context.Context) (*model.Stats, error) {
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
		status := model.DownloadStatus(d.Status)
		if status == model.StatusActive {
			activeDownloads++
			downloadSpeed += d.Speed
		} else if status == model.StatusPaused {
			pausedDownloads++
		} else if status == model.StatusWaiting {
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

	

		// Fetch current counts from DB to reflect deletions

		_, currentCompleted, _ := s.downloadRepo.List(ctx, []string{string(model.StatusComplete)}, 0, 0)

		_, currentFailed, _ := s.downloadRepo.List(ctx, []string{string(model.StatusError)}, 0, 0)

	

						return &model.Stats{

	

							Active: model.ActiveStats{

	

								Downloads:     activeDownloads,

	

								DownloadSpeed: downloadSpeed,

	

								Uploads:       activeUploads,

	

								UploadSpeed:   uploadSpeed,

	

							},

	

							Queue: model.QueueStats{

	

								Pending: pendingDownloads,

	

								Paused:  pausedDownloads,

	

							},

	

							Totals: model.TotalStats{

	

								TotalDownloaded:    historical["total_downloaded"],

	

								TotalUploaded:      historical["total_uploaded"],

	

								TasksFinished:      int64(currentCompleted),

	

								TasksFailed:        int64(currentFailed),

	

							},

	

						}, nil

	

					}

	

					

	

				

	

			

	

		

	