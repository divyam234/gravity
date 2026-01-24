package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/store"

	"go.uber.org/zap"
)

type UploadService struct {
	repo         *store.DownloadRepo
	settingsRepo *store.SettingsRepo
	engine       engine.UploadEngine
	bus          *event.Bus
	ctx          context.Context
	logger       *zap.Logger
}

func NewUploadService(repo *store.DownloadRepo, settingsRepo *store.SettingsRepo, eng engine.UploadEngine, bus *event.Bus, l *zap.Logger) *UploadService {
	s := &UploadService{
		repo:         repo,
		settingsRepo: settingsRepo,
		engine:       eng,
		bus:          bus,
		logger:       l.With(zap.String("service", "upload")),
	}

	// Wire up engine events
	eng.OnProgress(s.handleProgress)
	eng.OnComplete(s.handleComplete)
	eng.OnError(s.handleError)

	return s
}

func (s *UploadService) Start(ctx context.Context) {
	s.ctx = ctx

	// 1. Initial sync to catch stale uploads
	if err := s.Sync(ctx); err != nil {
		s.logger.Error("failed to sync uploads", zap.Error(err))
	}

	// 2. Listen for download completions to trigger auto-upload
	events := s.bus.Subscribe()
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case ev := <-events:
				if ev.Type == event.DownloadCompleted {
					d := ev.Data.(*model.Download)
					
					settings, err := s.settingsRepo.Get(s.ctx)
					if err != nil || settings == nil {
						continue
					}

					// Logic:
					// 1. If task has explicit destination -> use it.
					// 2. If task has NO destination AND AutoUpload is ON -> use DefaultRemote.
					target := d.Destination
					if target == "" && settings.Upload.AutoUpload {
						target = settings.Upload.DefaultRemote
					}

					if target != "" {
						// Update the download object with the resolved destination so engine knows where to put it
						d.Destination = target
						s.TriggerUpload(s.ctx, d)
					}
				}
			}
		}
	}()
}

func (s *UploadService) Sync(ctx context.Context) error {
	// Find all downloads stuck in "uploading" state
	uploads, _, err := s.repo.List(ctx, []string{string(model.StatusUploading)}, 1000, 0, false)
	if err != nil {
		return err
	}

	settings, _ := s.settingsRepo.Get(ctx)
	autoUpload := true
	if settings != nil {
		autoUpload = settings.Upload.AutoUpload
	}

	for _, d := range uploads {
		s.logger.Info("resetting stale upload", zap.String("id", d.ID))
		d.Status = model.StatusComplete
		d.UploadStatus = ""
		d.UploadJobID = ""
		
		target := d.Destination
		if target == "" && autoUpload {
			target = settings.Upload.DefaultRemote
		}

		s.repo.Update(ctx, d)

		if target != "" {
			s.logger.Debug("re-triggering stale upload", zap.String("id", d.ID), zap.String("dest", target))
			d.Destination = target
			go s.TriggerUpload(s.ctx, d)
		} else {
			s.logger.Debug("skipping stale upload resume: no destination and auto-upload disabled", zap.String("id", d.ID))
		}
	}

	return nil
}

func (s *UploadService) TriggerUpload(ctx context.Context, d *model.Download) error {
	s.logger.Info("triggering upload", 
		zap.String("id", d.ID), 
		zap.String("destination", d.Destination))

	// Generate job ID upfront so we can save it before starting the upload
	// This avoids race condition with handleProgress overwriting it
	jobID := time.Now().UnixNano()
	jobIDStr := fmt.Sprintf("%d", jobID)

	d.Status = model.StatusUploading
	d.UploadStatus = "running"
	d.UploadJobID = jobIDStr // Save job ID BEFORE starting upload
	s.repo.Update(ctx, d)

	// Trigger in engine
	// Use DownloadDir (absolute) if available, otherwise Filename (relative/fallback)
	srcPath := d.DownloadDir
	if srcPath == "" {
		srcPath = d.Filename
	}
	_, err := s.engine.Upload(ctx, srcPath, d.Destination, engine.UploadOptions{
		TrackingID: d.ID,  // Pass download ID for progress tracking callbacks
		JobID:      jobID, // Pass the pre-generated job ID
	})
	if err != nil {
		d.Status = model.StatusError
		d.Error = "Upload failed: " + err.Error()
		d.UploadJobID = "" // Clear job ID on failure
		s.repo.Update(ctx, d)
		return err
	}

	s.bus.Publish(event.Event{
		Type:      event.UploadStarted,
		Timestamp: time.Now(),
		Data:      map[string]string{"id": d.ID, "destination": d.Destination},
	})

	return nil
}

func (s *UploadService) handleProgress(downloadID string, p engine.UploadProgress) {
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	d, err := s.repo.Get(ctx, downloadID)
	if err != nil {
		// Download may have been deleted - this is expected, silently ignore
		return
	}

	// Update DB with latest progress
	if p.Size > 0 {
		d.UploadProgress = int((p.Uploaded * 100) / p.Size)
	}
	d.UploadSpeed = p.Speed
	s.repo.Update(ctx, d)

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

func (s *UploadService) handleComplete(downloadID string) {
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	d, err := s.repo.Get(ctx, downloadID)
	if err != nil {
		// Download may have been deleted - this is expected, silently ignore
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

	// Check if we should delete the local file
	settings, err := s.settingsRepo.Get(ctx)
	shouldDelete := true // Default to true
	if err == nil && settings != nil {
		shouldDelete = settings.Upload.RemoveLocal
	}

	if shouldDelete && d.DownloadDir != "" {
		s.logger.Debug("deleting local copy after upload", zap.String("path", d.DownloadDir))
		if err := os.RemoveAll(d.DownloadDir); err != nil {
			s.logger.Error("failed to delete local file", zap.String("path", d.DownloadDir), zap.Error(err))
		}
		// Also try removing .aria2 control file just in case
		os.Remove(d.DownloadDir + ".aria2")
	}
}

func (s *UploadService) handleError(downloadID string, err error) {
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	d, repoErr := s.repo.Get(ctx, downloadID)
	if repoErr != nil {
		// Download may have been deleted - this is expected, silently ignore
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
