package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

	// 2. Listen for download completions to trigger auto-upload via typed channel
	lifecycleEvents := s.bus.SubscribeLifecycle()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("panic in upload lifecycle listener", zap.Any("panic", r))
			}
			s.bus.UnsubscribeLifecycle(lifecycleEvents)
		}()
		for {
			select {
			case <-s.ctx.Done():
				return
			case ev := <-lifecycleEvents:
				if ev.Type != event.DownloadCompleted {
					continue
				}

				// Safe type assertion - skip if data is not a Download
				d, ok := ev.Data.(*model.Download)
				if !ok {
					s.logger.Warn("unexpected data type in lifecycle event",
						zap.String("type", string(ev.Type)))
					continue
				}

				settings, err := s.settingsRepo.Get(s.ctx)
				if err != nil || settings == nil {
					continue
				}

				if d.Destination == "" && settings.Upload.AutoUpload {
					d.Destination = settings.Upload.DefaultRemote
				}

				if d.Destination != "" {
					s.TriggerUpload(s.ctx, d)
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

	s.bus.PublishLifecycle(event.LifecycleEvent{
		Type:      event.UploadStarted,
		ID:        d.ID,
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

	s.bus.PublishProgress(event.ProgressEvent{
		ID:       d.ID,
		Type:     "upload",
		Uploaded: p.Uploaded,
		Size:     p.Size,
		Speed:    p.Speed,
	})
}

func (s *UploadService) handleComplete(downloadID string) {
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	d, err := s.repo.Get(ctx, downloadID)
	if err != nil {
		return
	}

	d.Status = model.StatusComplete
	d.UploadStatus = "complete"
	d.UploadProgress = 100
	s.repo.Update(ctx, d)

	s.bus.PublishLifecycle(event.LifecycleEvent{
		Type:      event.UploadCompleted,
		ID:        d.ID,
		Timestamp: time.Now(),
		Data:      d,
	})

	settings, err := s.settingsRepo.Get(ctx)
	shouldDelete := true
	if err == nil && settings != nil {
		shouldDelete = settings.Upload.RemoveLocal
	}
	if d.RemoveLocal != nil {
		shouldDelete = *d.RemoveLocal
	}

	if shouldDelete && d.DownloadDir != "" {
		filePath := filepath.Join(d.DownloadDir, d.Filename)
		s.logger.Debug("deleting local copy after upload", zap.String("path", filePath))
		if err := os.RemoveAll(filePath); err != nil {
			s.logger.Error("failed to delete local file", zap.String("path", filePath), zap.Error(err))
		}
		os.Remove(filePath + ".aria2")
		os.Remove(filePath + ".rclone")
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

	s.bus.PublishLifecycle(event.LifecycleEvent{
		Type:      event.UploadError,
		ID:        d.ID,
		Timestamp: time.Now(),
		Error:     d.Error,
		Data:      map[string]string{"id": d.ID, "error": d.Error},
	})
}
