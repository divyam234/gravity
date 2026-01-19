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

// MockDownloadEngine for testing
type mockDownloadEngine struct {
	onProgress func(string, engine.Progress)
}

func (m *mockDownloadEngine) Start(ctx context.Context) error { return nil }
func (m *mockDownloadEngine) Stop() error                     { return nil }
func (m *mockDownloadEngine) Add(ctx context.Context, url string, opts engine.DownloadOptions) (string, error) {
	return "mock_gid_" + uuid.New().String()[:4], nil
}
func (m *mockDownloadEngine) Pause(ctx context.Context, id string) error  { return nil }
func (m *mockDownloadEngine) Resume(ctx context.Context, id string) error { return nil }
func (m *mockDownloadEngine) Cancel(ctx context.Context, id string) error { return nil }
func (m *mockDownloadEngine) Remove(ctx context.Context, id string) error { return nil }
func (m *mockDownloadEngine) Status(ctx context.Context, id string) (*engine.DownloadStatus, error) {
	return &engine.DownloadStatus{ID: id, Status: "active"}, nil
}
func (m *mockDownloadEngine) List(ctx context.Context) ([]*engine.DownloadStatus, error) {
	return nil, nil
}
func (m *mockDownloadEngine) OnProgress(h func(string, engine.Progress)) { m.onProgress = h }
func (m *mockDownloadEngine) OnComplete(h func(string, string))          {}
func (m *mockDownloadEngine) OnError(h func(string, error))              {}

func TestDownloadFlow(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "gravity-integration-*")
	defer os.RemoveAll(tempDir)

	s, _ := store.New(tempDir)
	bus := event.NewBus()

	// Setup services
	registry := provider.NewRegistry()
	registry.Register(direct.New())
	ps := service.NewProviderService(store.NewProviderRepo(s.GetDB()), registry)

	me := &mockDownloadEngine{}
	ds := service.NewDownloadService(store.NewDownloadRepo(s.GetDB()), me, bus, ps)

	handler := api.NewDownloadHandler(ds)

	// 1. Create a download
	ch := bus.Subscribe()
	defer bus.Unsubscribe(ch)

	payload := map[string]string{
		"url":      "https://example.com/test.zip",
		"filename": "test.zip",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/downloads", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	var download model.Download
	json.Unmarshal(w.Body.Bytes(), &download)

	if download.URL != "https://example.com/test.zip" {
		t.Errorf("expected URL https://example.com/test.zip, got %s", download.URL)
	}

	if download.Status != model.StatusDownloading {
		t.Errorf("expected status downloading, got %s", download.Status)
	}

	// 2. Verify event was published
	select {
	case ev := <-ch:
		if ev.Type != event.DownloadCreated {
			t.Errorf("expected event download.created, got %s", ev.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event")
	}

	// 3. List downloads
	req = httptest.NewRequest("GET", "/api/v1/downloads", nil)
	w = httptest.NewRecorder()
	handler.List(w, req)
}
