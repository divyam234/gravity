package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

	// Throttling for database writes
	lastPersistMap map[string]time.Time
}

func NewDownloadService(repo *store.DownloadRepo, eng engine.DownloadEngine, ue engine.UploadEngine, bus *event.Bus, provider *ProviderService) *DownloadService {
	s := &DownloadService{
		repo:           repo,
		engine:         eng,
		uploadEngine:   ue,
		bus:            bus,
		provider:       provider,
		lastPersistMap: make(map[string]time.Time),
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
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Attach files for multi-file downloads
	files, err := s.repo.GetFiles(ctx, id)
	if err == nil && len(files) > 0 {
		d.Files = files
	}

	// Attach peers if it's an active aria2 download
	if d.Status == model.StatusActive && d.EngineID != "" && (d.MagnetSource == "aria2" || !d.IsMagnet) {
		peers, err := s.engine.GetPeers(ctx, d.EngineID)
		if err == nil {
			d.PeerDetails = make([]model.Peer, 0, len(peers))
			for _, p := range peers {
				d.PeerDetails = append(d.PeerDetails, model.Peer{
					IP:            p.IP,
					Port:          p.Port,
					DownloadSpeed: p.DownloadSpeed,
					UploadSpeed:   p.UploadSpeed,
					IsSeeder:      p.IsSeeder,
				})
			}
		}
	}

	return d, nil
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

	// 2. Create maps for quick lookup
	gidMap := make(map[string]*engine.DownloadStatus)
	nameMap := make(map[string]*engine.DownloadStatus)
	for _, t := range engineTasks {
		gidMap[t.ID] = t
		if t.Filename != "" {
			nameMap[t.Filename] = t
		}
	}

	// 3. Get all non-complete downloads from repo
	dbDownloads, _, err := s.repo.List(ctx, []string{
		string(model.StatusActive),
		string(model.StatusPaused),
		string(model.StatusWaiting),
	}, 1000, 0)
	if err != nil {
		return err
	}

	// 4. Match and update
	for _, d := range dbDownloads {
		var matched *engine.DownloadStatus

		// Try match by EngineID (GID)
		if d.EngineID != "" {
			matched = gidMap[d.EngineID]
		}

		// Fallback to name match if GID didn't work
		if matched == nil && d.Filename != "" {
			matched = nameMap[d.Filename]
		}

		if matched != nil {
			d.EngineID = matched.ID
			d.Status = model.DownloadStatus(matched.Status)
			s.repo.Update(ctx, d)
		} else if d.Status == model.StatusActive {
			// Only mark as lost if it was active and we really can't find it.
			// But let's be less aggressive: maybe it's just slow to load.
			// We'll mark it as waiting instead of error, so it can be resumed or re-found later.
			d.Status = model.StatusWaiting
			d.Error = "Engine task not found, waiting for recovery"
			s.repo.Update(ctx, d)
		}
	}

	return nil
}

func (s *DownloadService) handleProgress(engineID string, p engine.Progress) {
	ctx := context.Background()

	// 1. Check if this is a per-file progress update from aria2 (format "gid:index")
	if strings.Contains(engineID, ":") {
		parts := strings.Split(engineID, ":")
		if len(parts) == 2 {
			parentGID := parts[0]
			fileIndex, _ := strconv.Atoi(parts[1])

			// Find parent download
			parent, err := s.repo.GetByEngineID(ctx, parentGID)
			if err == nil {
				// Find specific file by parent ID and index
				file, err := s.repo.GetFileByDownloadIDAndIndex(ctx, parent.ID, fileIndex)
				if err == nil {
					file.Downloaded = p.Downloaded
					file.Progress = 0
					if file.Size > 0 {
						file.Progress = int((file.Downloaded * 100) / file.Size)
					}

					oldStatus := file.Status
					if file.Progress == 100 {
						file.Status = model.StatusComplete
					} else {
						file.Status = model.StatusActive
					}

					// Only persist to DB if status changed or 10 seconds passed
					lastPersist := s.lastPersistMap[file.ID]
					if file.Status != oldStatus || time.Since(lastPersist) > 10*time.Second {
						s.repo.UpdateFile(ctx, file)
						s.lastPersistMap[file.ID] = time.Now()
					}

					// Parent aggregate progress is updated via the aggregate event from aria2
					return
				}
			}
		}
	}

	// 2. Try to find if this is a single-file download
	d, err := s.repo.GetByEngineID(ctx, engineID)
	if err == nil {
		// Update in-memory values for event broadcasting
		d.Downloaded = p.Downloaded
		d.Size = p.Size
		d.Speed = p.Speed
		d.ETA = p.ETA
		d.Seeders = p.Seeders
		d.Peers = p.Peers

		// Only persist to DB every 10 seconds to reduce load,
		// but always broadcast progress events to UI.
		lastPersist := s.lastPersistMap[d.ID]
		if time.Since(lastPersist) > 10*time.Second {
			s.repo.Update(ctx, d)
			s.lastPersistMap[d.ID] = time.Now()
		}

		s.publishProgress(d)
		return
	}

	// 3. Try to find if this is an individual file within a magnet (AllDebrid style)
	file, err := s.repo.GetFileByEngineID(ctx, engineID)
	if err == nil {
		file.Downloaded = p.Downloaded
		file.Progress = 0
		if file.Size > 0 {
			file.Progress = int((file.Downloaded * 100) / file.Size)
		}

		oldStatus := file.Status
		file.Status = model.StatusActive

		// Only persist to DB if status changed or 10 seconds passed
		lastPersist := s.lastPersistMap[file.ID]
		if file.Status != oldStatus || time.Since(lastPersist) > 10*time.Second {
			s.repo.UpdateFile(ctx, file)
			s.lastPersistMap[file.ID] = time.Now()
		}

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

	// Only persist to DB every 10 seconds to reduce load,
	// but always broadcast progress events to UI.
	lastPersist := s.lastPersistMap[d.ID]
	if time.Since(lastPersist) > 10*time.Second {
		s.repo.Update(ctx, d)
		s.lastPersistMap[d.ID] = time.Now()
	}

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
			"seeders":    d.Seeders,
			"peers":      d.Peers,
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

		// Also mark all associated files as complete in one batch
		s.repo.MarkAllFilesComplete(ctx, d.ID)

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
