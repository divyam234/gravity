package integration

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gravity/internal/engine/aria2"
	"gravity/internal/engine/rclone"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/provider"
	"gravity/internal/provider/direct"
	"gravity/internal/service"
	"gravity/internal/store"
)

// rangeHandler handles basic range requests for testing
func rangeHandler(content []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "testfile.bin", time.Now(), bytes.NewReader(content))
	}
}

func TestRealFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real flow test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "gravity-real-flow-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	downloadDir := filepath.Join(tempDir, "downloads")
	uploadDir := filepath.Join(tempDir, "uploads")
	os.MkdirAll(downloadDir, 0755)
	os.MkdirAll(uploadDir, 0755)

	testFileContent := make([]byte, 5*1024*1024) // 5MB
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "test.bin", time.Now(), bytes.NewReader(testFileContent))
	}))
	defer ts.Close()

	s, err := store.New(tempDir)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	bus := event.NewBus()
	de := aria2.NewEngine(16800, tempDir)
	ue := rclone.NewEngine(15572)

	ctx := context.Background()
	if err := de.Start(ctx); err != nil {
		t.Fatalf("failed to start aria2 engine: %v", err)
	}
	defer de.Stop()
	if err := ue.Start(ctx); err != nil {
		t.Fatalf("failed to start rclone engine: %v", err)
	}
	defer ue.Stop()

	ue.CreateRemote(ctx, "test-local", "alias", map[string]string{"remote": uploadDir})

	registry := provider.NewRegistry()
	registry.Register(direct.New())
	ps := service.NewProviderService(store.NewProviderRepo(s.GetDB()), registry)
	dr := store.NewDownloadRepo(s.GetDB())
	setr := store.NewSettingsRepo(s.GetDB())
	ds := service.NewDownloadService(dr, setr, de, ue, bus, ps)
	us := service.NewUploadService(dr, ue, bus)
	us.Start()
	ds.Start()

	_, err = ds.Create(ctx, ts.URL, "testfile.bin", "test-local:/")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	done := make(chan struct{})
	sub := bus.Subscribe()
	defer bus.Unsubscribe(sub)

	go func() {
		for ev := range sub {
			if ev.Type == event.UploadCompleted {
				close(done)
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(45 * time.Second):
		t.Fatal("Timeout waiting for upload completion")
	}

	if _, err := os.Stat(filepath.Join(uploadDir, "testfile.bin")); os.IsNotExist(err) {
		t.Error("File missing in upload directory")
	}
}

func TestRealFolderFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real flow test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "gravity-real-folder-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	downloadDir := filepath.Join(tempDir, "downloads")
	uploadDir := filepath.Join(tempDir, "uploads")
	os.MkdirAll(downloadDir, 0755)
	os.MkdirAll(uploadDir, 0755)

	torrentDir := filepath.Join(downloadDir, "MyTorrent")
	os.MkdirAll(torrentDir, 0755)
	os.WriteFile(filepath.Join(torrentDir, "file1.txt"), []byte("content1"), 0644)

	s, err := store.New(tempDir)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	bus := event.NewBus()
	ue := rclone.NewEngine(15573)
	ctx := context.Background()
	ue.Start(ctx)
	defer ue.Stop()

	ue.CreateRemote(ctx, "test-folder-local", "alias", map[string]string{"remote": uploadDir})

	dr := store.NewDownloadRepo(s.GetDB())
	us := service.NewUploadService(dr, ue, bus)
	us.Start()

	d := &model.Download{
		ID:          "d_folder_test",
		Status:      model.StatusComplete,
		Filename:    "MyTorrent",
		LocalPath:   torrentDir,
		Destination: "test-folder-local:/",
		UpdatedAt:   time.Now(),
	}
	dr.Create(ctx, d)
	us.TriggerUpload(ctx, d)

	done := make(chan struct{})
	sub := bus.Subscribe()
	defer bus.Unsubscribe(sub)

	go func() {
		for ev := range sub {
			if ev.Type == event.UploadCompleted {
				close(done)
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("Timeout waiting for folder upload")
	}

	if _, err := os.Stat(filepath.Join(uploadDir, "MyTorrent", "file1.txt")); os.IsNotExist(err) {
		t.Error("Folder/File missing in upload directory")
	}
}

func TestRealRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real flow test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "gravity-real-recovery-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	bus := event.NewBus()
	de := aria2.NewEngine(16801, tempDir)
	ue := rclone.NewEngine(15574)
	s, err := store.New(tempDir)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	ctx := context.Background()
	de.Start(ctx)
	ue.Start(ctx)

	registry := provider.NewRegistry()
	registry.Register(direct.New())
	ps := service.NewProviderService(store.NewProviderRepo(s.GetDB()), registry)
	dr := store.NewDownloadRepo(s.GetDB())
	setr := store.NewSettingsRepo(s.GetDB())
	ds := service.NewDownloadService(dr, setr, de, ue, bus, ps)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.Write(make([]byte, 1024))
	}))
	defer ts.Close()

	d, _ := ds.Create(ctx, ts.URL, "recovery.bin", "")
	ds.ProcessQueue(ctx)

	// Kill engine and restart
	de.Stop()
	time.Sleep(1 * time.Second)

	de2 := aria2.NewEngine(16801, tempDir)
	de2.Start(ctx)
	defer de2.Stop()

	ds2 := service.NewDownloadService(dr, setr, de2, ue, bus, ps)
	err = ds2.Sync(ctx)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	d2, _ := dr.Get(ctx, d.ID)
	// After sync, it should be either active (re-added) or waiting (to be picked up by queue)
	if d2.Status != model.StatusActive && d2.Status != model.StatusWaiting && d2.Status != model.StatusComplete {
		t.Errorf("expected status active, waiting or complete, got %s", d2.Status)
	}
}
