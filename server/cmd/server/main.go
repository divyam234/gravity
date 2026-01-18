package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"aria2-rclone-ui/internal/api"
	"aria2-rclone-ui/internal/aria2"
	"aria2-rclone-ui/internal/poller"
	"aria2-rclone-ui/internal/rclone"
	"aria2-rclone-ui/internal/store"
	"aria2-rclone-ui/internal/utils"
)

const (
	Aria2RPCPort = 6800
	RcloneRCPort = 5572
	ServerPort   = 8080
)

func generateSecret() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "aria2-secret-fallback"
	}
	return hex.EncodeToString(b)
}

func main() {
	// 0. Initialize Configuration
	aria2Secret := os.Getenv("ARIA2_SECRET")
	if aria2Secret == "" {
		aria2Secret = generateSecret()
	}
	log.Printf("Using Aria2 Secret: %s", aria2Secret)

	// 1. Initialize DB
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".aria2-rclone")
	os.MkdirAll(dataDir, 0755)

	db, err := store.New(filepath.Join(dataDir, store.DbName))
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// 1. Initialize Runners
	aria2Runner := aria2.NewRunner(Aria2RPCPort, aria2Secret)
	rcloneRunner := rclone.NewRunner(fmt.Sprintf("localhost:%d", RcloneRCPort))

	// 2. Start Processes
	log.Println("Starting Aria2...")
	if err := aria2Runner.Start(); err != nil {
		log.Fatalf("Failed to start Aria2: %v", err)
	}
	defer aria2Runner.Stop()

	log.Println("Starting Rclone...")
	if err := rcloneRunner.Start(); err != nil {
		log.Printf("Warning: Failed to start Rclone (Auto-upload disabled): %v", err)
	} else {
		defer rcloneRunner.Stop()
	}

	// 3. Setup HTTP Server
	mux := http.NewServeMux()

	// Unified RPC Handler
	rpcHandler := api.NewRPCHandler(
		fmt.Sprintf("http://localhost:%d", Aria2RPCPort),
		fmt.Sprintf("http://localhost:%d", RcloneRCPort),
		aria2Secret,
		db,
	)

	mux.Handle("/jsonrpc", rpcHandler)

	// Serve Frontend
	fs := http.FileServer(http.Dir("./dist"))
	mux.Handle("/", fs)

	// 4. Start Server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", ServerPort),
		Handler: mux,
	}

	go func() {
		log.Printf("Server listening on http://localhost:%d", ServerPort)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Create cancellable context for background tasks
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()

	// 5. Restore Session & Connect to Aria2 WebSocket
	go func() {
		time.Sleep(2 * time.Second) // Wait for Aria2 to fully start

		aria2Client := aria2.NewClient(fmt.Sprintf("ws://localhost:%d/jsonrpc", Aria2RPCPort), aria2Secret)
		rcloneClient := rclone.NewClient(fmt.Sprintf("http://localhost:%d", RcloneRCPort))

		// --- Background Monitor (Watchdog) ---
		poller := poller.New(db, aria2Client, rcloneClient)
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-bgCtx.Done():
					return
				case <-ticker.C:
					poller.Sync()
				}
			}
		}()

		// --- Restoration Logic ---
		// 1. Reset "uploading" tasks to "pending" (assume interrupted)
		if err := db.ResetStuckUploads(); err != nil {
			log.Printf("Failed to reset stuck uploads: %v", err)
		}

		pending, err := db.GetPendingUploads()
		if err != nil {
			log.Printf("Failed to fetch pending uploads: %v", err)
		} else {
			log.Printf("Checking %d pending tasks for restoration...", len(pending))
			for _, task := range pending {
				_, err := aria2Client.TellStatus(task.Gid)
				if err != nil {
					// Likely 404/Error -> Task missing from Aria2
					log.Printf("Restoring task %s (Remote: %s)", task.Gid, task.TargetRemote)

					// Re-submit
					// We need to ensure GID is passed in options
					opts := task.Options
					if opts == nil {
						opts = make(map[string]interface{})
					}
					opts["gid"] = task.Gid

					// Aria2 Client Call needs specific args.
					// Our simple client helper might need raw 'addUri' support or we use Call() directly
					var method string
					var params []interface{}

					if task.Torrent != "" {
						method = "aria2.addTorrent"
						params = []interface{}{task.Torrent, task.URIs, opts}
					} else if task.Metalink != "" {
						method = "aria2.addMetalink"
						params = []interface{}{task.Metalink, opts}
					} else {
						method = "aria2.addUri"
						params = []interface{}{task.URIs, opts}
					}

					// We use variadic params... expansion
					_, err := aria2Client.Call(method, params...)
					if err != nil {
						log.Printf("Failed to restore %s: %v", task.Gid, err)
						db.IncrementRetry(task.Gid)
					}
				} else {
					log.Printf("Task %s is already active.", task.Gid)
				}
			}
		}

		aria2Client.SetOnCompleteHandler(func(gid string) {
			// Check if this task has an Rclone target in DB
			upload, err := db.GetUpload(gid)
			if err != nil {
				// Not a managed upload, ignore
				return
			}

			if upload.Status != "pending" {
				return
			}

			log.Printf("Task %s completed. Starting upload to %s...", gid, upload.TargetRemote)

			// Get File Path
			status, err := aria2Client.TellStatus(gid)
			if err != nil {
				log.Printf("Failed to get status for %s: %v", gid, err)
				return
			}

			// Record download stats
			if totalLength, ok := status["totalLength"].(string); ok {
				if bytes, err := strconv.ParseInt(totalLength, 10, 64); err == nil && bytes > 0 {
					db.AddDownloadedBytes(bytes)
					db.IncrementCompletedTasks()
				}
			}

			// Magnet Handoff: Check for followedBy
			if followedBy, ok := status["followedBy"].([]interface{}); ok && len(followedBy) > 0 {
				for _, child := range followedBy {
					if childGid, ok := child.(string); ok {
						// Check if we already track it
						if _, err := db.GetUpload(childGid); err != nil {
							log.Printf("Magnet Handoff (WS): Adopting child GID %s", childGid)
							db.SaveUpload(store.UploadState{
								Gid:          childGid,
								TargetRemote: upload.TargetRemote,
								Status:       "pending",
								StartedAt:    time.Now(),
								Options:      upload.Options,
							})
						}
					}
				}
			}

			files, ok := status["files"].([]interface{})
			if !ok || len(files) == 0 {
				log.Printf("No files found for %s", gid)
				return
			}
			file0, ok := files[0].(map[string]interface{})
			if !ok {
				log.Printf("Invalid file structure for %s", gid)
				return
			}
			path, ok := file0["path"].(string)
			if !ok || path == "" {
				log.Printf("No file path for %s", gid)
				return
			}

			// Trigger Rclone
			// For local files, srcFs should be the directory, srcRemote the filename
			dir := filepath.Dir(path)
			filename := filepath.Base(path)
			remotePath := filename // Destination path relative to remote root

			// Async Upload
			db.UpdateStatus(gid, "uploading", "")
			customJobId := utils.HexGidToInt64(gid)
			jobId, err := rcloneClient.CopyFileAsync(dir, filename, upload.TargetRemote, remotePath, customJobId)
			if err != nil {
				log.Printf("Failed to start async upload for %s: %v", gid, err)
				db.UpdateStatus(gid, "error", "")
			} else {
				log.Printf("Started async upload for %s (Job: %s)", gid, jobId)
				db.UpdateJob(gid, jobId)
			}
		})

		log.Println("Connecting to Aria2 WebSocket...")
		if err := aria2Client.Listen(bgCtx); err != nil {
			if bgCtx.Err() == nil {
				log.Printf("Aria2 Listener stopped: %v", err)
			}
		}
	}()

	// 6. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
