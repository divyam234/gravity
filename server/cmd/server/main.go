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
	"strings"
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

	// Allow overriding download dir for tests
	if dlDir := os.Getenv("ARIA2_DIR"); dlDir != "" {
		aria2Runner = aria2.NewCustomRunner(Aria2RPCPort, aria2Secret, dlDir)
	}

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
		aria2Client := aria2.NewClient(fmt.Sprintf("ws://localhost:%d/jsonrpc", Aria2RPCPort), aria2Secret)
		rcloneClient := rclone.NewClient(fmt.Sprintf("http://localhost:%d", RcloneRCPort))

		// Wait for Aria2 to be ready
		log.Println("Waiting for Aria2 RPC to be ready...")
		waitCtx, waitCancel := context.WithTimeout(bgCtx, 10*time.Second)
		defer waitCancel()
		if err := waitForAria2(waitCtx, aria2Client); err != nil {
			log.Printf("Aria2 failed to become ready: %v", err)
			return
		}
		log.Println("Aria2 RPC is ready.")

		// --- Startup Synchronization (Anti-Drift) ---
		// Import all active and stopped tasks from Aria2 into BoltDB on startup
		// This ensures BoltDB is the true source of truth, even after restart
		if err := importActiveTasks(aria2Client, db); err != nil {
			log.Printf("Failed to import Aria2 tasks on startup: %v", err)
		}

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

		// --- WebSocket Event Handlers for Real-Time Sync ---

		// onDownloadStart: Shadow Import for tasks added externally (CLI, other UIs)
		aria2Client.SetOnStartHandler(func(gid string) {
			// Check if we already have this task in DB
			if _, err := db.GetUpload(gid); err == nil {
				// Already tracking, just update status
				log.Printf("Task %s started (already tracked)", gid)
				db.UpdateStatus(gid, "active", "")
				return
			}

			// New task - fetch details and create shadow record
			log.Printf("Task %s started (shadow import)", gid)
			status, err := aria2Client.TellStatus(gid)
			if err != nil {
				log.Printf("Failed to fetch status for new task %s: %v", gid, err)
				return
			}

			filePath := ""
			totalLengthInt := int64(0)
			ariaStatus, _ := status["status"].(string)

			if files, ok := status["files"].([]interface{}); ok && len(files) > 0 {
				if file0, ok := files[0].(map[string]interface{}); ok {
					filePath, _ = file0["path"].(string)
				}
			}

			if totalLength, ok := status["totalLength"].(string); ok {
				totalLengthInt, _ = strconv.ParseInt(totalLength, 10, 64)
			}

			if err := db.UpsertTask(store.UploadState{
				Gid:         gid,
				Status:      ariaStatus,
				FilePath:    filePath,
				TotalLength: totalLengthInt,
				StartedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}); err != nil {
				log.Printf("Failed to shadow import task %s: %v", gid, err)
			}
		})

		// onDownloadPause: Update DB status and progress
		aria2Client.SetOnPauseHandler(func(gid string) {
			log.Printf("Task %s paused via WebSocket", gid)
			if status, err := aria2Client.TellStatus(gid); err == nil {
				cLen, _ := strconv.ParseInt(status["completedLength"].(string), 10, 64)
				tLen, _ := strconv.ParseInt(status["totalLength"].(string), 10, 64)
				ariaStatus, _ := status["status"].(string)

				db.UpdateProgress(gid, cLen, tLen)
				db.UpdateStatus(gid, ariaStatus, "")
			} else {
				db.UpdateStatus(gid, "paused", "")
			}
		})

		// onDownloadStop: Update DB status and progress
		aria2Client.SetOnStopHandler(func(gid string) {
			log.Printf("Task %s stopped via WebSocket", gid)
			if status, err := aria2Client.TellStatus(gid); err == nil {
				cLen, _ := strconv.ParseInt(status["completedLength"].(string), 10, 64)
				tLen, _ := strconv.ParseInt(status["totalLength"].(string), 10, 64)
				ariaStatus, _ := status["status"].(string)

				db.UpdateProgress(gid, cLen, tLen)
				db.UpdateStatus(gid, ariaStatus, "")
			} else {
				// Task removed from Aria2 memory entirely
				log.Printf("Task %s removed externally (not found in memory)", gid)
				db.UpdateStatus(gid, "removed", "")
			}
		})

		// onDownloadError: Update DB with error message and progress
		aria2Client.SetOnErrorHandler(func(gid string) {
			log.Printf("Task %s errored via WebSocket", gid)
			status, err := aria2Client.TellStatus(gid)
			errorMessage := "Unknown error"

			if err == nil {
				cLen, _ := strconv.ParseInt(status["completedLength"].(string), 10, 64)
				tLen, _ := strconv.ParseInt(status["totalLength"].(string), 10, 64)
				db.UpdateProgress(gid, cLen, tLen)

				if code, ok := status["errorCode"].(string); ok {
					if msg, ok := status["errorMessage"].(string); ok {
						errorMessage = fmt.Sprintf("[%s] %s", code, msg)
					}
				} else if msg, ok := status["errorMessage"].(string); ok {
					errorMessage = msg
				}
			}

			db.UpdateError(gid, errorMessage)
		})

		aria2Client.SetOnCompleteHandler(func(gid string) {
			// Get task from DB (we should have it if started from UI)
			upload, err := db.GetUpload(gid)

			// Get File Path & Status from Aria2
			status, errAria := aria2Client.TellStatus(gid)
			if errAria != nil {
				log.Printf("Failed to get status for %s: %v", gid, errAria)
				return
			}

			path := ""
			files, ok := status["files"].([]interface{})
			if ok && len(files) > 0 {
				file0, ok := files[0].(map[string]interface{})
				if ok {
					path, _ = file0["path"].(string)
				}
			}

			totalLengthInt := int64(0)
			if totalLength, ok := status["totalLength"].(string); ok {
				totalLengthInt, _ = strconv.ParseInt(totalLength, 10, 64)
			}

			// Record download stats
			if totalLengthInt > 0 {
				db.AddDownloadedBytes(totalLengthInt)
				db.IncrementCompletedTasks()
			}

			// If not in DB, create a minimal record so it shows in finished list
			if err != nil {
				log.Printf("Task %s completed but not in DB. Adopting as local-only.", gid)
				db.SaveUpload(store.UploadState{
					Gid:         gid,
					Status:      "complete",
					StartedAt:   time.Now(),
					FilePath:    path,
					TotalLength: totalLengthInt,
					UpdatedAt:   time.Now(),
				})
				return
			}

			if upload.Status != "pending" {
				return
			}

			// Magnet Handoff: Check for followedBy
			if followedBy, ok := status["followedBy"].([]interface{}); ok && len(followedBy) > 0 {
				for _, child := range followedBy {
					if childGid, ok := child.(string); ok {
						if adopted, _ := db.AdoptChildTask(childGid, gid, upload.TargetRemote, upload.Options); adopted {
							log.Printf("Magnet Handoff (WS): Adopting child GID %s", childGid)
						}
					}
				}
			}

			if upload.TargetRemote == "" {
				// Local only download, just mark complete
				log.Printf("Task %s (local) completed.", gid)
				if err := db.MarkComplete(gid, path, totalLengthInt); err != nil {
					log.Printf("Failed to mark task %s as complete: %v", gid, err)
				}
				return
			}

			log.Printf("Task %s completed. Starting upload to %s...", gid, upload.TargetRemote)

			// Determine if it's a folder or a single file
			// Aria2 'dir' is the base download folder.
			// For single files, path is dir/filename.
			// For folders, path is dir/folder/file.
			baseDir, _ := status["dir"].(string)

			// Trigger Rclone
			customJobId := utils.HexGidToInt64(gid)
			var jobId string
			var errTrigger error

			// If the file is in a subfolder of baseDir, upload the subfolder
			rel, errRel := filepath.Rel(baseDir, path)
			if errRel == nil && strings.Contains(rel, string(os.PathSeparator)) {
				// It's in a subfolder. Find the top-level subfolder.
				parts := strings.Split(rel, string(os.PathSeparator))
				topFolder := parts[0]
				srcFs := filepath.Join(baseDir, topFolder)
				dstFs := fmt.Sprintf("%s/%s", upload.TargetRemote, topFolder)
				log.Printf("Detected folder download. Uploading %s to %s", srcFs, dstFs)
				jobId, errTrigger = rcloneClient.CopyDirAsync(srcFs, dstFs, gid, customJobId)
			} else {
				// Single file
				filename := filepath.Base(path)
				log.Printf("Detected single file download. Uploading %s to %s", path, upload.TargetRemote)
				jobId, errTrigger = rcloneClient.CopyFileAsync(baseDir, filename, upload.TargetRemote, filename, gid, customJobId)
			}

			if errTrigger != nil {
				log.Printf("Failed to start async upload for %s: %v", gid, errTrigger)
				db.UpdateStatus(gid, "error", "")
			} else {
				log.Printf("Started async upload for %s (Job: %s)", gid, jobId)
				if err := db.StartUpload(gid, jobId, path, totalLengthInt); err != nil {
					log.Printf("Failed to update upload start for %s (already uploading?): %v", gid, err)
				}
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

	// Force one final sync of all active tasks before closing DB
	aria2Client := aria2.NewClient(fmt.Sprintf("ws://localhost:%d/jsonrpc", Aria2RPCPort), aria2Secret)
	log.Println("Final sync of active tasks...")
	syncAllActiveTasks(aria2Client, db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func syncAllActiveTasks(aria2Client *aria2.Client, db *store.DB) {
	activeRes, err := aria2Client.Call("aria2.tellActive")
	if err == nil {
		if activeTasks, ok := activeRes.([]interface{}); ok {
			for _, taskInterface := range activeTasks {
				if taskMap, ok := taskInterface.(map[string]interface{}); ok {
					gid, _ := taskMap["gid"].(string)
					completedLength, _ := taskMap["completedLength"].(string)
					totalLength, _ := taskMap["totalLength"].(string)
					status, _ := taskMap["status"].(string)

					cLen, _ := strconv.ParseInt(completedLength, 10, 64)
					tLen, _ := strconv.ParseInt(totalLength, 10, 64)

					db.UpdateProgress(gid, cLen, tLen)
					db.UpdateStatus(gid, status, "")
				}
			}
		}
	}
}

// importActiveTasks imports all tasks from Aria2 into BoltDB to sync state on startup
func importActiveTasks(aria2Client *aria2.Client, db *store.DB) error {
	log.Println("Importing Aria2 tasks into BoltDB...")

	// Get active tasks (tellActive takes optional keys, not offset/num)
	activeRes, err := aria2Client.Call("aria2.tellActive")
	if err == nil {
		if activeTasks, ok := activeRes.([]interface{}); ok {
			for _, taskInterface := range activeTasks {
				if taskMap, ok := taskInterface.(map[string]interface{}); ok {
					gid, _ := taskMap["gid"].(string)
					totalLength, _ := taskMap["totalLength"].(string)
					status, _ := taskMap["status"].(string)

					// Extract files for path
					filePath := ""
					if files, ok := taskMap["files"].([]interface{}); ok && len(files) > 0 {
						if file0, ok := files[0].(map[string]interface{}); ok {
							filePath, _ = file0["path"].(string)
						}
					}

					totalLengthInt, _ := strconv.ParseInt(totalLength, 10, 64)

					// Upsert into BoltDB
					if err := db.UpsertTask(store.UploadState{
						Gid:         gid,
						Status:      status,
						FilePath:    filePath,
						TotalLength: totalLengthInt,
						StartedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					}); err != nil {
						log.Printf("Failed to import active task %s: %v", gid, err)
					} else {
						log.Printf("Imported active task %s", gid)
					}
				}
			}
		}
	}

	// Get stopped tasks (recent ones)
	stoppedRes, err := aria2Client.Call("aria2.tellStopped", 0, 100)
	if err == nil {
		if stoppedTasks, ok := stoppedRes.([]interface{}); ok {
			for _, taskInterface := range stoppedTasks {
				if taskMap, ok := taskInterface.(map[string]interface{}); ok {
					gid, _ := taskMap["gid"].(string)
					totalLength, _ := taskMap["totalLength"].(string)
					status, _ := taskMap["status"].(string)

					filePath := ""
					if files, ok := taskMap["files"].([]interface{}); ok && len(files) > 0 {
						if file0, ok := files[0].(map[string]interface{}); ok {
							filePath, _ = file0["path"].(string)
						}
					}

					totalLengthInt, _ := strconv.ParseInt(totalLength, 10, 64)

					if err := db.UpsertTask(store.UploadState{
						Gid:         gid,
						Status:      status,
						FilePath:    filePath,
						TotalLength: totalLengthInt,
						StartedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					}); err != nil {
						log.Printf("Failed to import stopped task %s: %v", gid, err)
					}
				}
			}
		}
	}

	return nil
}

func waitForAria2(ctx context.Context, client *aria2.Client) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := client.GetVersion(); err == nil {
				return nil
			}
		}
	}
}
