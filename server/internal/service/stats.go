package service

import (
	"context"
	"gravity/internal/engine"
	"gravity/internal/store"
)

type StatsService struct {
	repo           *store.StatsRepo
	downloadEngine engine.DownloadEngine
	uploadEngine   engine.UploadEngine
}

func NewStatsService(repo *store.StatsRepo, de engine.DownloadEngine, ue engine.UploadEngine) *StatsService {
	return &StatsService{
		repo:           repo,
		downloadEngine: de,
		uploadEngine:   ue,
	}
}

func (s *StatsService) GetCurrent(ctx context.Context) (map[string]interface{}, error) {
	historical, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	downloads, _ := s.downloadEngine.List(ctx)

	activeDownloads := 0
	downloadSpeed := int64(0)
	for _, d := range downloads {
		if d.Status == "active" || d.Status == "downloading" {
			activeDownloads++
			downloadSpeed += d.Speed
		}
	}

	return map[string]interface{}{
		"active": map[string]interface{}{
			"downloads":     activeDownloads,
			"downloadSpeed": downloadSpeed,
		},
		"totals": historical,
	}, nil
}
