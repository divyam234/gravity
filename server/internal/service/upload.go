package service

import (
	"context"
	"log"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/store"
)

type UploadService struct {
	repo   *store.DownloadRepo
	engine engine.UploadEngine
	bus    *event.Bus
}

func NewUploadService(repo *store.DownloadRepo, eng engine.UploadEngine, bus *event.Bus) *UploadService {
	s := &UploadService{
		repo:   repo,
		engine: eng,
		bus:    bus,
	}

	// Wire up engine events
	eng.OnProgress(s.handleProgress)
	eng.OnComplete(s.handleComplete)
	eng.OnError(s.handleError)

	return s
}

func (s *UploadService) Start() {
	// Listen for download completions to trigger auto-upload
	events := s.bus.Subscribe()
	go func() {
		for ev := range events {
			if ev.Type == event.DownloadCompleted {
				d := ev.Data.(*model.Download)
				if d.Destination != "" {
					s.TriggerUpload(context.Background(), d)
				}
			}
		}
	}()
}

func (s *UploadService) TriggerUpload(ctx context.Context, d *model.Download) error {
	log.Printf("Upload: Triggering upload for %s to %s", d.ID, d.Destination)

	d.Status = model.StatusUploading
	d.UploadStatus = "running"
	s.repo.Update(ctx, d)

	// Trigger in engine
	// In a real scenario, we need the local file path.
	// Aria2 gives us the path in OnComplete.
	// For now, let's assume destination is correctFs:remotePath
	jobID, err := s.engine.Upload(ctx, d.Filename, d.Destination, engine.UploadOptions{})
	if err != nil {
		d.Status = model.StatusError
		d.Error = "Upload failed: " + err.Error()
		s.repo.Update(ctx, d)
		return err
	}

	d.UploadJobID = jobID
	s.repo.Update(ctx, d)

	s.bus.Publish(event.Event{
		Type:      event.UploadStarted,
		Timestamp: time.Now(),
		Data:      map[string]string{"id": d.ID, "destination": d.Destination},
	})

	return nil
}

func (s *UploadService) handleProgress(jobID string, p engine.UploadProgress) {
	ctx := context.Background()
	d, err := s.repo.GetByUploadJobID(ctx, jobID)
	if err != nil {
		return
	}

	s.bus.Publish(event.Event{
		Type:      event.UploadProgress,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"id":       d.ID,
			"uploaded": p.Uploaded,
			"size":     p.Size,
			"speed":    p.Speed,
		},
	})
}

func (s *UploadService) handleComplete(jobID string) {
	ctx := context.Background()
	d, err := s.repo.GetByUploadJobID(ctx, jobID)
	if err != nil {
		return
	}

	d.Status = model.StatusComplete
	d.UploadStatus = "complete"
	d.UploadProgress = 100
	s.repo.Update(ctx, d)

	s.bus.Publish(event.Event{
		Type:      event.UploadCompleted,
		Timestamp: time.Now(),
		Data:      d,
	})
}

func (s *UploadService) handleError(jobID string, err error) {
	ctx := context.Background()
	d, repoErr := s.repo.GetByUploadJobID(ctx, jobID)
	if repoErr != nil {
		return
	}

	d.Status = model.StatusError
	d.Error = "Upload error: " + err.Error()
	d.UploadStatus = "error"
	s.repo.Update(ctx, d)

	s.bus.Publish(event.Event{
		Type:      event.UploadError,
		Timestamp: time.Now(),
		Data:      map[string]string{"id": d.ID, "error": d.Error},
	})
}
