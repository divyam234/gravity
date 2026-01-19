package aria2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"gravity/internal/engine"
)

type Engine struct {
	runner *Runner
	client *Client

	onProgress func(id string, progress engine.Progress)
	onComplete func(id string, filePath string)
	onError    func(id string, err error)

	mu sync.RWMutex
}

func NewEngine(port int, secret, dataDir string) *Engine {
	runner := NewRunner(port, secret, dataDir)
	client := NewClient(fmt.Sprintf("ws://localhost:%d/jsonrpc", port), secret)

	return &Engine{
		runner: runner,
		client: client,
	}
}

func (e *Engine) Start(ctx context.Context) error {
	if err := e.runner.Start(); err != nil {
		return err
	}

	// Wait for Aria2 to be ready
	ready := false
	for i := 0; i < 10; i++ {
		if _, err := e.client.Call(ctx, "aria2.getVersion"); err == nil {
			ready = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !ready {
		return fmt.Errorf("aria2 engine failed to become ready")
	}

	// Start listener
	go func() {
		err := e.client.Listen(context.Background(), e.handleNotification)
		if err != nil {
			log.Printf("Aria2 listener stopped: %v", err)
		}
	}()

	// Start poller for progress
	go e.pollProgress()

	return nil
}

func (e *Engine) Stop() error {
	return e.runner.Stop()
}

func (e *Engine) Add(ctx context.Context, url string, opts engine.DownloadOptions) (string, error) {
	ariaOpts := make(map[string]interface{})
	if opts.Filename != "" {
		ariaOpts["out"] = opts.Filename
	}
	if opts.Dir != "" {
		ariaOpts["dir"] = opts.Dir
	}
	for k, v := range opts.Headers {
		ariaOpts["header"] = append(ariaOpts["header"].([]string), fmt.Sprintf("%s: %s", k, v))
	}
	if opts.MaxSpeed > 0 {
		ariaOpts["max-download-limit"] = strconv.FormatInt(opts.MaxSpeed, 10)
	}
	if opts.Connections > 0 {
		ariaOpts["split"] = strconv.Itoa(opts.Connections)
		ariaOpts["max-connection-per-server"] = strconv.Itoa(opts.Connections)
	}

	res, err := e.client.Call(ctx, "aria2.addUri", []string{url}, ariaOpts)
	if err != nil {
		return "", err
	}

	var gid string
	if err := json.Unmarshal(res, &gid); err != nil {
		return "", err
	}

	return gid, nil
}

func (e *Engine) Pause(ctx context.Context, id string) error {
	_, err := e.client.Call(ctx, "aria2.pause", id)
	return err
}

func (e *Engine) Resume(ctx context.Context, id string) error {
	_, err := e.client.Call(ctx, "aria2.unpause", id)
	return err
}

func (e *Engine) Cancel(ctx context.Context, id string) error {
	_, err := e.client.Call(ctx, "aria2.forceRemove", id)
	return err
}

func (e *Engine) Remove(ctx context.Context, id string) error {
	_, err := e.client.Call(ctx, "aria2.removeDownloadResult", id)
	return err
}

func (e *Engine) Status(ctx context.Context, id string) (*engine.DownloadStatus, error) {
	res, err := e.client.Call(ctx, "aria2.tellStatus", id)
	if err != nil {
		return nil, err
	}

	var status Aria2Task
	if err := json.Unmarshal(res, &status); err != nil {
		return nil, err
	}

	return e.mapStatus(&status), nil
}

func (e *Engine) List(ctx context.Context) ([]*engine.DownloadStatus, error) {
	// For simplicity, just get active and stopped tasks
	var results []*engine.DownloadStatus

	active, _ := e.client.Call(ctx, "aria2.tellActive")
	var activeTasks []Aria2Task
	json.Unmarshal(active, &activeTasks)
	for _, t := range activeTasks {
		results = append(results, e.mapStatus(&t))
	}

	stopped, _ := e.client.Call(ctx, "aria2.tellStopped", 0, 100)
	var stoppedTasks []Aria2Task
	json.Unmarshal(stopped, &stoppedTasks)
	for _, t := range stoppedTasks {
		results = append(results, e.mapStatus(&t))
	}

	return results, nil
}

func (e *Engine) OnProgress(h func(string, engine.Progress)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onProgress = h
}
func (e *Engine) OnComplete(h func(string, string)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onComplete = h
}
func (e *Engine) OnError(h func(string, error)) { e.mu.Lock(); defer e.mu.Unlock(); e.onError = h }

func (e *Engine) handleNotification(method string, gid string) {
	switch method {
	case "aria2.onDownloadComplete":
		if h := e.onComplete; h != nil {
			status, _ := e.Status(context.Background(), gid)
			filePath := ""
			if status != nil {
				filePath = status.Dir + "/" + status.Filename
			}
			h(gid, filePath)
		}
	case "aria2.onDownloadError":
		if h := e.onError; h != nil {
			status, _ := e.Status(context.Background(), gid)
			errMsg := "unknown error"
			if status != nil {
				errMsg = status.Error
			}
			h(gid, fmt.Errorf(errMsg))
		}
	}
}

func (e *Engine) pollProgress() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h := e.onProgress
		if h == nil {
			continue
		}

		tasks, _ := e.List(context.Background())
		for _, t := range tasks {
			if t.Status == "active" || t.Status == "downloading" {
				h(t.ID, engine.Progress{
					Downloaded: t.Downloaded,
					Size:       t.Size,
					Speed:      t.Speed,
					ETA:        t.Eta,
				})
			}
		}
	}
}

type Aria2Task struct {
	Gid             string      `json:"gid"`
	Status          string      `json:"status"`
	TotalLength     string      `json:"totalLength"`
	CompletedLength string      `json:"completedLength"`
	DownloadSpeed   string      `json:"downloadSpeed"`
	Eta             string      `json:"eta,omitempty"`
	Connections     string      `json:"connections"`
	ErrorMessage    string      `json:"errorMessage"`
	Dir             string      `json:"dir"`
	Files           []Aria2File `json:"files"`
}

func (e *Engine) mapStatus(t *Aria2Task) *engine.DownloadStatus {
	total, _ := strconv.ParseInt(t.TotalLength, 10, 64)
	completed, _ := strconv.ParseInt(t.CompletedLength, 10, 64)
	speed, _ := strconv.ParseInt(t.DownloadSpeed, 10, 64)
	conn, _ := strconv.Atoi(t.Connections)
	eta, _ := strconv.Atoi(t.Eta)

	filename := ""
	if len(t.Files) > 0 {
		filename = t.Files[0].Path
	}

	return &engine.DownloadStatus{
		ID:          t.Gid,
		Status:      t.Status,
		URL:         "", // Aria2 doesn't return original URL easily in tellStatus
		Filename:    filename,
		Dir:         t.Dir,
		Size:        total,
		Downloaded:  completed,
		Speed:       speed,
		Connections: conn,
		Eta:         eta,
		Error:       t.ErrorMessage,
	}
}

type Aria2File struct {
	Path string `json:"path"`
}
