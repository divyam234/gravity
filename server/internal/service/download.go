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
	repo         *store.DownloadRepo
	engine       engine.DownloadEngine
	uploadEngine engine.UploadEngine
	bus          *event.Bus
	provider     *ProviderService
}

func NewDownloadService(repo *store.DownloadRepo, eng engine.DownloadEngine, ue engine.UploadEngine, bus *event.Bus, provider *ProviderService) *DownloadService {
	s := &DownloadService{
		repo:         repo,
		engine:       eng,
		uploadEngine: ue,
		bus:          bus,
		provider:     provider,
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
		Status:      model.StatusWaiting,
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
	d.Status = model.StatusActive
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

func (s *DownloadService) Get(ctx context.Context, id string) (*model.Download, error) {
	return s.repo.Get(ctx, id)
}

func (s *DownloadService) Pause(ctx context.Context, id string) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if d.IsMagnet && d.MagnetSource == "alldebrid" {
		files, _ := s.repo.GetFiles(ctx, d.ID)
		for _, f := range files {
			if f.EngineID != "" {
				s.engine.Pause(ctx, f.EngineID)
			}
		}
	} else if d.EngineID != "" {
		if err := s.engine.Pause(ctx, d.EngineID); err != nil {
			return err
		}
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

	if d.IsMagnet && d.MagnetSource == "alldebrid" {
		files, _ := s.repo.GetFiles(ctx, d.ID)
		for _, f := range files {
			if f.EngineID != "" {
				s.engine.Resume(ctx, f.EngineID)
			}
		}
	} else if d.EngineID != "" {
		if err := s.engine.Resume(ctx, d.EngineID); err != nil {
			return err
		}
	}

	d.Status = model.StatusActive
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

func (s *DownloadService) Retry(ctx context.Context, id string) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	// Only retry failed or completed tasks
	if d.Status != model.StatusError && d.Status != model.StatusComplete {
		return fmt.Errorf("cannot retry download in status %s", d.Status)
	}

	// Re-add to engine
	// Note: We might be missing headers if they were required and not stored.
	// Assuming public URL or auth in URL for now.
	engineID, err := s.engine.Add(ctx, d.ResolvedURL, engine.DownloadOptions{
		Filename: d.Filename,
		Dir:      d.Destination,
	})
	if err != nil {
		return err
	}

	d.EngineID = engineID
	d.Status = model.StatusActive
	d.Error = ""
	d.Downloaded = 0 // Reset progress?
	// d.Size = 0 // Keep size if known
	d.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, d); err != nil {
		return err
	}

	s.bus.Publish(event.Event{
		Type:      event.DownloadStarted, // Or similar
		Timestamp: time.Now(),
		Data:      d,
	})

	return nil
}

func (s *DownloadService) Delete(ctx context.Context, id string, deleteFiles bool) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	// 1. Stop in engine
	if d.IsMagnet && d.MagnetSource == "alldebrid" {
		// Cancel all individual files for AllDebrid
		files, _ := s.repo.GetFiles(ctx, d.ID)
		for _, f := range files {
			if f.EngineID != "" {
				s.engine.Cancel(ctx, f.EngineID)
				if deleteFiles {
					s.engine.Remove(ctx, f.EngineID)
				}
			}
		}
	} else if d.EngineID != "" {
		// Stop parent task for raw magnets or single files
		s.engine.Cancel(ctx, d.EngineID)
		if deleteFiles {
			s.engine.Remove(ctx, d.EngineID)
		}
	}

	// 2. Also stop upload if active
	if d.Status == model.StatusUploading && d.UploadJobID != "" {
		s.uploadEngine.Cancel(ctx, d.UploadJobID)
	}

	// 3. Clean up database
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Files will be deleted by CASCADE in DB
	return nil
}

func (s *DownloadService) List(ctx context.Context, status []string, limit, offset int) ([]*model.Download, int, error) {
	return s.repo.List(ctx, status, limit, offset)
}

func (s *DownloadService) GetFiles(ctx context.Context, id string) ([]model.DownloadFile, error) {
	return s.repo.GetFiles(ctx, id)
}

func (s *DownloadService) Sync(ctx context.Context) error {
	// 1. Get all downloads from engine
	engineTasks, err := s.engine.List(ctx)
	if err != nil {
		return err
	}

	// 2. Get all non-complete downloads from repo
	dbDownloads, _, err := s.repo.List(ctx, []string{
		string(model.StatusActive),
		string(model.StatusPaused),
		string(model.StatusWaiting),
	}, 1000, 0)
	if err != nil {
		return err
	}

	// 3. Map engine tasks by filename (simplistic mapping)
	engineMap := make(map[string]*engine.DownloadStatus)
	for _, t := range engineTasks {
		if t.Filename != "" {
			engineMap[t.Filename] = t
		}
	}

	// 4. Update EngineID for downloads that exist in engine
	for _, d := range dbDownloads {
		if t, ok := engineMap[d.Filename]; ok {
			d.EngineID = t.ID
			// Engine status is already mapped to canonical Gravity status
			d.Status = model.DownloadStatus(t.Status)
			s.repo.Update(ctx, d)
		} else {
			// If not in engine but was downloading, it might have failed or session lost
			// Only mark as error if it was actually downloading before
			if d.Status == model.StatusActive {
				d.Status = model.StatusError
				d.Error = "Engine task lost"
				s.repo.Update(ctx, d)
			}
		}
	}

	return nil
}

func (s *DownloadService) handleProgress(engineID string, p engine.Progress) {
	ctx := context.Background()

	// 1. Try to find if this is a single-file download
	d, err := s.repo.GetByEngineID(ctx, engineID)
	if err == nil {
		// Update DB with latest progress
		d.Downloaded = p.Downloaded
		d.Size = p.Size
		d.Speed = p.Speed
		d.ETA = p.ETA
		s.repo.Update(ctx, d)

		s.publishProgress(d)
		return
	}

	// 2. Try to find if this is an individual file within a magnet
	file, err := s.repo.GetFileByEngineID(ctx, engineID)
	if err == nil {
		file.Downloaded = p.Downloaded
		file.Progress = 0
		if file.Size > 0 {
			file.Progress = int((file.Downloaded * 100) / file.Size)
		}
		file.Status = model.StatusActive
		s.repo.UpdateFile(ctx, file)

		// Update parent download aggregate progress
		parent, err := s.repo.Get(ctx, file.DownloadID)
		if err == nil {
			s.updateAggregateProgress(ctx, parent)
		}
		return
	}
}

func (s *DownloadService) updateAggregateProgress(ctx context.Context, d *model.Download) {
	files, err := s.repo.GetFiles(ctx, d.ID)
	if err != nil {
		return
	}

	var totalDownloaded int64
	var totalSize int64
	filesComplete := 0
	activeFiles := 0

	for _, f := range files {
		totalDownloaded += f.Downloaded
		totalSize += f.Size
		if f.Status == model.StatusComplete {
			filesComplete++
		} else if f.Status == model.StatusActive {
			activeFiles++
			// We don't have per-file speed in the file record yet,
			// but we could get it from the engine if needed.
			// For now, we'll let handleProgress trigger aggregate updates.
		}
	}

	// If we don't have aggregate speed yet, we might need to store it.
	// For simplicity, let's just update the totals for now.
	d.Downloaded = totalDownloaded
	d.Size = totalSize
	d.FilesComplete = filesComplete

	// If all files complete, mark download as complete or uploading
	if filesComplete == len(files) && len(files) > 0 && d.Status != model.StatusComplete && d.Status != model.StatusUploading {
		if d.Destination != "" {
			d.Status = model.StatusUploading
		} else {
			d.Status = model.StatusComplete
		}
		now := time.Now()
		d.CompletedAt = &now
	}

	s.repo.Update(ctx, d)
	s.publishProgress(d)
}

func (s *DownloadService) publishProgress(d *model.Download) {
	s.bus.Publish(event.Event{
		Type:      event.DownloadProgress,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"id":         d.ID,
			"downloaded": d.Downloaded,
			"size":       d.Size,
			"speed":      d.Speed,
			"eta":        d.ETA,
		},
	})
}

func (s *DownloadService) handleComplete(engineID string, filePath string) {
	ctx := context.Background()

	// 1. Try single-file
	d, err := s.repo.GetByEngineID(ctx, engineID)
	if err == nil {
		if d.Destination != "" {
			d.Status = model.StatusUploading
		} else {
			d.Status = model.StatusComplete
		}

		d.LocalPath = filePath
		now := time.Now()
		d.CompletedAt = &now
		s.repo.Update(ctx, d)

		s.bus.Publish(event.Event{
			Type:      event.DownloadCompleted,
			Timestamp: time.Now(),
			Data:      d,
		})
		return
	}

	// 2. Try file within magnet
	file, err := s.repo.GetFileByEngineID(ctx, engineID)
	if err == nil {
		file.Status = model.StatusComplete
		file.Downloaded = file.Size
		file.Progress = 100
		s.repo.UpdateFile(ctx, file)

		parent, err := s.repo.Get(ctx, file.DownloadID)
		if err == nil {
			s.updateAggregateProgress(ctx, parent)
		}
		return
	}
}

func (s *DownloadService) handleError(engineID string, err error) {
	ctx := context.Background()

	// 1. Try single-file
	d, repoErr := s.repo.GetByEngineID(ctx, engineID)
	if repoErr == nil {
		d.Status = model.StatusError
		d.Error = err.Error()
		s.repo.Update(ctx, d)

		s.bus.Publish(event.Event{
			Type:      event.DownloadError,
			Timestamp: time.Now(),
			Data:      map[string]string{"id": d.ID, "error": d.Error},
		})
		return
	}

	// 2. Try file within magnet
	file, repoErr := s.repo.GetFileByEngineID(ctx, engineID)
	if repoErr == nil {
		file.Status = model.StatusError
		file.Error = err.Error()
		s.repo.UpdateFile(ctx, file)

		parent, err := s.repo.Get(ctx, file.DownloadID)
		if err == nil {
			// Even if a file fails, we might want to continue others
			// For now, just update aggregate
			s.updateAggregateProgress(ctx, parent)
		}
		return
	}
}
