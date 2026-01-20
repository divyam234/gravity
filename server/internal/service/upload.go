package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/store"
)

type UploadService struct {
	repo         *store.DownloadRepo
	settingsRepo *store.SettingsRepo
	engine       engine.UploadEngine
	bus          *event.Bus
}

func NewUploadService(repo *store.DownloadRepo, settingsRepo *store.SettingsRepo, eng engine.UploadEngine, bus *event.Bus) *UploadService {
	s := &UploadService{
		repo:         repo,
		settingsRepo: settingsRepo,
		engine:       eng,
		bus:          bus,
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

	// Generate job ID upfront so we can save it before starting the upload
	// This avoids race condition with handleProgress overwriting it
	jobID := time.Now().UnixNano()
	jobIDStr := fmt.Sprintf("%d", jobID)

	d.Status = model.StatusUploading
	d.UploadStatus = "running"
	d.UploadJobID = jobIDStr // Save job ID BEFORE starting upload
	s.repo.Update(ctx, d)

	// Trigger in engine
	// Use LocalPath (absolute) if available, otherwise Filename (relative/fallback)
	srcPath := d.LocalPath
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	if err == nil {
		if val, ok := settings["delete_after_upload"]; ok && val == "false" {
			shouldDelete = false
		}
	}

	if shouldDelete && d.LocalPath != "" {
		log.Printf("Upload: Deleting local copy %s", d.LocalPath)
		if err := os.RemoveAll(d.LocalPath); err != nil {
			log.Printf("Upload: Failed to delete local file: %v", err)
		}
		// Also try removing .aria2 control file just in case
		os.Remove(d.LocalPath + ".aria2")
	}
}

func (s *UploadService) handleError(downloadID string, err error) {
	ctx := context.Background()
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
