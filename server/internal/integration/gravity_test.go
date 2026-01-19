package integration

import (
	"bytes"
	"context"
	"encoding/json"
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
	return &engine.DownloadStatus{ID: id, Status: "active"}, nil
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
	onComplete func(string)
}

func (m *mockUploadEngine) Start(ctx context.Context) error { return nil }
func (m *mockUploadEngine) Stop() error                     { return nil }
func (m *mockUploadEngine) Upload(ctx context.Context, src, dst string, opts engine.UploadOptions) (string, error) {
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
	ds := service.NewDownloadService(dr, me, bus, ps)
	us := service.NewUploadService(dr, mue, bus)
	ss := service.NewStatsService(store.NewStatsRepo(s.GetDB()), me, mue, bus)

	us.Start()
	ss.Start()

	return s, bus, ds, us, ss, me, mue
}

// --- Tests ---

func TestDownloadLifecycle(t *testing.T) {
	_, _, ds, _, _, me, _ := setupTest(t)
	handler := api.NewDownloadHandler(ds)

	// 1. Create
	body := `{"url": "https://example.com/file.zip", "filename": "file.zip"}`
	req := httptest.NewRequest("POST", "/api/v1/downloads", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Create failed: %d", w.Code)
	}

	var d model.Download
	json.Unmarshal(w.Body.Bytes(), &d)
	id := d.ID

	// 2. Pause
	req = httptest.NewRequest("POST", "/api/v1/downloads/"+id+"/pause", nil)
	w = httptest.NewRecorder()
	handler.Pause(w, req) // Uses chi params, mock might fail if not using router?
	// api.Handler functions expect chi context. We need to wrap or mock chi.
	// Easiest is to call Service directly for logic tests, or use router.
	// Let's use Service directly for lifecycle logic to avoid routing boilerplate in test.
	
	err := ds.Pause(context.Background(), id)
	if err != nil {
		t.Errorf("Pause failed: %v", err)
	}

	dUpdated, _ := ds.Get(context.Background(), id)
	if dUpdated.Status != model.StatusPaused {
		t.Errorf("expected paused, got %s", dUpdated.Status)
	}

	// 3. Resume
	ds.Resume(context.Background(), id)
	dUpdated, _ = ds.Get(context.Background(), id)
	if dUpdated.Status != model.StatusActive {
		t.Errorf("expected active, got %s", dUpdated.Status)
	}

	// 4. Complete (simulate engine event)
	if me.onComplete != nil {
		me.onComplete(dUpdated.EngineID, "/tmp/file.zip")
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
	
	// Simulate download complete
	if me.onComplete != nil {
		me.onComplete(d.EngineID, "/tmp/up.zip")
	}
	
	time.Sleep(100 * time.Millisecond)

	// Check if upload started
	d, _ = ds.Get(context.Background(), d.ID)
	if d.Status != model.StatusUploading {
		t.Errorf("expected uploading, got %s (upload logic might be async)", d.Status)
	}
	if d.UploadJobID == "" {
		t.Error("expected UploadJobID to be set")
	}

	// Simulate upload complete
	if mue.onComplete != nil {
		mue.onComplete(d.UploadJobID)
	}

	time.Sleep(100 * time.Millisecond)

	d, _ = ds.Get(context.Background(), d.ID)
	if d.UploadStatus != "complete" {
		t.Errorf("expected upload complete, got %s", d.UploadStatus)
	}
}

func TestStats(t *testing.T) {
	s, _, ds, _, ss, me, _ := setupTest(t)
	handler := api.NewStatsHandler(ss)

	// Create and complete a download
	d, _ := ds.Create(context.Background(), "https://example.com/100mb.bin", "100mb.bin", "")
	// Manually set size in repo since mock engine doesn't update it
	d.Size = 1024 * 1024 * 100 // 100MB
	store.NewDownloadRepo(s.GetDB()).Update(context.Background(), d)

	if me.onComplete != nil {
		me.onComplete(d.EngineID, "path")
	}
	time.Sleep(100 * time.Millisecond)

	// Verify stats
	req := httptest.NewRequest("GET", "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	handler.GetCurrent(w, req)

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)

	totals := res["totals"].(map[string]interface{})
	if totals["downloads_completed"].(float64) != 1 {
		t.Errorf("expected 1 completed, got %v", totals["downloads_completed"])
	}
	if totals["total_downloaded"].(float64) != 104857600 {
		t.Errorf("expected 100MB downloaded, got %v", totals["total_downloaded"])
	}
}

func TestSettings(t *testing.T) {
	s, _, _, _, _, me, _ := setupTest(t)
	repo := store.NewSettingsRepo(s.GetDB())
	handler := api.NewSettingsHandler(repo, me)

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

	// Verify engine configured
	if me.config["max-concurrent-downloads"] != "10" {
		t.Errorf("engine not configured correctly")
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

	d, _ = ds.Get(context.Background(), d.ID)
	if d.Status != model.StatusActive {
		t.Errorf("expected active after retry, got %s", d.Status)
	}
	if d.Error != "" {
		t.Errorf("expected error cleared, got %s", d.Error)
	}
}