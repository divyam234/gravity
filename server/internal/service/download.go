package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
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
	mu             sync.RWMutex
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

	// Merge live stats from engine if active
	if d.Status == model.StatusActive && d.EngineID != "" {
		status, err := s.engine.Status(ctx, d.EngineID)
		if err == nil {
			d.Downloaded = status.Downloaded
			d.Size = status.Size
			d.Speed = status.Speed
			d.ETA = status.Eta
			d.Seeders = status.Seeders
			d.Peers = status.Peers

			// Also fetch detailed peers for active aria2 downloads
			if d.MagnetSource == "aria2" || !d.IsMagnet {
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
		}
	}

	return d, nil
}

func (s *DownloadService) List(ctx context.Context, status []string, limit, offset int) ([]*model.Download, int, error) {
	downloads, total, err := s.repo.List(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Fetch all engine tasks to merge stats
	engineTasks, err := s.engine.List(ctx)
	if err == nil {
		// Map by ID for quick lookup
		liveMap := make(map[string]*engine.DownloadStatus)
		for _, t := range engineTasks {
			liveMap[t.ID] = t
		}

		// Merge live data into results
		for _, d := range downloads {
			if d.EngineID != "" {
				if live, ok := liveMap[d.EngineID]; ok {
					d.Downloaded = live.Downloaded
					d.Size = live.Size
					d.Speed = live.Speed
					d.ETA = live.Eta
					d.Seeders = live.Seeders
					d.Peers = live.Peers
				}
			}
		}
	}

	return downloads, total, nil
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
	d.Downloaded = 0
	d.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, d); err != nil {
		return err
	}

	s.bus.Publish(event.Event{
		Type:      event.DownloadStarted,
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

	return nil
}

func (s *DownloadService) GetFiles(ctx context.Context, id string) ([]model.DownloadFile, error) {
	return s.repo.GetFiles(ctx, id)
}

func (s *DownloadService) Sync(ctx context.Context) error {
	engineTasks, err := s.engine.List(ctx)
	if err != nil {
		return err
	}

	gidMap := make(map[string]*engine.DownloadStatus)
	nameMap := make(map[string]*engine.DownloadStatus)
	for _, t := range engineTasks {
		gidMap[t.ID] = t
		if t.Filename != "" {
			nameMap[t.Filename] = t
		}
	}

	dbDownloads, _, err := s.repo.List(ctx, []string{
		string(model.StatusActive),
		string(model.StatusPaused),
		string(model.StatusWaiting),
	}, 1000, 0)
	if err != nil {
		return err
	}

	for _, d := range dbDownloads {
		var matched *engine.DownloadStatus
		if d.EngineID != "" {
			matched = gidMap[d.EngineID]
		}
		if matched == nil && d.Filename != "" {
			matched = nameMap[d.Filename]
		}

		if matched != nil {
			d.EngineID = matched.ID
			d.Status = model.DownloadStatus(matched.Status)
			s.repo.Update(ctx, d)
		} else if d.Status == model.StatusActive {
			d.Status = model.StatusWaiting
			d.Error = "Engine task not found, waiting for recovery"
			s.repo.Update(ctx, d)
		}
	}

	return nil
}

func (s *DownloadService) handleProgress(engineID string, p engine.Progress) {
	ctx := context.Background()

	// 1. Check for per-file progress update (gid:index)
	if strings.Contains(engineID, ":") {
		parts := strings.Split(engineID, ":")
		if len(parts) == 2 {
			parentGID := parts[0]
			fileIndex, _ := strconv.Atoi(parts[1])

			parent, err := s.repo.GetByEngineID(ctx, parentGID)
			if err == nil {
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

					s.mu.Lock()
					lastPersist := s.lastPersistMap[file.ID]
					if file.Status != oldStatus || time.Since(lastPersist) > 10*time.Second {
						s.repo.UpdateFile(ctx, file)
						s.lastPersistMap[file.ID] = time.Now()
					}
					s.mu.Unlock()
					return
				}
			}
		}
	}

	// 2. Try single-file download
	d, err := s.repo.GetByEngineID(ctx, engineID)
	if err == nil {
		d.Downloaded = p.Downloaded
		d.Size = p.Size
		d.Speed = p.Speed
		d.ETA = p.ETA
		d.Seeders = p.Seeders
		d.Peers = p.Peers

		s.mu.Lock()
		lastPersist := s.lastPersistMap[d.ID]
		if time.Since(lastPersist) > 10*time.Second {
			s.repo.Update(ctx, d)
			s.lastPersistMap[d.ID] = time.Now()
		}
		s.mu.Unlock()

		s.publishProgress(d)
		return
	}

	// 3. Try file within magnet (AllDebrid)
	file, err := s.repo.GetFileByEngineID(ctx, engineID)
	if err == nil {
		file.Downloaded = p.Downloaded
		file.Progress = 0
		if file.Size > 0 {
			file.Progress = int((file.Downloaded * 100) / file.Size)
		}

		oldStatus := file.Status
		file.Status = model.StatusActive

		s.mu.Lock()
		lastPersist := s.lastPersistMap[file.ID]
		if file.Status != oldStatus || time.Since(lastPersist) > 10*time.Second {
			s.repo.UpdateFile(ctx, file)
			s.lastPersistMap[file.ID] = time.Now()
		}
		s.mu.Unlock()

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

	for _, f := range files {
		totalDownloaded += f.Downloaded
		totalSize += f.Size
		if f.Status == model.StatusComplete {
			filesComplete++
		}
	}

	d.Downloaded = totalDownloaded
	d.Size = totalSize
	d.FilesComplete = filesComplete

	if filesComplete == len(files) && len(files) > 0 && d.Status != model.StatusComplete && d.Status != model.StatusUploading {
		if d.Destination != "" {
			d.Status = model.StatusUploading
		} else {
			d.Status = model.StatusComplete
		}
		now := time.Now()
		d.CompletedAt = &now
	}

	s.mu.Lock()
	lastPersist := s.lastPersistMap[d.ID]
	if d.Status == model.StatusComplete || time.Since(lastPersist) > 10*time.Second {
		s.repo.Update(ctx, d)
		s.lastPersistMap[d.ID] = time.Now()
	}
	s.mu.Unlock()

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
		s.repo.MarkAllFilesComplete(ctx, d.ID)

		s.bus.Publish(event.Event{
			Type:      event.DownloadCompleted,
			Timestamp: time.Now(),
			Data:      d,
		})
		return
	}

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

	file, repoErr := s.repo.GetFileByEngineID(ctx, engineID)
	if repoErr == nil {
		file.Status = model.StatusError
		file.Error = err.Error()
		s.repo.UpdateFile(ctx, file)

		parent, err := s.repo.Get(ctx, file.DownloadID)
		if err == nil {
			s.updateAggregateProgress(ctx, parent)
		}
		return
	}
}
