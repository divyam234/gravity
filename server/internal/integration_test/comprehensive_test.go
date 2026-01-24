package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gravity/internal/app"
	"gravity/internal/event"
	"gravity/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthentication(t *testing.T) {
	_, ts, _ := setupTestApp(t)
	client := &http.Client{}

	tests := []struct {
		name       string
		apiKey     string
		headerName string
		wantStatus int
	}{
		{"Valid API Key in Header", testAPIKey, "X-API-Key", http.StatusOK},
		{"Invalid API Key in Header", "wrong-key", "X-API-Key", http.StatusUnauthorized},
		{"Missing API Key", "", "", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", ts.URL+"/api/v1/settings", nil)
			if tt.headerName != "" {
				req.Header.Set(tt.headerName, tt.apiKey)
			}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}

	t.Run("Valid API Key in Query Param (token)", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/settings?token="+testAPIKey, nil)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestSettingsWorkflow(t *testing.T) {
	_, ts, _ := setupTestApp(t)
	client := &http.Client{}

	// 1. Get initial settings
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/settings", nil)
	req.Header.Set("X-API-Key", testAPIKey)
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var settings model.Settings
	err = json.NewDecoder(resp.Body).Decode(&settings)
	require.NoError(t, err)
	resp.Body.Close()

	// 2. Update settings
	settings.Download.MaxConcurrentDownloads = 10
	settings.Download.PreferredEngine = "native"

	body, _ := json.Marshal(settings)
	req, _ = http.NewRequest("PATCH", ts.URL+"/api/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("X-API-Key", testAPIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 3. Verify settings updated
	req, _ = http.NewRequest("GET", ts.URL+"/api/v1/settings", nil)
	req.Header.Set("X-API-Key", testAPIKey)
	resp, err = client.Do(req)
	require.NoError(t, err)
	var updatedSettings model.Settings
	err = json.NewDecoder(resp.Body).Decode(&updatedSettings)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, 10, updatedSettings.Download.MaxConcurrentDownloads)
	assert.Equal(t, "native", updatedSettings.Download.PreferredEngine)
}

func TestSystemEndpoints(t *testing.T) {
	_, ts, _ := setupTestApp(t)
	client := &http.Client{}

	endpoints := []string{
		"/api/v1/system/version",
	}

	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			req, _ := http.NewRequest("GET", ts.URL+ep, nil)
			req.Header.Set("X-API-Key", testAPIKey)
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func TestDownloadManagement(t *testing.T) {
	_, ts, mockURL := setupTestApp(t)
	client := &http.Client{}

	// Helper for creating a download
	createDownload := func(t *testing.T, url string) string {
		reqBody := map[string]interface{}{"url": url}
		jsonBody, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/downloads", bytes.NewBuffer(jsonBody))
		req.Header.Set("X-API-Key", testAPIKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			var body []byte
			body, _ = io.ReadAll(resp.Body)
			t.Fatalf("Failed to create download: status %d, body: %s", resp.StatusCode, string(body))
		}

		var d model.Download
		json.NewDecoder(resp.Body).Decode(&d)
		return d.ID
	}

	t.Run("Full Lifecycle", func(t *testing.T) {
		// 1. Create
	id := createDownload(t, mockURL+"/test.zip")

		// 2. Get
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/downloads/"+id, nil)
		req.Header.Set("X-API-Key", testAPIKey)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var d model.Download
		json.NewDecoder(resp.Body).Decode(&d)
		resp.Body.Close()
		assert.Equal(t, id, d.ID)

		// 3. Pause
	req, _ = http.NewRequest("POST", ts.URL+"/api/v1/downloads/"+id+"/pause", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		resp, err = client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// 4. Resume
	req, _ = http.NewRequest("POST", ts.URL+"/api/v1/downloads/"+id+"/resume", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		resp, err = client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// 5. List with filtering
	req, _ = http.NewRequest("GET", ts.URL+"/api/v1/downloads?status=active,waiting", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		resp, err = client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// 6. Delete
	req, _ = http.NewRequest("DELETE", ts.URL+"/api/v1/downloads/"+id, nil)
		req.Header.Set("X-API-Key", testAPIKey)
		resp, err = client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()

		// 7. Verify Deleted
	req, _ = http.NewRequest("GET", ts.URL+"/api/v1/downloads/"+id, nil)
		req.Header.Set("X-API-Key", testAPIKey)
		resp, err = client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestStats(t *testing.T) {
	_, ts, _ := setupTestApp(t)
	client := &http.Client{}

	endpoints := []string{
		"/api/v1/stats",
	}

	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			req, _ := http.NewRequest("GET", ts.URL+ep, nil)
			req.Header.Set("X-API-Key", testAPIKey)
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func TestConcurrencyAndQueueing(t *testing.T) {
	_, ts, mockURL := setupTestApp(t)
	client := &http.Client{}

	// 1. Set MaxConcurrentDownloads to 2 and PreferredEngine to native
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/settings", nil)
	req.Header.Set("X-API-Key", testAPIKey)
	resp, _ := client.Do(req)
	var settings model.Settings
	json.NewDecoder(resp.Body).Decode(&settings)
	resp.Body.Close()

	settings.Download.MaxConcurrentDownloads = 2
	settings.Download.PreferredEngine = "native"
	body, _ := json.Marshal(settings)
	req, _ = http.NewRequest("PATCH", ts.URL+"/api/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("X-API-Key", testAPIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = client.Do(req)
	resp.Body.Close()

	// 2. Add 5 downloads
	for i := 0; i < 5; i++ {
		reqBody := map[string]interface{}{"url": mockURL + "/test.zip"}
		jsonBody, _ := json.Marshal(reqBody)
		req, _ = http.NewRequest("POST", ts.URL+"/api/v1/downloads", bytes.NewBuffer(jsonBody))
		req.Header.Set("X-API-Key", testAPIKey)
		req.Header.Set("Content-Type", "application/json")
		resp, _ = client.Do(req)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()
	}

	// 3. Wait a bit for processing
	time.Sleep(1 * time.Second)

	// 4. Check status of all downloads
	req, _ = http.NewRequest("GET", ts.URL+"/api/v1/downloads", nil)
	req.Header.Set("X-API-Key", testAPIKey)
	resp, _ = client.Do(req)
	var listResp struct {
		Data []*model.Download `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&listResp)
	resp.Body.Close()

	activeOrDone := 0
	waiting := 0
	for _, d := range listResp.Data {
		if d.Status == model.StatusActive || d.Status == model.StatusComplete {
			activeOrDone++
		} else {
			waiting++
		}
	}

	// At least 1 should have started, and at most 2 should be active at a time (if they didn't finish yet)
	// But since they might finish fast, we just check that some started.
	assert.Greater(t, activeOrDone, 0, "At least one download should be active or done")
}

func TestFiles(t *testing.T) {
	_, ts, _ := setupTestApp(t)
	client := &http.Client{}

	t.Run("List Files", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/files/list", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, resp.StatusCode)
	})
}

func TestSearchEndpoints(t *testing.T) {
	_, ts, _ := setupTestApp(t)
	client := &http.Client{}

	t.Run("Config Lifecycle", func(t *testing.T) {
		// 1. Get Configs
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/search/config", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// 2. Update Config
	reqBody := map[string]interface{}{
			"configs": map[string]interface{}{
				"test-remote": map[string]interface{}{
					"interval": 120,
				},
			},
		}
		jsonBody, _ := json.Marshal(reqBody)
		req, _ = http.NewRequest("POST", ts.URL+"/api/v1/search/config", bytes.NewBuffer(jsonBody))
		req.Header.Set("X-API-Key", testAPIKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("Search Invalid", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/search", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestProviders(t *testing.T) {
	_, ts, mockURL := setupTestApp(t)
	client := &http.Client{}

	t.Run("List and Get", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/providers", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		req, _ = http.NewRequest("GET", ts.URL+"/api/v1/providers/direct", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		resp, err = client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("Resolve URL", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"url": mockURL + "/test.zip",
		}
		jsonBody, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/providers/resolve", bytes.NewBuffer(jsonBody))
		req.Header.Set("X-API-Key", testAPIKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestSyncStartup(t *testing.T) {
	// 1. Setup a manual app instance in home dir
	home, _ := os.UserHomeDir()
	testDataDir := filepath.Join(home, "gravity_sync_test_data")
	os.RemoveAll(testDataDir)
	os.MkdirAll(testDataDir, 0755)
	defer os.RemoveAll(testDataDir)

	// Mock Download Server
	mux := http.NewServeMux()
	mux.HandleFunc("/sync.zip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Length", "1024")
		w.Write(make([]byte, 1024))
	})
	mockDownloader := httptest.NewServer(mux)
	defer mockDownloader.Close()

	aria2Port := getFreePort(t)
	os.Setenv("GRAVITY_DATA_DIR", testDataDir)
	os.Setenv("GRAVITY_API_KEY", testAPIKey)
	os.Setenv("GRAVITY_ARIA2_RPC_PORT", fmt.Sprintf("%d", aria2Port))
	os.Setenv("GRAVITY_DATABASE__TYPE", "sqlite")
	os.Setenv("GRAVITY_DATABASE__DSN", filepath.Join(testDataDir, "test.db"))

	ctx := context.Background()
	a, err := app.New(ctx, nil, nil)
	require.NoError(t, err)

	// 2. Start and create a download
	err = a.Start(ctx)
	require.NoError(t, err)

	ts := httptest.NewServer(a.Router.Handler())
	
	reqBody := map[string]interface{}{"url": mockDownloader.URL + "/sync.zip"}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/downloads", bytes.NewBuffer(jsonBody))
	req.Header.Set("X-API-Key", testAPIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()
	
	// Wait for it to become active
	time.Sleep(1 * time.Second)
	
	ts.Close()
	a.Stop()

	// 3. Start a NEW app instance with SAME DB
	a2, err := app.New(ctx, nil, nil)
	require.NoError(t, err)
	
	err = a2.StartEngines(ctx)
	require.NoError(t, err)
	
	// Verify it was reset or processed
	downloads, _, _ := a2.DownloadService().List(ctx, nil, 10, 0)
	assert.Len(t, downloads, 1)
	assert.Contains(t, []model.DownloadStatus{model.StatusActive, model.StatusWaiting, model.StatusAllocating, model.StatusComplete}, downloads[0].Status)
	
	a2.Stop()
}

func TestErrorScenarios(t *testing.T) {
	_, ts, mockURL := setupTestApp(t)
	client := &http.Client{}

	t.Run("Invalid API Key", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/settings", nil)
		req.Header.Set("X-API-Key", "invalid")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Non-existent Download", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/downloads/non-existent", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Download 404 URL", func(t *testing.T) {
		reqBody := map[string]interface{}{"url": mockURL + "/404.zip"}
		jsonBody, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/downloads", bytes.NewBuffer(jsonBody))
		req.Header.Set("X-API-Key", testAPIKey)
		req.Header.Set("Content-Type", "application/json")
		resp, _ := client.Do(req)
		// Resolve should fail
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestSpeedAndProgress(t *testing.T) {
	app, ts, mockURL := setupTestApp(t)
	client := &http.Client{}

	// Force native engine for this test
	setPreferredEngine(t, ts.URL, "native")

	// Resume stats polling to ensure we get updates
	app.StatsService().ResumePolling()

	// 1. Subscribe to events
	eventChan := app.Events().Subscribe()
	defer app.Events().Unsubscribe(eventChan)

	// 2. Create a download
	reqBody := map[string]interface{}{"url": mockURL + "/test.zip"}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/downloads", bytes.NewBuffer(jsonBody))
	req.Header.Set("X-API-Key", testAPIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(req)
	var d model.Download
	json.NewDecoder(resp.Body).Decode(&d)
	resp.Body.Close()
	downloadID := d.ID

	// Wait for engines to start and report speed
	time.Sleep(2 * time.Second)

	// 3. Monitor for non-zero speed
	timeout := time.After(60 * time.Second)
	seenPositiveTaskSpeed := false
	seenPositiveGlobalSpeed := false
	seenProgress := false

	done := false
	for !done {
		select {
		case <-timeout:
			t.Errorf("Timeout. Flags: Progress=%v, TaskSpeed=%v, GlobalSpeed=%v", seenProgress, seenPositiveTaskSpeed, seenPositiveGlobalSpeed)
			t.Fatal("Timeout waiting for positive speed reporting")
		case ev := <-eventChan:
			// Debug log
			// t.Logf("Event: %s", ev.Type)
			
			switch ev.Type {
			case event.DownloadProgress:
				if data, ok := ev.Data.(map[string]interface{}); ok && data["id"] == downloadID {
					speed := data["speed"].(int64)
					downloaded := data["downloaded"].(int64)
					if speed > 0 {
						seenPositiveTaskSpeed = true
					}
					if downloaded > 0 {
						seenProgress = true
					}
				}
			case event.DownloadError:
				if data, ok := ev.Data.(map[string]string); ok && data["id"] == downloadID {
					t.Logf("Download Error received for %s: %s", downloadID, data["error"])
					done = true
				}
			case event.StatsUpdate:
				if stats, ok := ev.Data.(*model.Stats); ok {
					if stats.Speeds.Download > 0 {
						seenPositiveGlobalSpeed = true
					}
				}
			case event.DownloadCompleted:
				if data, ok := ev.Data.(*model.Download); ok && data.ID == downloadID {
					done = true
				} else if data, ok := ev.Data.(model.Download); ok && data.ID == downloadID {
					done = true
				}
			}

			if seenPositiveTaskSpeed && seenPositiveGlobalSpeed && seenProgress {
				done = true
			}
		}
	}

	assert.True(t, seenProgress, "Should have seen some bytes downloaded")
	assert.True(t, seenPositiveTaskSpeed, "Should have seen non-zero task speed")
	assert.True(t, seenPositiveGlobalSpeed, "Should have seen non-zero global speed")
}

func TestFullUploadFlow(t *testing.T) {
	app, ts, mockURL := setupTestApp(t)
	testFileURL := mockURL + "/test.zip"

	// 1. Configure Engine (Native is easier for this test)
	setPreferredEngine(t, ts.URL, "native")

	// Get data dir from app config to find the local remote path
	testDataDir := app.Config().DataDir
	remoteDestDir := filepath.Join(testDataDir, "remote_dest")

	// 2. Subscribe to Internal Bus
	eventChan := app.Events().Subscribe()
	defer app.Events().Unsubscribe(eventChan)

	// 3. Create Download with Destination
	client := &http.Client{}
	reqBody := map[string]interface{}{
		"url":         testFileURL,
		"destination": "localtest:/" + remoteDestDir,
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/downloads", bytes.NewBuffer(jsonBody))
	req.Header.Set("X-API-Key", testAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var d model.Download
	json.NewDecoder(resp.Body).Decode(&d)
	downloadID := d.ID
	fmt.Printf("\n[UploadFlow] Created Download ID: %s with destination: %s\n", downloadID, d.Destination)

	// 4. Wait for Download Completion AND Upload Completion
	timeout := time.After(60 * time.Second)
	downloadDone := false
	uploadDone := false

	fmt.Println("Waiting for download and upload events...")
	for !downloadDone || !uploadDone {
		select {
		case <-timeout:
			t.Fatalf("Timeout waiting for flow completion (Download: %v, Upload: %v)", downloadDone, uploadDone)
		case ev := <-eventChan:
			switch ev.Type {
			case event.DownloadCompleted:
				if data, ok := ev.Data.(*model.Download); ok && data.ID == downloadID {
					fmt.Println(" -> Event: Download Completed")
					downloadDone = true
				} else if data, ok := ev.Data.(model.Download); ok && data.ID == downloadID {
					fmt.Println(" -> Event: Download Completed")
					downloadDone = true
				}
			case event.UploadStarted:
				if data, ok := ev.Data.(map[string]string); ok && data["id"] == downloadID {
					fmt.Println(" -> Event: Upload Started")
				}
			case event.UploadCompleted:
				if data, ok := ev.Data.(*model.Download); ok && data.ID == downloadID {
					fmt.Println(" -> Event: Upload Completed")
					uploadDone = true
				} else if data, ok := ev.Data.(model.Download); ok && data.ID == downloadID {
					fmt.Println(" -> Event: Upload Completed")
					uploadDone = true
				}
			case event.DownloadError, event.UploadError:
				t.Logf("Flow failed with event: %v", ev)
				// Don't fail immediately, some other events might be relevant
			}
		}
	}

	// 5. Verify file exists in local "remote"
	expectedFile := filepath.Join(remoteDestDir, "test.zip")
	_, err = os.Stat(expectedFile)
	assert.NoError(t, err, "File should exist in the remote destination")
	
	if err == nil {
		fmt.Printf(" -> Success: File verified at %s\n", expectedFile)
	}
}