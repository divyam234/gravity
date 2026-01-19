package poller

import (
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
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
	// Sync background uploads/restorations
	p.syncManagedTasks()

	// Sync active task progress for crash safety (Lazy Checkpoint)
	p.syncActiveProgress()
}

func (p *Poller) syncActiveProgress() {
	activeRes, err := p.aria2Client.Call("aria2.tellActive")
	if err != nil {
		return
	}

	if tasks, ok := activeRes.([]interface{}); ok {
		for _, t := range tasks {
			if taskMap, ok := t.(map[string]interface{}); ok {
				gid, _ := taskMap["gid"].(string)
				completedLength, _ := taskMap["completedLength"].(string)
				totalLength, _ := taskMap["totalLength"].(string)

				cLen, _ := strconv.ParseInt(completedLength, 10, 64)
				tLen, _ := strconv.ParseInt(totalLength, 10, 64)

				p.db.UpdateProgress(gid, cLen, tLen)
			}
		}
	}
}

func (p *Poller) syncManagedTasks() {
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
		p.db.UpdateStatus(task.Gid, "pending", "")
		return
	}

	status, err := p.rcloneClient.JobStatus(task.JobID)
	if err != nil {
		// Job failed
		log.Printf("Poller: Upload Job %s for GID %s failed: %v", task.JobID, task.Gid, err)
		// Retry logic?
		if task.RetryCount < 3 {
			p.db.IncrementRetry(task.Gid)
			p.db.UpdateStatus(task.Gid, "pending", "")
			log.Printf("Poller: Retrying task %s (Attempt %d)", task.Gid, task.RetryCount+1)
		} else {
			p.db.UpdateStatus(task.Gid, "error", "")
		}
		return
	}

	if status == "success" {
		log.Printf("Poller: Upload Job %s for GID %s succeeded", task.JobID, task.Gid)
		p.db.UpdateStatus(task.Gid, "complete", "")

		// Record upload stats - get file size from Aria2 status
		if ariaStatus, err := p.aria2Client.TellStatus(task.Gid); err == nil {
			if totalLength, ok := ariaStatus["totalLength"].(string); ok {
				if bytes, err := strconv.ParseInt(totalLength, 10, 64); err == nil && bytes > 0 {
					p.db.AddUploadedBytes(bytes)
					p.db.IncrementUploadedTasks()
				}
			}
		}
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
			p.db.UpdateStatus(task.Gid, "error", "")
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

			if adopted, _ := p.db.AdoptChildTask(childGid, task.Gid, task.TargetRemote, task.Options); adopted {
				log.Printf("Magnet Handoff: Adopting child GID %s from parent %s", childGid, task.Gid)
			}
		}
	}

	// 2. Check if Complete
	ariaStatus, _ := status["status"].(string)
	if ariaStatus == "complete" {
		log.Printf("Poller: Detected complete task %s (Missed WebSocket?). Triggering upload.", task.Gid)

		// Trigger Upload Logic
		path := extractFilePath(status)
		if path == "" {
			log.Printf("Poller: No file path found for task %s", task.Gid)
			return
		}

		if task.TargetRemote == "" {
			log.Printf("Poller: Task %s (local) completed.", task.Gid)
			totalLengthInt := int64(0)
			if totalLength, ok := status["totalLength"].(string); ok {
				totalLengthInt, _ = strconv.ParseInt(totalLength, 10, 64)
			}
			p.db.MarkComplete(task.Gid, path, totalLengthInt)
			return
		}

		baseDir, _ := status["dir"].(string)

		// Use Async Upload
		// Pass deterministic Job ID (Hex GID -> Int64)
		customJobId := utils.HexGidToInt64(task.Gid)
		var jobId string
		var errTrigger error

		// Detection logic for folder vs file
		rel, errRel := filepath.Rel(baseDir, path)
		if errRel == nil && strings.Contains(rel, string(filepath.Separator)) {
			// Folder
			parts := strings.Split(rel, string(filepath.Separator))
			topFolder := parts[0]
			srcFs := filepath.Join(baseDir, topFolder)
			dstFs := fmt.Sprintf("%s/%s", task.TargetRemote, topFolder)
			log.Printf("Poller: Detected folder download. Uploading %s to %s", srcFs, dstFs)
			jobId, errTrigger = p.rcloneClient.CopyDirAsync(srcFs, dstFs, task.Gid, customJobId)
		} else {
			// Single file
			filename := filepath.Base(path)
			log.Printf("Poller: Detected single file download. Uploading %s to %s", path, task.TargetRemote)
			jobId, errTrigger = p.rcloneClient.CopyFileAsync(baseDir, filename, task.TargetRemote, filename, task.Gid, customJobId)
		}

		if errTrigger != nil {
			log.Printf("Poller: Failed to start async upload for %s: %v", task.Gid, errTrigger)
			p.db.UpdateStatus(task.Gid, "error", "")
		} else {
			log.Printf("Poller: Started async upload for %s (Job: %s)", task.Gid, jobId)

			var totalLengthInt int64
			if totalLength, ok := status["totalLength"].(string); ok {
				totalLengthInt, _ = strconv.ParseInt(totalLength, 10, 64)
			}

			if err := p.db.StartUpload(task.Gid, jobId, path, totalLengthInt); err != nil {
				log.Printf("Poller: Failed to update upload start for %s (already uploading?): %v", task.Gid, err)
			}
		}
	} else if ariaStatus == "error" || ariaStatus == "removed" {
		p.db.UpdateStatus(task.Gid, "error", "")
	}
}

// extractFilePath safely extracts the first file path from an Aria2 status response.
func extractFilePath(status map[string]interface{}) string {
	files, ok := status["files"].([]interface{})
	if !ok || len(files) == 0 {
		return ""
	}
	file0, ok := files[0].(map[string]interface{})
	if !ok {
		return ""
	}
	path, _ := file0["path"].(string)
	return path
}
