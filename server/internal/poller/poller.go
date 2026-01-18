package poller

import (
	"log"
	"path/filepath"
	"time"

	"aria2-rclone-ui/internal/aria2"
	"aria2-rclone-ui/internal/rclone"
	"aria2-rclone-ui/internal/store"
	"aria2-rclone-ui/internal/utils"
)

type Poller struct {
	db           *store.DB
	aria2Client  *aria2.Client
	rcloneClient *rclone.Client
}

func New(db *store.DB, aria2 *aria2.Client, rclone *rclone.Client) *Poller {
	return &Poller{
		db:           db,
		aria2Client:  aria2,
		rcloneClient: rclone,
	}
}

// Sync performs a single reconciliation pass
func (p *Poller) Sync() {
	tasks, err := p.db.GetPendingUploads()
	if err != nil {
		log.Printf("Poller: Failed to fetch tasks: %v", err)
		return
	}

	for _, task := range tasks {
		if task.Status == "pending" {
			p.syncPendingTask(task)
		} else if task.Status == "uploading" {
			p.syncUploadingTask(task)
		}
	}
}

func (p *Poller) syncUploadingTask(task store.UploadState) {
	if task.JobID == "" {
		// Should not happen if started correctly via Async.
		// If empty, maybe it's a legacy synchronous upload or stuck.
		// Reset to pending to retry.
		log.Printf("Poller: Task %s is 'uploading' but no JobID. Resetting.", task.Gid)
		p.db.UpdateStatus(task.Gid, "pending", 0)
		return
	}

	status, err := p.rcloneClient.JobStatus(task.JobID)
	if err != nil {
		// Job failed
		log.Printf("Poller: Upload Job %s for GID %s failed: %v", task.JobID, task.Gid, err)
		// Retry logic?
		if task.RetryCount < 3 {
			p.db.IncrementRetry(task.Gid)
			p.db.UpdateStatus(task.Gid, "pending", 0)
			log.Printf("Poller: Retrying task %s (Attempt %d)", task.Gid, task.RetryCount+1)
		} else {
			p.db.UpdateStatus(task.Gid, "error", 0)
		}
		return
	}

	if status == "success" {
		log.Printf("Poller: Upload Job %s for GID %s succeeded", task.JobID, task.Gid)
		p.db.UpdateStatus(task.Gid, "complete", 0)
	}
	// If "running", do nothing
}

func (p *Poller) syncPendingTask(task store.UploadState) {
	// 1. Check Aria2 Status
	status, err := p.aria2Client.TellStatus(task.Gid)
	if err != nil {
		// Error fetching status (likely 404 if removed or lost)
		// If it's been pending for > 1 minute and not found, it might be a zombie
		if time.Since(task.StartedAt) > 1*time.Minute {
			// Try to restore or mark error?
			// For now, let's just log. The main restoration loop handles restart.
			// If it's 404, it means it's NOT in Aria2.
			log.Printf("Poller: Task %s not found in Aria2. Marking error.", task.Gid)
			p.db.UpdateStatus(task.Gid, "error", 0)
		}
		return
	}

	// 1.5 Check for FollowedBy (Magnet Handoff)
	if followedBy, ok := status["followedBy"].([]interface{}); ok && len(followedBy) > 0 {
		for _, child := range followedBy {
			childGid, ok := child.(string)
			if !ok {
				continue
			}

			// Check if we already track it
			_, err := p.db.GetUpload(childGid)
			if err != nil {
				// Not found, so adopt it
				log.Printf("Magnet Handoff: Adopting child GID %s from parent %s", childGid, task.Gid)
				p.db.SaveUpload(store.UploadState{
					Gid:          childGid,
					TargetRemote: task.TargetRemote,
					Status:       "pending",
					StartedAt:    time.Now(),
					Options:      task.Options, // Inherit options
					// We don't inherit URIs/Torrent as it's a derived task
				})
			}
		}
	}

	// 2. Check if Complete
	ariaStatus, _ := status["status"].(string)
	if ariaStatus == "complete" {
		log.Printf("Poller: Detected complete task %s (Missed WebSocket?). Triggering upload.", task.Gid)

		// Trigger Upload Logic (Duplicated from main.go - should be refactored)
		files, ok := status["files"].([]interface{})
		if !ok || len(files) == 0 {
			return
		}
		file0 := files[0].(map[string]interface{})
		path := file0["path"].(string)

		// For local files, srcFs is dir, srcRemote is filename
		dir := filepath.Dir(path)
		filename := filepath.Base(path)
		remotePath := filename

		p.db.UpdateStatus(task.Gid, "uploading", 0)

		// Use Async Upload
		// Pass deterministic Job ID (Hex GID -> Int64)
		customJobId := utils.HexGidToInt64(task.Gid)

		jobId, err := p.rcloneClient.CopyFileAsync(dir, filename, task.TargetRemote, remotePath, customJobId)
		if err != nil {
			log.Printf("Failed to start async upload for %s: %v", task.Gid, err)
			p.db.UpdateStatus(task.Gid, "error", 0)
		} else {
			log.Printf("Started async upload for %s (Job: %s)", task.Gid, jobId)
			p.db.UpdateJob(task.Gid, jobId)
		}
	} else if ariaStatus == "error" || ariaStatus == "removed" {
		p.db.UpdateStatus(task.Gid, "error", 0)
	}
}
