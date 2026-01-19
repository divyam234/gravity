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

	// 1. Setup temporary directory for test
	tempDir, err := os.MkdirTemp("", "gravity-real-flow-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	downloadDir := filepath.Join(tempDir, "downloads")
	uploadDir := filepath.Join(tempDir, "uploads")
	os.MkdirAll(downloadDir, 0755)
	os.MkdirAll(uploadDir, 0755)

	// 2. Setup a dummy file server to download from
	fileSize := 10 * 1024 * 1024 // 10MB
	dummyData := make([]byte, fileSize)
	for i := range dummyData {
		dummyData[i] = byte(i % 256)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))
		w.Write(dummyData)
	}))
	defer ts.Close()

	// 3. Initialize components
	s, err := store.New(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	bus := event.NewBus()

	// Engines (using non-conflicting ports)
	de := aria2.NewEngine(16800, "test-secret", tempDir)
	ue := rclone.NewEngine(15572)

	ctx := context.Background()
	if err := de.Start(ctx); err != nil {
		t.Fatalf("failed to start aria2: %v", err)
	}
	defer de.Stop()

	if err := ue.Start(ctx); err != nil {
		t.Fatalf("failed to start rclone: %v", err)
	}
	defer ue.Stop()

	// Configure Rclone Local Remote
	err = ue.CreateRemote(ctx, "test-local", "alias", map[string]string{
		"remote": uploadDir,
	})
	if err != nil {
		t.Fatalf("failed to create rclone remote: %v", err)
	}

	// Services
	registry := provider.NewRegistry()
	registry.Register(direct.New())
	ps := service.NewProviderService(store.NewProviderRepo(s.GetDB()), registry)
	dr := store.NewDownloadRepo(s.GetDB())
	ds := service.NewDownloadService(dr, de, bus, ps)
	us := service.NewUploadService(dr, ue, bus)
	us.Start()

	// 4. Start Download
	t.Log("Starting download...")
	download, err := ds.Create(ctx, ts.URL, "testfile.bin", "test-local:/")
	if err != nil {
		t.Fatalf("failed to create download: %v", err)
	}
	t.Logf("Download created: %s", download.ID)

	// 5. Wait for events
	done := make(chan struct{})
	timeout := time.After(60 * time.Second)
	sub := bus.Subscribe()
	defer bus.Unsubscribe(sub)

	go func() {
		for ev := range sub {
			switch ev.Type {
			case event.DownloadProgress:
				// t.Logf("Download Progress: %v", ev.Data)
			case event.DownloadCompleted:
				t.Log("Download completed!")
			case event.UploadProgress:
				// t.Logf("Upload Progress: %v", ev.Data)
			case event.UploadCompleted:
				t.Log("Upload completed!")
				close(done)
				return
			case event.DownloadError, event.UploadError:
				t.Errorf("Flow error: %v", ev.Data)
				close(done)
				return
			}
		}
	}()

	select {
	case <-done:
		t.Log("End-to-end flow finished successfully")
	case <-timeout:
		t.Fatal("Test timed out waiting for flow completion")
	}

	// 6. Verify file existence in upload dir
	finalPath := filepath.Join(uploadDir, "testfile.bin")
	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		t.Errorf("Uploaded file missing at %s", finalPath)
		} else {
			t.Logf("Verified file at %s", finalPath)
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
	
		// Simulate a multi-file download result
		torrentDir := filepath.Join(downloadDir, "MyTorrent")
		os.MkdirAll(torrentDir, 0755)
		os.WriteFile(filepath.Join(torrentDir, "file1.txt"), []byte("content1"), 0644)
		os.WriteFile(filepath.Join(torrentDir, "file2.txt"), []byte("content2"), 0644)
	
		// Initialize components
		s, _ := store.New(tempDir)
		bus := event.NewBus()
		ue := rclone.NewEngine(15573) // Different port
	
		ctx := context.Background()
		if err := ue.Start(ctx); err != nil {
			t.Fatalf("failed to start rclone: %v", err)
		}
		defer ue.Stop()
	
		err = ue.CreateRemote(ctx, "test-folder-local", "alias", map[string]string{
			"remote": uploadDir,
		})
		if err != nil {
			t.Fatalf("failed to create rclone remote: %v", err)
		}
	
		dr := store.NewDownloadRepo(s.GetDB())
		us := service.NewUploadService(dr, ue, bus)
		us.Start()
	
		// Create a mock download record that is "Completed" and points to the folder
		d := &model.Download{
			ID:          "d_folder_test",
			Status:      model.StatusComplete,
			Filename:    "MyTorrent",
			LocalPath:   torrentDir,
			Destination: "test-folder-local:/",
			UpdatedAt:   time.Now(),
		}
		dr.Create(ctx, d)
	
		// Trigger upload
		t.Log("Triggering folder upload...")
		err = us.TriggerUpload(ctx, d)
		if err != nil {
			t.Fatalf("TriggerUpload failed: %v", err)
		}
	
		// Wait for completion
		done := make(chan struct{})
		sub := bus.Subscribe()
		defer bus.Unsubscribe(sub)
	
		go func() {
			for ev := range sub {
				if ev.Type == event.UploadCompleted {
					t.Log("Upload completed!")
					close(done)
					return
				}
				if ev.Type == event.UploadError {
					t.Errorf("Upload error: %v", ev.Data)
					close(done)
					return
				}
			}
		}()
	
		select {
		case <-done:
			t.Log("Folder flow finished successfully")
		case <-time.After(30 * time.Second):
			t.Fatal("Test timed out")
		}
	
		// Verify existence
		if _, err := os.Stat(filepath.Join(uploadDir, "MyTorrent", "file1.txt")); os.IsNotExist(err) {
			t.Error("Folder upload failed: file1.txt missing")
		}
		if _, err := os.Stat(filepath.Join(uploadDir, "MyTorrent", "file2.txt")); os.IsNotExist(err) {
			t.Error("Folder upload failed: file2.txt missing")
		}
	}
	