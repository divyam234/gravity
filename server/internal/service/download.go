package service

import (
	"context"
	"fmt"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/store"

	"github.com/google/uuid"
)

type DownloadService struct {
	repo     *store.DownloadRepo
	engine   engine.DownloadEngine
	bus      *event.Bus
	provider *ProviderService
}

func NewDownloadService(repo *store.DownloadRepo, eng engine.DownloadEngine, bus *event.Bus, provider *ProviderService) *DownloadService {
	s := &DownloadService{
		repo:     repo,
		engine:   eng,
		bus:      bus,
		provider: provider,
	}

	// Wire up engine events
	eng.OnProgress(s.handleProgress)
	eng.OnComplete(s.handleComplete)
	eng.OnError(s.handleError)

	return s
}

func (s *DownloadService) Create(ctx context.Context, url string, filename string, destination string) (*model.Download, error) {
	// 1. Resolve URL through providers
	res, providerName, err := s.provider.Resolve(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve URL: %w", err)
	}

	if filename == "" {
		filename = res.Filename
	}

	d := &model.Download{
		ID:          "d_" + uuid.New().String()[:8],
		URL:         url,
		ResolvedURL: res.URL,
		Provider:    providerName,
		Filename:    filename,
		Size:        res.Size,
		Destination: destination,
		Status:      model.StatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}

	// 2. Start download in engine with resolved URL and headers
	engineID, err := s.engine.Add(ctx, res.URL, engine.DownloadOptions{
		Filename: filename,
		Headers:  res.Headers,
	})
	if err != nil {
		d.Status = model.StatusError
		d.Error = err.Error()
		s.repo.Update(ctx, d)
		return nil, err
	}

	d.EngineID = engineID
	d.Status = model.StatusDownloading
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}

	s.bus.Publish(event.Event{
		Type:      event.DownloadCreated,
		Timestamp: time.Now(),
		Data:      d,
	})

	return d, nil
}

func (s *DownloadService) Pause(ctx context.Context, id string) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := s.engine.Pause(ctx, d.EngineID); err != nil {
		return err
	}

	d.Status = model.StatusPaused
	if err := s.repo.Update(ctx, d); err != nil {
		return err
	}

	s.bus.Publish(event.Event{
		Type:      event.DownloadPaused,
		Timestamp: time.Now(),
		Data:      map[string]string{"id": id},
	})

	return nil
}

func (s *DownloadService) Resume(ctx context.Context, id string) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := s.engine.Resume(ctx, d.EngineID); err != nil {
		return err
	}

	d.Status = model.StatusDownloading
	if err := s.repo.Update(ctx, d); err != nil {
		return err
	}

	s.bus.Publish(event.Event{
		Type:      event.DownloadResumed,
		Timestamp: time.Now(),
		Data:      map[string]string{"id": id},
	})

	return nil
}

func (s *DownloadService) Delete(ctx context.Context, id string, deleteFiles bool) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	// Stop in engine
	s.engine.Cancel(ctx, d.EngineID)
	if deleteFiles {
		s.engine.Remove(ctx, d.EngineID)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *DownloadService) List(ctx context.Context, status []string, limit, offset int) ([]*model.Download, int, error) {
	return s.repo.List(ctx, status, limit, offset)
}

func (s *DownloadService) handleProgress(engineID string, p engine.Progress) {
	ctx := context.Background()
	d, err := s.repo.GetByEngineID(ctx, engineID)
	if err != nil {
		return
	}

	s.bus.Publish(event.Event{
		Type:      event.DownloadProgress,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"id":         d.ID,
			"downloaded": p.Downloaded,
			"size":       p.Size,
			"speed":      p.Speed,
			"eta":        p.ETA,
		},
	})
}

func (s *DownloadService) handleComplete(engineID string, filePath string) {
	ctx := context.Background()
	d, err := s.repo.GetByEngineID(ctx, engineID)
	if err != nil {
		return
	}

	d.Status = model.StatusComplete
	now := time.Now()
	d.CompletedAt = &now
	s.repo.Update(ctx, d)

	s.bus.Publish(event.Event{
		Type:      event.DownloadCompleted,
		Timestamp: time.Now(),
		Data:      d,
	})
}

func (s *DownloadService) handleError(engineID string, err error) {
	ctx := context.Background()
	d, repoErr := s.repo.GetByEngineID(ctx, engineID)
	if repoErr != nil {
		return
	}

	d.Status = model.StatusError
	d.Error = err.Error()
	s.repo.Update(ctx, d)

	s.bus.Publish(event.Event{
		Type:      event.DownloadError,
		Timestamp: time.Now(),
		Data:      map[string]string{"id": d.ID, "error": d.Error},
	})
}
