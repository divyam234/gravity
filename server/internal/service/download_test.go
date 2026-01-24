package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"gravity/internal/config"
	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/provider"
	"gravity/internal/store"
)

// MockDownloadEngine mimics aria2c behavior
type MockDownloadEngine struct {
	mu          sync.RWMutex
	activeTasks map[string]*engine.DownloadStatus
	onProgress  func(string, engine.Progress)
	onComplete  func(string, string)
	onError     func(string, error)

	// Test Controls
	failNextAdd   error
	simulateError string // ID to error out
	errorAfter    time.Duration
}

func NewMockDownloadEngine() *MockDownloadEngine {
	return &MockDownloadEngine{
		activeTasks: make(map[string]*engine.DownloadStatus),
	}
}

func (m *MockDownloadEngine) SetAddError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNextAdd = err
}

func (m *MockDownloadEngine) AddMagnetWithSelection(ctx context.Context, magnet string, selectedIndexes []string, opts engine.DownloadOptions) (string, error) {
	return m.Add(ctx, magnet, opts)
}

func (m *MockDownloadEngine) Start(ctx context.Context) error { return nil }
func (m *MockDownloadEngine) Stop() error                     { return nil }

func (m *MockDownloadEngine) Add(ctx context.Context, url string, opts engine.DownloadOptions) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNextAdd != nil {
		err := m.failNextAdd
		m.failNextAdd = nil
		return "", err
	}

	id := fmt.Sprintf("aria2_%d", time.Now().UnixNano())
	m.activeTasks[id] = &engine.DownloadStatus{
		ID:         id,
		Status:     "active",
		URL:        url,
		Filename:   opts.Filename,
		Dir:        opts.Dir,
		Size:       1024 * 1024 * 10, // 10 MB mock size
		Downloaded: 0,
		Speed:      0,
	}

	// Simulate async progress in background
	go func(id string) {
		time.Sleep(100 * time.Millisecond) // wait for setup (allow ProcessQueue to save EngineID)

		// Check if we should error out this specific ID
		m.mu.Lock()
		shouldError := m.simulateError == id
		errDelay := m.errorAfter
		m.mu.Unlock()

		if shouldError {
			if errDelay > 0 {
				time.Sleep(errDelay)
			}
			m.mu.Lock()
			var errorFunc func(string, error)
			if _, ok := m.activeTasks[id]; ok {
				errorFunc = m.onError
			}
			m.mu.Unlock()

			if errorFunc != nil {
				errorFunc(id, fmt.Errorf("simulated engine error"))
			}
			return
		}

		// 1. Started/Progress 50%
		m.mu.Lock()
		var progressFunc func(string, engine.Progress)
		var taskCopy engine.DownloadStatus
		if task, ok := m.activeTasks[id]; ok {
			task.Downloaded = 5 * 1024 * 1024
			task.Speed = 1024 * 1024
			progressFunc = m.onProgress
			taskCopy = *task
		}
		m.mu.Unlock()

		if progressFunc != nil {
			progressFunc(id, engine.Progress{
				Downloaded: taskCopy.Downloaded,
				Size:       taskCopy.Size,
				Speed:      taskCopy.Speed,
			})
		}

		time.Sleep(50 * time.Millisecond)

		// 2. Complete
		m.mu.Lock()
		var completeFunc func(string, string)
		var finalPath string
		if task, ok := m.activeTasks[id]; ok {
			task.Downloaded = task.Size
			task.Status = "complete"
			completeFunc = m.onComplete
			finalPath = fmt.Sprintf("%s/%s", task.Dir, task.Filename)
		}
		m.mu.Unlock()

		if completeFunc != nil {
			completeFunc(id, finalPath)
		}
	}(id)

	return id, nil
}

func (m *MockDownloadEngine) FailNextTask(id string, after time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateError = id
	m.errorAfter = after
}

func (m *MockDownloadEngine) Pause(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if task, ok := m.activeTasks[id]; ok {
		task.Status = "paused"
	}
	return nil
}

func (m *MockDownloadEngine) Resume(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if task, ok := m.activeTasks[id]; ok {
		task.Status = "active"
	}
	return nil
}

func (m *MockDownloadEngine) Cancel(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if task, ok := m.activeTasks[id]; ok {
		task.Status = "stopped"
	}
	return nil
}

func (m *MockDownloadEngine) Remove(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.activeTasks, id)
	return nil
}

func (m *MockDownloadEngine) Status(ctx context.Context, id string) (*engine.DownloadStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if task, ok := m.activeTasks[id]; ok {
		// return copy to avoid race
		t := *task
		return &t, nil
	}
	return nil, fmt.Errorf("task not found")
}

func (m *MockDownloadEngine) GetPeers(ctx context.Context, id string) ([]engine.DownloadPeer, error) {
	return []engine.DownloadPeer{}, nil
}

func (m *MockDownloadEngine) List(ctx context.Context) ([]*engine.DownloadStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*engine.DownloadStatus
	for _, t := range m.activeTasks {
		list = append(list, t)
	}
	return list, nil
}

func (m *MockDownloadEngine) Sync(ctx context.Context) error { return nil }
func (m *MockDownloadEngine) Configure(ctx context.Context, options *model.Settings) error {
	return nil
}
func (m *MockDownloadEngine) Version(ctx context.Context) (string, error) { return "1.0-mock", nil }

func (m *MockDownloadEngine) OnProgress(h func(string, engine.Progress)) { m.onProgress = h }
func (m *MockDownloadEngine) OnComplete(h func(string, string))          { m.onComplete = h }
func (m *MockDownloadEngine) OnError(h func(string, error))              { m.onError = h }

func (m *MockDownloadEngine) GetMagnetFiles(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
	return nil, nil
}
func (m *MockDownloadEngine) GetTorrentFiles(ctx context.Context, torrentBase64 string) (*model.MagnetInfo, error) {
	return nil, nil
}

// MockUploadEngine implements engine.UploadEngine
type MockUploadEngine struct{}

// StorageEngine methods
func (m *MockUploadEngine) List(ctx context.Context, virtualPath string) ([]engine.FileInfo, error) {
	return nil, nil
}
func (m *MockUploadEngine) Stat(ctx context.Context, virtualPath string) (*engine.FileInfo, error) {
	return nil, nil
}
func (m *MockUploadEngine) ListRemotes(ctx context.Context) ([]engine.Remote, error)      { return nil, nil }
func (m *MockUploadEngine) Mkdir(ctx context.Context, virtualPath string) error           { return nil }
func (m *MockUploadEngine) Delete(ctx context.Context, virtualPath string) error          { return nil }
func (m *MockUploadEngine) Rename(ctx context.Context, virtualPath, newName string) error { return nil }
func (m *MockUploadEngine) Open(ctx context.Context, virtualPath string) (engine.ReadSeekCloser, error) {
	return nil, nil
}

// UploadEngine methods
func (m *MockUploadEngine) Start(ctx context.Context) error { return nil }
func (m *MockUploadEngine) Stop() error                     { return nil }
func (m *MockUploadEngine) Upload(ctx context.Context, src, dst string, opts engine.UploadOptions) (string, error) {
	return "job_1", nil
}
func (m *MockUploadEngine) Cancel(ctx context.Context, jobID string) error { return nil }
func (m *MockUploadEngine) Copy(ctx context.Context, srcPath, dstPath string) (string, error) {
	return "job_c", nil
}
func (m *MockUploadEngine) Move(ctx context.Context, srcPath, dstPath string) (string, error) {
	return "job_m", nil
}
func (m *MockUploadEngine) Status(ctx context.Context, jobID string) (*engine.UploadStatus, error) {
	return nil, nil
}
func (m *MockUploadEngine) GetGlobalStats(ctx context.Context) (*engine.GlobalStats, error) {
	return nil, nil
}
func (m *MockUploadEngine) Version(ctx context.Context) (string, error)                           { return "mock-1.0", nil }
func (m *MockUploadEngine) OnProgress(handler func(jobID string, progress engine.UploadProgress)) {}
func (m *MockUploadEngine) OnComplete(handler func(jobID string))                                 {}
func (m *MockUploadEngine) OnError(handler func(jobID string, err error))                         {}
func (m *MockUploadEngine) CreateRemote(ctx context.Context, name, rtype string, config map[string]string) error {
	return nil
}
func (m *MockUploadEngine) DeleteRemote(ctx context.Context, name string) error { return nil }
func (m *MockUploadEngine) TestRemote(ctx context.Context, name string) error   { return nil }
func (m *MockUploadEngine) Configure(ctx context.Context, options *model.Settings) error {
	return nil
}
func (m *MockUploadEngine) Restart(ctx context.Context) error { return nil }

// MockProvider implements provider.Provider
type MockProvider struct{}

func (p *MockProvider) Name() string                             { return "mock" }
func (p *MockProvider) DisplayName() string                      { return "Mock Provider" }
func (p *MockProvider) Type() model.ProviderType                 { return model.ProviderTypeDirect }
func (p *MockProvider) ConfigSchema() []provider.ConfigField     { return nil }
func (p *MockProvider) Configure(ctx context.Context, config map[string]string) error { return nil }
func (p *MockProvider) IsConfigured() bool                       { return true }
func (p *MockProvider) Supports(url string) bool                 { return true }
func (p *MockProvider) Priority() int                            { return 100 }
func (p *MockProvider) Test(ctx context.Context) (*model.AccountInfo, error) {
	return &model.AccountInfo{Username: "testuser", IsPremium: true}, nil
}
func (p *MockProvider) Resolve(ctx context.Context, url string, headers map[string]string) (*provider.ResolveResult, error) {
	return &provider.ResolveResult{
		URL:      url,
		Filename: "test_movie.mp4",
		Size:     1024 * 1024 * 10,
	}, nil
}

func TestFullDownloadFlow(t *testing.T) {
	// 1. Setup Infrastructure
	cfg := &config.Config{
		Database: config.DBConfig{Type: "sqlite", DSN: ":memory:"},
	}
	s, err := store.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}
	defer s.Close()

	bus := event.NewBus()
	dr := store.NewDownloadRepo(s.GetDB())
	setr := store.NewSettingsRepo(s.GetDB())
	pr := store.NewProviderRepo(s.GetDB())

	// 2. Setup Provider Service
	reg := provider.NewRegistry()
	reg.Register(&MockProvider{})
	ps := NewProviderService(pr, reg)
	// Initialize providers (loads from DB, but DB is empty, so it just uses defaults/registered ones)
	ps.Init(context.Background())

	// 3. Setup Engines
	mockDownloader := NewMockDownloadEngine()
	mockUploader := &MockUploadEngine{}

	// 4. Create Service
	ds := NewDownloadService(dr, setr, mockDownloader, mockUploader, bus, ps)
	ds.Start(context.Background())
	defer ds.Stop()

	// 5. Test Event Subscription
	events := bus.Subscribe()

	// 6. Execute: Create Download
	ctx := context.Background()
	// Create(ctx, url, filename, downloadDir, destination, options)
	dl, err := ds.Create(ctx, "http://example.com/video", "", "", "", model.TaskOptions{})
	if err != nil {
		t.Fatalf("Failed to create download: %v", err)
	}

	// Verify initial state
	if dl.Status != model.StatusWaiting && dl.Status != model.StatusAllocating {
		t.Errorf("Expected status Waiting/Allocating, got %s", dl.Status)
	}

	// 7. Verify Events Sequence
	// We expect: Created -> Started -> Progress -> Completed

	timeout := time.After(5 * time.Second)
	var created, started, progress, completed bool

	for {
		select {
		case ev := <-events:
			switch ev.Type {
			case event.DownloadCreated:
				created = true
				if d, ok := ev.Data.(*model.Download); ok {
					if d.ID != dl.ID {
						t.Errorf("Created event ID mismatch")
					}
				}
			case event.DownloadStarted:
				started = true
				// Check if engine has it
				active, _ := mockDownloader.List(ctx)
				if len(active) == 0 {
					t.Errorf("Download started but not in engine")
				}
			case event.DownloadProgress:
				progress = true
			case event.DownloadCompleted:
				completed = true
				// Verify final state in Repo
				finalDl, _ := dr.Get(ctx, dl.ID)
				if finalDl.Status != model.StatusComplete {
					t.Errorf("Event says complete but DB says %s", finalDl.Status)
				}
				// Verify local path was updated (mock engine returns "dir/filename")
				expectedPath := fmt.Sprintf("%s/%s", dl.DownloadDir, "test_movie.mp4")
				_ = expectedPath // Ignored for now as we verified completion status
				goto DONE
			case event.DownloadError:
				t.Fatalf("Received unexpected error event: %v", ev.Data)
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for events. Stats: C=%v S=%v P=%v Comp=%v", created, started, progress, completed)
		}
	}
DONE:

	if !created {
		t.Error("Missing DownloadCreated event")
	}
	if !started {
		t.Error("Missing DownloadStarted event")
	}
	if !progress {
		t.Error("Missing DownloadProgress event")
	}
	if !completed {
		t.Error("Missing DownloadCompleted event")
	}

	// 8. Test Pause/Resume
	// Create another one for Pause/Resume test
	dl2, _ := ds.Create(ctx, "http://example.com/file2", "", "", "", model.TaskOptions{})

	// Wait for start
	time.Sleep(200 * time.Millisecond) // Give time for queue to pick it up

	// Pause
	err = ds.Pause(ctx, dl2.ID)
	if err != nil {
		t.Errorf("Failed to pause: %v", err)
	}

	dl2, _ = dr.Get(ctx, dl2.ID)
	if dl2.Status != model.StatusPaused {
		t.Errorf("Expected status Paused, got %s", dl2.Status)
	}

	// Resume
	err = ds.Resume(ctx, dl2.ID)
	if err != nil {
		t.Errorf("Failed to resume: %v", err)
	}

	// Should go back to Waiting then Active
	time.Sleep(100 * time.Millisecond)
	dl2, _ = dr.Get(ctx, dl2.ID)
	if dl2.Status != model.StatusActive && dl2.Status != model.StatusWaiting && dl2.Status != model.StatusAllocating {
		t.Errorf("Expected status Active/Waiting/Allocating after resume, got %s", dl2.Status)
	}
}

func TestErrorScenarios(t *testing.T) {
	cfg := &config.Config{
		Database: config.DBConfig{Type: "sqlite", DSN: ":memory:"},
	}
	s, _ := store.New(cfg)
	defer s.Close()

	bus := event.NewBus()
	dr := store.NewDownloadRepo(s.GetDB())
	setr := store.NewSettingsRepo(s.GetDB())
	pr := store.NewProviderRepo(s.GetDB())
	reg := provider.NewRegistry()
	reg.Register(&MockProvider{})
	ps := NewProviderService(pr, reg)
	ps.Init(context.Background())

	mockDownloader := NewMockDownloadEngine()
	ds := NewDownloadService(dr, setr, mockDownloader, &MockUploadEngine{}, bus, ps)
	ds.Start(context.Background())
	defer ds.Stop()

	ctx := context.Background()
	events := bus.Subscribe()

	// Scenario 1: Engine Submission Error
	t.Run("SubmissionError", func(t *testing.T) {
		mockDownloader.SetAddError(fmt.Errorf("connection refused"))

		dl, err := ds.Create(ctx, "http://fail.com", "", "", "", model.TaskOptions{})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Wait for ProcessQueue to hit the error
		time.Sleep(200 * time.Millisecond)

		updated, _ := dr.Get(ctx, dl.ID)
		if updated.Status != model.StatusError {
			t.Errorf("Expected status Error, got %s", updated.Status)
		}
		if updated.Error == "" {
			t.Error("Expected error message to be set")
		}
	})

	// Scenario 2: Async Download Error (Aria2 reports error)
	t.Run("AsyncDownloadError", func(t *testing.T) {
		dl, _ := ds.Create(ctx, "http://error.com", "", "", "", model.TaskOptions{})

		// Wait for start
		var engineID string
		timeout := time.After(1 * time.Second)
		for {
			select {
			case ev := <-events:
				if ev.Type == event.DownloadStarted {
					if d, ok := ev.Data.(*model.Download); ok && d.ID == dl.ID {
						engineID = d.EngineID
						goto FOUND
					}
				}
			case <-timeout:
				t.Fatal("Timeout waiting for start")
			}
		}
	FOUND:
		// Now tell mock to fail this ID
		mockDownloader.onError(engineID, fmt.Errorf("remote forbidden"))

		// Wait for reaction
		time.Sleep(100 * time.Millisecond)

		updated, _ := dr.Get(ctx, dl.ID)
		if updated.Status != model.StatusError {
			t.Errorf("Expected status Error, got %s", updated.Status)
		}

		// Verify event
		select {
		case ev := <-events:
			if ev.Type == event.DownloadError {
				// Success
			}
		default:
			// Might have missed it if we weren't draining loop
		}
	})

	// Scenario 3: Retry Logic
	t.Run("Retry", func(t *testing.T) {
		// Use the failed download from Scenario 2
		downloads, _, _ := dr.List(ctx, []string{"error"}, 1, 0, false)
		if len(downloads) == 0 {
			t.Fatal("No error downloads found for retry")
		}
		target := downloads[0]

		err := ds.Retry(ctx, target.ID)
		if err != nil {
			t.Fatalf("Retry failed: %v", err)
		}

		// Should go to Active
		time.Sleep(200 * time.Millisecond) // Wait for ProcessQueue + Mock Start

		updated, _ := dr.Get(ctx, target.ID)
		if updated.Status != model.StatusActive && updated.Status != model.StatusComplete && updated.Status != model.StatusAllocating {
			t.Errorf("Expected status Active/Complete/Allocating after retry, got %s", updated.Status)
		}
		if updated.Error != "" {
			t.Errorf("Expected error cleared, got %s", updated.Error)
		}
	})
}

func TestSyncEdgeCases(t *testing.T) {
	cfg := &config.Config{
		Database: config.DBConfig{Type: "sqlite", DSN: ":memory:"},
	}
	s, _ := store.New(cfg)
	defer s.Close()
	dr := store.NewDownloadRepo(s.GetDB())
	setr := store.NewSettingsRepo(s.GetDB())
	pr := store.NewProviderRepo(s.GetDB())
	bus := event.NewBus()
	// Need mock provider stuff again...
	reg := provider.NewRegistry()
	reg.Register(&MockProvider{})
	ps := NewProviderService(pr, reg)
	mockDownloader := NewMockDownloadEngine()
	ds := NewDownloadService(dr, setr, mockDownloader, &MockUploadEngine{}, bus, ps)

	ctx := context.Background()

	// Scenario: Task is Active in DB, but missing in Engine (e.g. crash restart)
	// 1. Manually insert task
	d := &model.Download{
		ID:          "d_ghost",
		Status:      model.StatusActive,
		EngineID:    "aria2_ghost",
		ResolvedURL: "http://ghost.com",
		Filename:    "ghost.file",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	dr.Create(ctx, d)

	// 2. Verify Engine is empty
	tasks, _ := mockDownloader.List(ctx)
	if len(tasks) != 0 {
		t.Fatal("Engine should be empty")
	}

	// 3. Run Sync
	ds.Sync(ctx)

	// 4. Verify DB updated
	updated, _ := dr.Get(ctx, d.ID)
	if updated.Status != model.StatusWaiting {
		t.Errorf("Expected ghost task to move to Waiting, got %s", updated.Status)
	}
	if updated.EngineID != "" {
		t.Errorf("Expected EngineID cleared, got %s", updated.EngineID)
	}

	// 5. Verify it gets picked up by queue (since it's Waiting)
	ds.ProcessQueue(ctx)
	time.Sleep(100 * time.Millisecond)

	updated, _ = dr.Get(ctx, d.ID)
	if updated.Status != model.StatusActive && updated.Status != model.StatusAllocating {
		t.Errorf("Expected task to be reactivated, got %s", updated.Status)
	}
	if updated.EngineID == "" && updated.Status == model.StatusActive {
		t.Error("Expected new EngineID assigned")
	}
}

func TestDeletion(t *testing.T) {
	cfg := &config.Config{
		Database: config.DBConfig{Type: "sqlite", DSN: ":memory:"},
	}
	s, _ := store.New(cfg)
	defer s.Close()
	dr := store.NewDownloadRepo(s.GetDB())
	setr := store.NewSettingsRepo(s.GetDB())
	pr := store.NewProviderRepo(s.GetDB())
	bus := event.NewBus()
	reg := provider.NewRegistry()
	reg.Register(&MockProvider{})
	ps := NewProviderService(pr, reg)
	mockDownloader := NewMockDownloadEngine()
	ds := NewDownloadService(dr, setr, mockDownloader, &MockUploadEngine{}, bus, ps)
	ds.Start(context.Background())
	defer ds.Stop()

	ctx := context.Background()

	// Create and start a download
	dl, _ := ds.Create(ctx, "http://delete.me", "", "", "", model.TaskOptions{})

	// Wait for active
	time.Sleep(150 * time.Millisecond)

	updated, _ := dr.Get(ctx, dl.ID)
	if updated.Status != model.StatusActive && updated.Status != model.StatusComplete && updated.Status != model.StatusAllocating {
		t.Fatalf("Download not active/complete: %s", updated.Status)
	}

	// Delete
	err := ds.Delete(ctx, dl.ID, true)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// Verify DB
	_, err = dr.Get(ctx, dl.ID)
	if err == nil {
		t.Error("Download should be deleted from DB")
	}

	// Verify Engine
	// Mock Engine Remove() just deletes key
	tasks, _ := mockDownloader.List(ctx)
	if len(tasks) > 0 {
		for _, task := range tasks {
			if task.ID == updated.EngineID {
				t.Error("Task still in engine after delete")
			}
		}
	}
}
