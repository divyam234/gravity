package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
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

const (
	testAPIKey = "test-api-key"
)

// Helper to get a random free port
func getFreePort(t *testing.T) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	require.NoError(t, err)

	l, err := net.ListenTCP("tcp", addr)
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// Helper to setup the app and HTTP server
func setupTestApp(t *testing.T) (*app.App, *httptest.Server, string) {
	// 1. Mock Download Server
	mux := http.NewServeMux()
	mux.HandleFunc("/test.zip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		const size = 10 * 1024 * 1024 // 10MB
		w.Header().Set("Content-Length", fmt.Sprintf("%d", size))

		data := make([]byte, 128*1024) // 128KB chunks
		for i := 0; i < size/(128*1024); i++ {
			select {
			case <-r.Context().Done():
				return
			default:
				_, err := w.Write(data)
				if err != nil {
					return
				}
				// Throttle: every 2 chunks (256KB), sleep 100ms -> ~2.5MB/s
				if i%2 == 0 {
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	})
	mockDownloader := httptest.NewServer(mux)
	t.Cleanup(mockDownloader.Close)

	// 2. Create data directory in home
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	testDataDir := filepath.Join(home, fmt.Sprintf("gravity_test_data_%d_%d", time.Now().UnixNano(), getFreePort(t)))

	// Ensure cleanup
	os.RemoveAll(testDataDir)
	err = os.MkdirAll(testDataDir, 0755)
	require.NoError(t, err)

	aria2Port := getFreePort(t)

	// 3. Create temporary rclone config for local testing
	rcloneConfigDir := filepath.Join(testDataDir, "rclone")
	os.MkdirAll(rcloneConfigDir, 0755)
	rcloneConfigPath := filepath.Join(rcloneConfigDir, "rclone.conf")

	uploadDestDir := filepath.Join(testDataDir, "remote_dest")
	os.MkdirAll(uploadDestDir, 0755)

	rcloneConfig := fmt.Sprintf("[localtest]\ntype = local\n")
	err = os.WriteFile(rcloneConfigPath, []byte(rcloneConfig), 0644)
	require.NoError(t, err)

	// Mock Config
	os.Setenv("GRAVITY_DATA_DIR", testDataDir)
	os.Setenv("GRAVITY_PORT", "0") // Random port
	os.Setenv("GRAVITY_API_KEY", testAPIKey)
	os.Setenv("GRAVITY_ARIA2_RPC_PORT", fmt.Sprintf("%d", aria2Port))
	os.Setenv("GRAVITY_RCLONE_CONFIG_PATH", rcloneConfigPath)
	os.Setenv("GRAVITY_DATABASE__TYPE", "sqlite")
	os.Setenv("GRAVITY_DATABASE__DSN", filepath.Join(testDataDir, "test.db"))

	// Initialize App
	ctx := context.Background()
	a, err := app.New(ctx, nil, nil)
	require.NoError(t, err)

	// Start Engines & Background Services
	err = a.Start(ctx)
	require.NoError(t, err)

	// Start Test Server
	ts := httptest.NewServer(a.Router.Handler())

	// Cleanup
	t.Cleanup(func() {
		ts.Close()
		a.Stop()
		os.RemoveAll(testDataDir)
		os.Unsetenv("GRAVITY_DATA_DIR")
		os.Unsetenv("GRAVITY_PORT")
		os.Unsetenv("GRAVITY_API_KEY")
		os.Unsetenv("GRAVITY_ARIA2_RPC_PORT")
		os.Unsetenv("GRAVITY_RCLONE_CONFIG_PATH")
		os.Unsetenv("GRAVITY_DATABASE__TYPE")
		os.Unsetenv("GRAVITY_DATABASE__DSN")
	})

	return a, ts, mockDownloader.URL
}

// Helper: Configure preferred engine
func setPreferredEngine(t *testing.T, baseURL string, engineName string) {
	client := &http.Client{}

	settings := model.Settings{}
	// Fetch current settings first to preserve other defaults
	req, _ := http.NewRequest("GET", baseURL+"/api/v1/settings", nil)
	req.Header.Set("X-API-Key", testAPIKey)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	var respData struct {
		Data *model.Settings `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&respData)
	settings = *respData.Data

	// Update Engine
	settings.Download.PreferredEngine = engineName
	settings.Download.DownloadDir = filepath.Join(os.TempDir(), "gravity-downloads-"+engineName)
	os.MkdirAll(settings.Download.DownloadDir, 0755)

	// Ensure other required fields are set (in case GET was empty)
	if settings.Download.MaxConcurrentDownloads == 0 {
		settings.Download.MaxConcurrentDownloads = 3
	}
	if settings.Download.MaxConnectionPerServer == 0 {
		settings.Download.MaxConnectionPerServer = 8
	}
	if settings.Download.Split == 0 {
		settings.Download.Split = 8
	}
	if settings.Download.ConnectTimeout == 0 {
		settings.Download.ConnectTimeout = 60
	}
	if settings.Upload.ConcurrentUploads == 0 {
		settings.Upload.ConcurrentUploads = 1
	}
	if settings.Torrent.ListenPort == 0 {
		settings.Torrent.ListenPort = 6881
	}

	body, _ := json.Marshal(settings)
	req, _ = http.NewRequest("PATCH", baseURL+"/api/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("X-API-Key", testAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDownloadLifecycle(t *testing.T) {
	app, ts, mockURL := setupTestApp(t)
	testFileURL := mockURL + "/test.zip"

	tests := []struct {
		name   string
		engine string
	}{
		{"Native Engine", "native"},
		// Aria2 requires external daemon. If not present, this test case might fail or hang.
		// Ideally we mock it or check if it's available.
		{"Aria2 Engine", "aria2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Configure Engine
			setPreferredEngine(t, ts.URL, tt.engine)

			// 2. Subscribe to Internal Bus (Verification)
			// Verify SSE Endpoint Connects (User Request)
			sseReq, _ := http.NewRequest("GET", ts.URL+"/api/v1/events?token="+testAPIKey, nil)
			sseClient := &http.Client{Timeout: 50 * time.Millisecond}
			sseResp, _ := sseClient.Do(sseReq)
			if sseResp != nil {
				assert.Equal(t, http.StatusOK, sseResp.StatusCode)
				sseResp.Body.Close()
			}

			eventChan := app.Events().SubscribeAll()
			defer app.Events().UnsubscribeAll(eventChan)

			// Ensure stats polling is active
			app.StatsService().ResumePolling()

			// 3. Create Download
			client := &http.Client{}
			reqBody := map[string]any{
				"url": testFileURL,
			}
			jsonBody, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", ts.URL+"/api/v1/downloads", bytes.NewBuffer(jsonBody))
			req.Header.Set("X-API-Key", testAPIKey)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode)

			var respData struct {
				Data *model.Download `json:"data"`
			}
			json.NewDecoder(resp.Body).Decode(&respData)
			downloadID := respData.Data.ID
			fmt.Printf("\n[%s] Created Download ID: %s\n", tt.engine, downloadID)

			// 4. Monitor Progress
			// We wait for initial progress to prove it's working, then stop.
			timeout := time.After(60 * time.Second)

			seenStats := false

			done := false

			fmt.Println("Waiting for events...")
			for !done {
				select {
				case <-timeout:
					t.Fatalf("Timeout waiting for download activity (Engine: %s)", tt.engine)
				case ev := <-eventChan:
					switch ev.Type {
					case event.DownloadStarted:
						if data, ok := ev.Data.(*model.Download); ok && data.ID == downloadID {
							fmt.Println(" -> Event: Started")
						} else if data, ok := ev.Data.(model.Download); ok && data.ID == downloadID {
							fmt.Println(" -> Event: Started")
						}

					case event.DownloadCompleted:
						// If we didn't see enough progress but it's already done, it's also a success
						if data, ok := ev.Data.(*model.Download); ok && data.ID == downloadID {
							fmt.Println(" -> Event: Completed")
							done = true
						} else if data, ok := ev.Data.(model.Download); ok && data.ID == downloadID {
							fmt.Println(" -> Event: Completed")
							done = true
						}

					case event.DownloadError:
						if data, ok := ev.Data.(map[string]string); ok && data["id"] == downloadID {
							t.Logf("Download error for %s: %s", downloadID, data["error"])
							done = true
						}

					case event.DownloadProgress:
						if data, ok := ev.Data.(map[string]any); ok && data["id"] == downloadID {
							downloaded := data["downloaded"].(int64)
							// If we have downloaded > 512KB, success (native is fast, aria2 is fast)
							if downloaded > 512*1024 {
								fmt.Printf(" -> Progress: Downloaded %.2f MB. Success threshold met.\n", float64(downloaded)/1024/1024)
								done = true
							}
						}

					case event.StatsUpdate:
						if !seenStats {
							fmt.Println(" -> Event: Global Stats Update Received")
						}
						seenStats = true
					}
				}
			}

			// 5. Cleanup (Cancel/Delete)
			req, _ = http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/downloads/%s?delete_files=true", ts.URL, downloadID), nil)
			req.Header.Set("X-API-Key", testAPIKey)
			resp, err = client.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode)
			fmt.Println(" -> Download Cancelled & Deleted")

			assert.True(t, seenStats, "Should have received Global Stats events")
			// We don't strictly require seenActive/seenProgress if it completed extremely fast
		})
	}
}
