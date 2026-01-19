package rclone

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"gravity/internal/engine"
)

type Engine struct {
	runner *Runner
	client *Client

	onProgress func(jobID string, progress engine.UploadProgress)
	onComplete func(jobID string)
	onError    func(jobID string, err error)

	activeJobs map[string]struct {
		trackID string
		size    int64
	}
	mu sync.RWMutex
}

func NewEngine(port int) *Engine {
	runner := NewRunner(port)
	client := NewClient(fmt.Sprintf("http://localhost:%d", port))

	return &Engine{
		runner: runner,
		client: client,
		activeJobs: make(map[string]struct {
			trackID string
			size    int64
		}),
	}
}

func (e *Engine) Start(ctx context.Context) error {
	if err := e.runner.Start(); err != nil {
		return err
	}

	// Wait for Rclone to be ready
	ready := false
	for i := 0; i < 10; i++ {
		if _, err := e.client.Call(ctx, "core/version", nil); err == nil {
			ready = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !ready {
		return fmt.Errorf("rclone engine failed to become ready")
	}

	// Start poller for progress
	go e.pollProgress()

	return nil
}

func (e *Engine) Stop() error {
	return e.runner.Stop()
}

func (e *Engine) Upload(ctx context.Context, src, dst string, opts engine.UploadOptions) (string, error) {
	info, err := os.Stat(src)
	if err != nil {
		return "", fmt.Errorf("failed to stat source: %w", err)
	}

	var method string
	params := map[string]interface{}{
		"_async": true,
		"_group": opts.TrackingID,
		"_jobid": opts.JobID,
	}

	if info.IsDir() {
		method = "sync/copy"
		params["srcFs"] = src
		params["dstFs"] = filepath.Join(dst, filepath.Base(src))
	} else {
		method = "operations/copyfile"
		params["srcFs"] = filepath.Dir(src)
		params["srcRemote"] = filepath.Base(src)
		params["dstFs"] = dst
		params["dstRemote"] = filepath.Base(src)
	}

	_, err = e.client.Call(ctx, method, params)
	if err != nil {
		return "", err
	}

	jobID := fmt.Sprintf("%d", opts.JobID)
	e.mu.Lock()
	e.activeJobs[jobID] = struct {
		trackID string
		size    int64
	}{trackID: opts.TrackingID, size: info.Size()}
	e.mu.Unlock()

	return jobID, nil
}

func (e *Engine) Cancel(ctx context.Context, jobID string) error {
	// Convert string job ID to int64 for rclone API
	jobIDInt, err := strconv.ParseInt(jobID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid job ID: %w", err)
	}

	// Get the tracking ID before we delete from activeJobs
	e.mu.RLock()
	jobData, exists := e.activeJobs[jobID]
	e.mu.RUnlock()

	// Stop the job
	params := map[string]interface{}{
		"jobid": jobIDInt,
	}
	_, err = e.client.Call(ctx, "job/stop", params)
	// Clean up stats for this group if we have the tracking ID
	if exists && jobData.trackID != "" {
		e.client.Call(ctx, "core/stats-delete", map[string]interface{}{
			"group": jobData.trackID,
		})
	}

	// Always clean up our tracking
	e.mu.Lock()
	delete(e.activeJobs, jobID)
	e.mu.Unlock()

	return err
}

func (e *Engine) Status(ctx context.Context, jobID string) (*engine.UploadStatus, error) {
	// Convert string job ID to int64 for rclone API
	jobIDInt, err := strconv.ParseInt(jobID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %w", err)
	}

	params := map[string]interface{}{
		"jobid": jobIDInt,
	}
	res, err := e.client.Call(ctx, "job/status", params)
	if err != nil {
		return nil, err
	}

	var status struct {
		Finished bool   `json:"finished"`
		Success  bool   `json:"success"`
		Error    string `json:"error"`
		Group    string `json:"group"`
	}
	if err := json.Unmarshal(res, &status); err != nil {
		return nil, err
	}

	result := &engine.UploadStatus{
		JobID:  jobID,
		Status: "running",
		Error:  status.Error,
	}

	if status.Finished {
		if status.Success {
			result.Status = "complete"
		} else {
			result.Status = "error"
		}
	}

	return result, nil
}

func (e *Engine) GetGlobalStats(ctx context.Context) (*engine.GlobalStats, error) {
	res, err := e.client.Call(ctx, "core/stats", nil)
	if err != nil {
		return nil, err
	}

	var stats struct {
		Transferring []struct {
			Speed float64 `json:"speed"`
		} `json:"transferring"`
	}
	if err := json.Unmarshal(res, &stats); err != nil {
		return nil, err
	}

	currentSpeed := float64(0)
	for _, t := range stats.Transferring {
		currentSpeed += t.Speed
	}

	return &engine.GlobalStats{
		Speed:           int64(currentSpeed),
		ActiveTransfers: len(stats.Transferring),
	}, nil
}

func (e *Engine) Version(ctx context.Context) (string, error) {
	res, err := e.client.Call(ctx, "core/version", nil)
	if err != nil {
		return "", err
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(res, &v); err != nil {
		return "", err
	}
	return v.Version, nil
}

func (e *Engine) ListRemotes(ctx context.Context) ([]engine.Remote, error) {
	res, err := e.client.Call(ctx, "config/listremotes", nil)
	if err != nil {
		return nil, err
	}

	var remotes struct {
		Remotes []string `json:"remotes"`
	}
	if err := json.Unmarshal(res, &remotes); err != nil {
		return nil, err
	}

	results := make([]engine.Remote, 0, len(remotes.Remotes))
	for _, r := range remotes.Remotes {
		results = append(results, engine.Remote{
			Name:      r,
			Connected: true,
		})
	}

	return results, nil
}

func (e *Engine) CreateRemote(ctx context.Context, name, rtype string, config map[string]string) error {
	params := map[string]interface{}{
		"name":       name,
		"type":       rtype,
		"parameters": config,
	}
	_, err := e.client.Call(ctx, "config/create", params)
	return err
}

func (e *Engine) DeleteRemote(ctx context.Context, name string) error {
	params := map[string]interface{}{
		"name": name,
	}
	_, err := e.client.Call(ctx, "config/delete", params)
	return err
}

func (e *Engine) TestRemote(ctx context.Context, name string) error {
	params := map[string]interface{}{
		"fs": name + ":",
	}
	_, err := e.client.Call(ctx, "operations/list", params)
	return err
}

func (e *Engine) OnProgress(h func(string, engine.UploadProgress)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onProgress = h
}
func (e *Engine) OnComplete(h func(string))     { e.mu.Lock(); defer e.mu.Unlock(); e.onComplete = h }
func (e *Engine) OnError(h func(string, error)) { e.mu.Lock(); defer e.mu.Unlock(); e.onError = h }

func (e *Engine) pollProgress() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		e.mu.RLock()
		jobs := make(map[string]struct {
			trackID string
			size    int64
		})
		for id, data := range e.activeJobs {
			jobs[id] = data
		}
		e.mu.RUnlock()

		for id, data := range jobs {
			status, err := e.Status(context.Background(), id)
			if err != nil {
				continue
			}

			if status.Status == "complete" {
				if h := e.onComplete; h != nil {
					h(data.trackID) // Pass trackID (download ID) instead of rclone job ID
				}
				e.mu.Lock()
				delete(e.activeJobs, id)
				e.mu.Unlock()
			} else if status.Status == "error" {
				if h := e.onError; h != nil {
					h(data.trackID, fmt.Errorf("%s", status.Error)) // Pass trackID (download ID)
				}
				e.mu.Lock()
				delete(e.activeJobs, id)
				e.mu.Unlock()
			} else {
				// Also query group-specific stats for better accuracy
				groupRes, err := e.client.Call(context.Background(), "core/stats", map[string]string{"group": data.trackID})
				if err == nil {
					var gStats struct {
						Bytes int64   `json:"bytes"`
						Speed float64 `json:"speed"`
					}
					if err := json.Unmarshal(groupRes, &gStats); err == nil {
						if h := e.onProgress; h != nil {
							// Pass trackID (download ID) instead of rclone job ID
							h(data.trackID, engine.UploadProgress{
								Uploaded: gStats.Bytes,
								Size:     data.size,
								Speed:    int64(gStats.Speed),
							})
						}
					}
				}
			}
		}
	}
}
