package rclone

import (
	"context"
	"encoding/json"
	"fmt"
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

	activeJobs map[string]struct{}
	mu         sync.RWMutex
}

func NewEngine(port int) *Engine {
	runner := NewRunner(port)
	client := NewClient(fmt.Sprintf("http://localhost:%d", port))

	return &Engine{
		runner:     runner,
		client:     client,
		activeJobs: make(map[string]struct{}),
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
	params := map[string]interface{}{
		"srcFs":  src,
		"dstFs":  dst,
		"_async": true,
	}

	res, err := e.client.Call(ctx, "sync/copy", params)
	if err != nil {
		return "", err
	}

	var asyncRes struct {
		JobID int64 `json:"jobid"`
	}
	if err := json.Unmarshal(res, &asyncRes); err != nil {
		return "", err
	}

	jobID := fmt.Sprintf("%d", asyncRes.JobID)
	e.mu.Lock()
	e.activeJobs[jobID] = struct{}{}
	e.mu.Unlock()

	return jobID, nil
}

func (e *Engine) Cancel(ctx context.Context, jobID string) error {
	params := map[string]interface{}{
		"jobid": jobID,
	}
	_, err := e.client.Call(ctx, "job/stop", params)

	e.mu.Lock()
	delete(e.activeJobs, jobID)
	e.mu.Unlock()

	return err
}

func (e *Engine) Status(ctx context.Context, jobID string) (*engine.UploadStatus, error) {
	params := map[string]interface{}{
		"jobid": jobID,
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

	results := make([]engine.Remote, len(remotes.Remotes))
	for i, r := range remotes.Remotes {
		results[i] = engine.Remote{
			Name:      r,
			Connected: true,
		}
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
		jobs := make([]string, 0, len(e.activeJobs))
		for id := range e.activeJobs {
			jobs = append(jobs, id)
		}
		e.mu.RUnlock()

		for _, id := range jobs {
			status, err := e.Status(context.Background(), id)
			if err != nil {
				continue
			}

			if status.Status == "complete" {
				if h := e.onComplete; h != nil {
					h(id)
				}
				e.mu.Lock()
				delete(e.activeJobs, id)
				e.mu.Unlock()
			} else if status.Status == "error" {
				if h := e.onError; h != nil {
					h(id, fmt.Errorf(status.Error))
				}
				e.mu.Lock()
				delete(e.activeJobs, id)
				e.mu.Unlock()
			} else {
				res, err := e.client.Call(context.Background(), "core/stats", nil)
				if err == nil {
					var stats struct {
						Transferring []struct {
							Size  int64  `json:"size"`
							Bytes int64  `json:"bytes"`
							Speed int64  `json:"speed"`
							Group string `json:"group"`
						} `json:"transferring"`
					}
					if err := json.Unmarshal(res, &stats); err == nil {
						for _, t := range stats.Transferring {
							if t.Group == "job/"+id {
								if h := e.onProgress; h != nil {
									h(id, engine.UploadProgress{
										Uploaded: t.Bytes,
										Size:     t.Size,
										Speed:    t.Speed,
									})
								}
							}
						}
					}
				}
			}
		}
	}
}
