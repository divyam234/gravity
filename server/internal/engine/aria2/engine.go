package aria2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
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

	waiting, _ := e.client.Call(ctx, "aria2.tellWaiting", 0, 100)
	var waitingTasks []Aria2Task
	json.Unmarshal(waiting, &waitingTasks)
	for _, t := range waitingTasks {
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

func (e *Engine) Sync(ctx context.Context) error {
	// Aria2 handles session recovery via its session file automatically.
	// We can use this method to verify connectivity if needed.
	return nil
}

func (e *Engine) Configure(ctx context.Context, options map[string]string) error {
	// Map generic keys to Aria2 keys
	keyMap := map[string]string{
		"downloadDir":              "dir",
		"maxConcurrentDownloads":   "max-concurrent-downloads",
		"globalDownloadSpeedLimit": "max-overall-download-limit",
		"globalUploadSpeedLimit":   "max-overall-upload-limit",
		"maxConnectionsPerServer":  "max-connection-per-server",
		"concurrency":              "split",
		"userAgent":                "user-agent",
		"proxyUrl":                 "all-proxy",
		"proxyUser":                "all-proxy-user",
		"proxyPassword":            "all-proxy-passwd",
		"seedRatio":                "seed-ratio",
		"seedTimeLimit":            "seed-time",
		"connectTimeout":           "connect-timeout",
		"maxRetries":               "max-tries",
		"checkIntegrity":           "check-integrity",
		"continueDownloads":        "continue",
		"checkCertificate":         "check-certificate",
	}

	ariaOpts := make(map[string]interface{})
	for k, v := range options {
		if ariaKey, ok := keyMap[k]; ok {
			ariaOpts[ariaKey] = v
		} else {
			// If key not in map, assume it might be a direct key or ignored?
			// For safety and abstraction, we should probably ignore unknown keys or pass them through if we trust the source.
			// Let's pass through for now to support legacy/direct usage if needed, or strict.
			// Given "nice abstraction" goal, strict mapping is better, but passing through allows fallback.
			// I'll pass through with logging (if I had logger).
			// I'll pass through for flexibility.
			ariaOpts[k] = v
		}
	}
	_, err := e.client.Call(ctx, "aria2.changeGlobalOption", ariaOpts)
	return err
}

func (e *Engine) Version(ctx context.Context) (string, error) {
	res, err := e.client.Call(ctx, "aria2.getVersion", nil)
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

func (e *Engine) GetClient() *Client {
	return e.client
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
				filePath = status.Filename
				if !filepath.IsAbs(filePath) {
					filePath = filepath.Join(status.Dir, status.Filename)
				}

				// If the file is in a subdirectory of the download dir,
				// it's likely a multi-file download (torrent).
				// We should return the directory path instead of the first file.
				parent := filepath.Dir(filePath)
				if parent != filepath.Clean(status.Dir) && parent != "." && parent != "/" {
					filePath = parent
				}
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
			h(gid, fmt.Errorf("%s", errMsg))
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

	// Calculate ETA manually if speed > 0
	eta := 0
	if speed > 0 && total > 0 {
		remaining := total - completed
		if remaining > 0 {
			eta = int(remaining / speed)
		}
	}

	// Map Aria2 status to Gravity canonical status
	status := t.Status
	switch t.Status {
	case "active":
		status = "active"
	case "waiting":
		status = "waiting"
	case "paused":
		status = "paused"
	case "complete":
		status = "complete"
	case "error":
		status = "error"
	}

	filename := ""
	if len(t.Files) > 0 {
		filename = filepath.Base(t.Files[0].Path)
	}

	return &engine.DownloadStatus{
		ID:          t.Gid,
		Status:      status,
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
