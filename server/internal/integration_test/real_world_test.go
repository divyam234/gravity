package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"gravity/internal/event"
	"gravity/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLargeFileDownload tests a real-world 1GB download to ensure stability
// Note: This test requires internet access and will download ~1.1GB.
func TestLargeFileDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file download in short mode")
	}

	app, ts, _ := setupTestApp(t)
	targetURL := "https://edgedl.me.gvt1.com/android/studio/ide-zips/2025.2.3.9/android-studio-2025.2.3.9-linux.tar.gz"

	// 1. Configure Engine (Native for speed and Rclone stability check)
	setPreferredEngine(t, ts.URL, "native")

	// 2. Subscribe to events
	eventChan := app.Events().Subscribe()
	defer app.Events().Unsubscribe(eventChan)

	// 3. Create Download
	client := &http.Client{}
	reqBody := map[string]interface{}{
		"url": targetURL,
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
	fmt.Printf("\n[LargeTest] Started download %s for %s\n", downloadID, targetURL)

	// 4. Monitor until completion or timeout (long timeout for 1GB)
	timeout := time.After(15 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	done := false
	lastProgress := 0.0

	fmt.Println("Monitoring progress...")
	for !done {
		select {
		case <-timeout:
			t.Fatalf("Timeout waiting for 1GB download to complete")
		case <-ticker.C:
			// Just a heartbeat
		case ev := <-eventChan:
			switch ev.Type {
			case event.DownloadProgress:
				if data, ok := ev.Data.(map[string]interface{}); ok && data["id"] == downloadID {
					downloaded := data["downloaded"].(int64)
					total := data["size"].(int64)
					if total > 0 {
						p := (float64(downloaded) / float64(total)) * 100
						if p > lastProgress + 5 || p >= 99.9 {
							fmt.Printf(" -> Progress: %.2f%% (%d / %d MB)\n", p, downloaded/1024/1024, total/1024/1024)
							lastProgress = p
						}
					}
				}
			case event.DownloadCompleted:
				if data, ok := ev.Data.(*model.Download); ok && data.ID == downloadID {
					fmt.Println(" -> Event: Download Completed Successfully!")
					done = true
				} else if data, ok := ev.Data.(model.Download); ok && data.ID == downloadID {
					fmt.Println(" -> Event: Download Completed Successfully!")
					done = true
				}
			case event.DownloadError:
				if data, ok := ev.Data.(map[string]string); ok && data["id"] == downloadID {
					t.Fatalf("Download failed with error: %s", data["error"])
				}
			}
		}
	}

	assert.True(t, done, "Download should have completed")
}
