package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"gravity/internal/api"
	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/provider"
	"gravity/internal/provider/direct"
	"gravity/internal/service"
	"gravity/internal/store"

	"github.com/google/uuid"
)

// --- Mocks ---

type mockDownloadEngine struct {
	onProgress func(string, engine.Progress)
	onComplete func(string, string)
	onError    func(string, error)
	config     map[string]string
}

func (m *mockDownloadEngine) Start(ctx context.Context) error { return nil }
func (m *mockDownloadEngine) Stop() error                     { return nil }
func (m *mockDownloadEngine) Add(ctx context.Context, url string, opts engine.DownloadOptions) (string, error) {
	return "gid_" + uuid.New().String()[:8], nil
}
func (m *mockDownloadEngine) Pause(ctx context.Context, id string) error  { return nil }
func (m *mockDownloadEngine) Resume(ctx context.Context, id string) error { return nil }
func (m *mockDownloadEngine) Cancel(ctx context.Context, id string) error { return nil }
func (m *mockDownloadEngine) Remove(ctx context.Context, id string) error { return nil }
func (m *mockDownloadEngine) Status(ctx context.Context, id string) (*engine.DownloadStatus, error) {
	return &engine.DownloadStatus{ID: id, Status: "active", Dir: "/tmp", Filename: "file.zip"}, nil
}
func (m *mockDownloadEngine) GetPeers(ctx context.Context, id string) ([]engine.DownloadPeer, error) {
	return []engine.DownloadPeer{}, nil
}
func (m *mockDownloadEngine) List(ctx context.Context) ([]*engine.DownloadStatus, error) {
	return []*engine.DownloadStatus{}, nil
}
func (m *mockDownloadEngine) Sync(ctx context.Context) error {
	return nil
}
func (m *mockDownloadEngine) Configure(ctx context.Context, options map[string]string) error {
	m.config = options
	return nil
}
func (m *mockDownloadEngine) Version(ctx context.Context) (string, error) {
	return "mock-dl-1.0", nil
}
func (m *mockDownloadEngine) OnProgress(h func(string, engine.Progress)) { m.onProgress = h }
func (m *mockDownloadEngine) OnComplete(h func(string, string))          { m.onComplete = h }
func (m *mockDownloadEngine) OnError(h func(string, error))              { m.onError = h }

type mockUploadEngine struct {
	onComplete     func(string)
	lastTrackingID string // Store the tracking ID from the last upload
}

func (m *mockUploadEngine) Start(ctx context.Context) error { return nil }
func (m *mockUploadEngine) Stop() error                     { return nil }
func (m *mockUploadEngine) Upload(ctx context.Context, src, dst string, opts engine.UploadOptions) (string, error) {
	m.lastTrackingID = opts.TrackingID // Store tracking ID for test callbacks
	// Return the provided job ID if given, otherwise generate one
	if opts.JobID != 0 {
		return fmt.Sprintf("%d", opts.JobID), nil
	}
	return "job_" + uuid.New().String()[:8], nil
}
func (m *mockUploadEngine) Cancel(ctx context.Context, jobID string) error { return nil }
func (m *mockUploadEngine) Status(ctx context.Context, jobID string) (*engine.UploadStatus, error) {
	return &engine.UploadStatus{JobID: jobID, Status: "running"}, nil
}
func (m *mockUploadEngine) GetGlobalStats(ctx context.Context) (*engine.GlobalStats, error) {
	return &engine.GlobalStats{Speed: 1024, ActiveTransfers: 1}, nil
}
func (m *mockUploadEngine) Version(ctx context.Context) (string, error) {
	return "mock-ul-1.0", nil
}
func (m *mockUploadEngine) ListRemotes(ctx context.Context) ([]engine.Remote, error) {
	return []engine.Remote{{Name: "gdrive", Type: "drive"}}, nil
}
func (m *mockUploadEngine) CreateRemote(ctx context.Context, name, rtype string, config map[string]string) error {
	return nil
}
func (m *mockUploadEngine) DeleteRemote(ctx context.Context, name string) error { return nil }
func (m *mockUploadEngine) TestRemote(ctx context.Context, name string) error   { return nil }
func (m *mockUploadEngine) OnProgress(h func(string, engine.UploadProgress))    {}
func (m *mockUploadEngine) OnComplete(h func(string))                           { m.onComplete = h }
func (m *mockUploadEngine) OnError(h func(string, error))                       {}
func (m *mockUploadEngine) Configure(ctx context.Context, options map[string]string) error {
	return nil
}

// --- Helper ---

func setupTest(t *testing.T) (*store.Store, *event.Bus, *service.DownloadService, *service.UploadService, *service.StatsService, *mockDownloadEngine, *mockUploadEngine) {
	tempDir, _ := os.MkdirTemp("", "gravity-test-*")
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	s, err := store.New(tempDir)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	bus := event.NewBus()
	me := &mockDownloadEngine{}
	mue := &mockUploadEngine{}

	registry := provider.NewRegistry()
	registry.Register(direct.New())
	ps := service.NewProviderService(store.NewProviderRepo(s.GetDB()), registry)

	dr := store.NewDownloadRepo(s.GetDB())
	setr := store.NewSettingsRepo(s.GetDB())
	ds := service.NewDownloadService(dr, setr, me, mue, bus, ps)
	us := service.NewUploadService(dr, setr, mue, bus)
	ss := service.NewStatsService(store.NewStatsRepo(s.GetDB()), dr, me, mue, bus)

	us.Start()
	ss.Start()

	return s, bus, ds, us, ss, me, mue
}

// --- Tests ---

func TestDownloadLifecycle(t *testing.T) {
	_, _, ds, _, _, me, _ := setupTest(t)
	dh := api.NewDownloadHandler(ds)

	// 1. Create
	body := `{"url": "https://example.com/file.zip", "filename": "file.zip"}`
	req := httptest.NewRequest("POST", "/api/v1/downloads", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	dh.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create failed: %d, body: %s", w.Code, w.Body.String())
	}

	var d model.Download
	json.Unmarshal(w.Body.Bytes(), &d)
	id := d.ID

	// Initial status should be waiting
	if d.Status != model.StatusWaiting {
		t.Errorf("expected status waiting, got %s", d.Status)
	}

	// Manually process queue to make it active
	ds.ProcessQueue(context.Background())

	dActive, _ := ds.Get(context.Background(), id)
	if dActive.Status != model.StatusActive {
		t.Errorf("expected status active after processing queue, got %s", dActive.Status)
	}

	// 2. Pause
	err := ds.Pause(context.Background(), id)
	if err != nil {
		t.Fatalf("Pause failed: %v", err)
	}

	dPaused, _ := ds.Get(context.Background(), id)
	if dPaused.Status != model.StatusPaused {
		t.Errorf("expected paused, got %s", dPaused.Status)
	}

	// 3. Resume
	ds.Resume(context.Background(), id)
	dWaiting, _ := ds.Get(context.Background(), id)
	if dWaiting.Status != model.StatusWaiting {
		t.Errorf("expected waiting after resume, got %s", dWaiting.Status)
	}

	// Process queue again
	ds.ProcessQueue(context.Background())
	dActive2, _ := ds.Get(context.Background(), id)
	if dActive2.Status != model.StatusActive {
		t.Errorf("expected active after queue processing, got %s", dActive2.Status)
	}

	// 4. Complete (simulate engine event)
	if me.onComplete != nil {
		me.onComplete(dActive2.EngineID, "/tmp/file.zip")
	}

	// Wait for event bus
	time.Sleep(100 * time.Millisecond)

	dFinal, _ := ds.Get(context.Background(), id)
	if dFinal.Status != model.StatusComplete {
		t.Errorf("expected complete, got %s", dFinal.Status)
	}
}

func TestUploadLifecycle(t *testing.T) {
	_, _, ds, _, _, me, mue := setupTest(t)

	// Create with destination to trigger upload
	d, _ := ds.Create(context.Background(), "https://example.com/up.zip", "up.zip", "gdrive:/")
	ds.ProcessQueue(context.Background())
	dActive, _ := ds.Get(context.Background(), d.ID)

	// Simulate download complete
	if me.onComplete != nil {
		me.onComplete(dActive.EngineID, "/tmp/up.zip")
	}

	time.Sleep(100 * time.Millisecond)

	// Check if upload started
	dUp, _ := ds.Get(context.Background(), d.ID)
	if dUp.Status != model.StatusUploading {
		t.Errorf("expected uploading, got %s", dUp.Status)
	}
	if dUp.UploadJobID == "" {
		t.Error("expected UploadJobID to be set")
	}

	// Simulate upload complete
	if mue.onComplete != nil {
		mue.onComplete(d.ID)
	}

	time.Sleep(100 * time.Millisecond)

	dDone, _ := ds.Get(context.Background(), d.ID)
	if dDone.UploadStatus != "complete" {
		t.Errorf("expected upload complete, got %s", dDone.UploadStatus)
	}
}

func TestStats(t *testing.T) {
	s, _, ds, _, ss, me, _ := setupTest(t)
	handler := api.NewStatsHandler(ss)

	// Create and complete a download
	d, _ := ds.Create(context.Background(), "https://example.com/100mb.bin", "100mb.bin", "")
	ds.ProcessQueue(context.Background())
	dActive, _ := ds.Get(context.Background(), d.ID)

	// Manually set size in repo
	dActive.Size = 1024 * 1024 * 100 // 100MB
	store.NewDownloadRepo(s.GetDB()).Update(context.Background(), dActive)

	if me.onComplete != nil {
		me.onComplete(dActive.EngineID, "path")
	}
	time.Sleep(100 * time.Millisecond)

	// Verify stats
	req := httptest.NewRequest("GET", "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	handler.GetCurrent(w, req)

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)

	totals := res["totals"].(map[string]interface{})
	if totals["tasksFinished"].(float64) != 1 {
		t.Errorf("expected 1 completed, got %v", totals["tasksFinished"])
	}
}

func TestSettings(t *testing.T) {
	s, _, _, _, _, me, mue := setupTest(t)
	repo := store.NewSettingsRepo(s.GetDB())
	pr := store.NewProviderRepo(s.GetDB())
	handler := api.NewSettingsHandler(repo, pr, me, mue)

	body := `{"max-concurrent-downloads": "10"}`
	req := httptest.NewRequest("PATCH", "/api/v1/settings", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Update settings failed: %d", w.Code)
	}

	// Verify repo
	val, _ := repo.Get(context.Background())
	if val["max-concurrent-downloads"] != "10" {
		t.Errorf("expected 10, got %s", val["max-concurrent-downloads"])
	}
}

func TestRetry(t *testing.T) {
	s, _, ds, _, _, _, _ := setupTest(t)

	d, _ := ds.Create(context.Background(), "https://example.com/fail.zip", "fail.zip", "")

	// Manually set to error
	d.Status = model.StatusError
	d.Error = "Simulated error"
	store.NewDownloadRepo(s.GetDB()).Update(context.Background(), d)

	// Retry
	err := ds.Retry(context.Background(), d.ID)
	if err != nil {
		t.Errorf("Retry failed: %v", err)
	}

	dResumed, _ := ds.Get(context.Background(), d.ID)
	// Retry should put it back to Waiting, then queue picks it up
	if dResumed.Status != model.StatusActive && dResumed.Status != model.StatusWaiting {
		t.Errorf("expected active or waiting after retry, got %s", dResumed.Status)
	}
}
