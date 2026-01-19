package integration

import (
	"context"
	"fmt"
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

	testFileContent := make([]byte, 10*1024*1024)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(testFileContent)
	}))
	defer ts.Close()

	s, _ := store.New(tempDir)
	bus := event.NewBus()
	de := aria2.NewEngine(16800, "test-secret", tempDir)
	ue := rclone.NewEngine(15572)

	ctx := context.Background()
	de.Start(ctx)
	defer de.Stop()
	ue.Start(ctx)
	defer ue.Stop()

	ue.CreateRemote(ctx, "test-local", "alias", map[string]string{"remote": uploadDir})

	registry := provider.NewRegistry()
	registry.Register(direct.New())
	ps := service.NewProviderService(store.NewProviderRepo(s.GetDB()), registry)
	dr := store.NewDownloadRepo(s.GetDB())
	ds := service.NewDownloadService(dr, de, ue, bus, ps)
	us := service.NewUploadService(dr, ue, bus)
	us.Start()

	d, _ := ds.Create(ctx, ts.URL, "testfile.bin", "test-local:/")
	if d.ID == "" {
		t.Error("Download ID is empty")
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
	case <-time.After(30 * time.Second):
		t.Fatal("Timeout")
	}

	if _, err := os.Stat(filepath.Join(uploadDir, "testfile.bin")); os.IsNotExist(err) {
		t.Error("File missing")
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

	s, _ := store.New(tempDir)
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
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout")
	}

	if _, err := os.Stat(filepath.Join(uploadDir, "MyTorrent", "file1.txt")); os.IsNotExist(err) {
		t.Error("Folder missing")
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
	de := aria2.NewEngine(16801, "test-secret", tempDir)
	ue := rclone.NewEngine(15574)
	s, _ := store.New(tempDir)
	
	ctx := context.Background()
	de.Start(ctx)
	ue.Start(ctx)
	
	registry := provider.NewRegistry()
	registry.Register(direct.New())
	ps := service.NewProviderService(store.NewProviderRepo(s.GetDB()), registry)
	dr := store.NewDownloadRepo(s.GetDB())
	ds := service.NewDownloadService(dr, de, ue, bus, ps)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write(make([]byte, 1024))
	}))
	defer ts.Close()

	d, _ := ds.Create(ctx, ts.URL, "recovery.bin", "")
	de.Stop()

	de2 := aria2.NewEngine(16801, "test-secret", tempDir)
	ue2 := rclone.NewEngine(15575)
	de2.Start(ctx)
	ue2.Start(ctx)
	defer de2.Stop()
	defer ue2.Stop()

	time.Sleep(2 * time.Second)

	ds2 := service.NewDownloadService(dr, de2, ue2, bus, ps)
	ds2.Sync(ctx)

	d2, _ := dr.Get(ctx, d.ID)
	if d2.Status != model.StatusActive && d2.Status != model.StatusComplete {
		t.Errorf("expected status active or complete, got %s", d2.Status)
	}
}

func TestRealFlowDetailed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real flow test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "gravity-detailed-flow-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	downloadDir := filepath.Join(tempDir, "downloads")
	uploadDir := filepath.Join(tempDir, "uploads")
	os.MkdirAll(downloadDir, 0755)
	os.MkdirAll(uploadDir, 0755)

	fileSize := 50 * 1024 * 1024
	dummyData := make([]byte, fileSize)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))
		chunkSize := 1024 * 1024
		for i := 0; i < fileSize; i += chunkSize {
			end := i + chunkSize
			if end > fileSize {
				end = fileSize
			}
			w.Write(dummyData[i:end])
			time.Sleep(50 * time.Millisecond)
		}
	}))
	defer ts.Close()

	s, _ := store.New(tempDir)
	bus := event.NewBus()
	de := aria2.NewEngine(16802, "test-secret", tempDir)
	ue := rclone.NewEngine(15576)

	ctx := context.Background()
	de.Start(ctx)
	defer de.Stop()
	ue.Start(ctx)
	defer ue.Stop()

	ue.CreateRemote(ctx, "det-local", "alias", map[string]string{"remote": uploadDir})

	dr := store.NewDownloadRepo(s.GetDB())
	reg := provider.NewRegistry()
	reg.Register(direct.New())
	ps := service.NewProviderService(store.NewProviderRepo(s.GetDB()), reg)
	ps.Init(ctx)
	ds := service.NewDownloadService(dr, de, ue, bus, ps)
	us := service.NewUploadService(dr, ue, bus)
	us.Start()

	d, err := ds.Create(ctx, ts.URL, "detailed.bin", "det-local:/")
	if err != nil {
		t.Fatalf("failed to create download: %v", err)
	}

	// Poll for active progress
	success := false
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		d, _ = dr.Get(ctx, d.ID)
		if d.Status == model.StatusActive && d.Speed > 0 && d.Downloaded > 0 {
			success = true
			break
		}
	}
	if !success {
		t.Error("failed to catch non-zero download progress")
	}

	// Wait for Uploading state
	success = false
	for i := 0; i < 40; i++ {
		time.Sleep(500 * time.Millisecond)
		d, _ = dr.Get(ctx, d.ID)
		if d.Status == model.StatusUploading {
			if d.UploadSpeed > 0 || d.UploadProgress > 0 {
				success = true
				break
			}
		}
		if d.Status == model.StatusComplete {
			success = true
			break
		}
	}
	if !success {
		t.Error("failed to catch uploading state or progress")
	}

	// Wait for final completion
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		d, _ = dr.Get(ctx, d.ID)
		if d.Status == model.StatusComplete {
			break
		}
	}

	if d.Status != model.StatusComplete {
		t.Errorf("expected final status complete, got %s", d.Status)
	}

	if _, err := os.Stat(filepath.Join(uploadDir, "detailed.bin")); os.IsNotExist(err) {
		t.Error("final file not found")
	}
}