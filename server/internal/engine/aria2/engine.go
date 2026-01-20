package aria2

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
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

	// Track reported stopped GIDs to avoid duplicate events
	reportedGids map[string]bool
}

func NewEngine(port int, dataDir string) *Engine {
	runner := NewRunner(port, dataDir)
	client := NewClient(fmt.Sprintf("http://localhost:%d/jsonrpc", port))

	return &Engine{
		runner:       runner,
		client:       client,
		reportedGids: make(map[string]bool),
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

	// Start poller for progress and lifecycle
	go e.poll()

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
	if len(opts.Headers) > 0 {
		var headers []string
		for k, v := range opts.Headers {
			headers = append(headers, fmt.Sprintf("%s: %s", k, v))
		}
		ariaOpts["header"] = headers
	}
	if opts.MaxSpeed > 0 {
		ariaOpts["max-download-limit"] = strconv.FormatInt(opts.MaxSpeed, 10)
	}
	if opts.Connections > 0 {
		ariaOpts["split"] = strconv.Itoa(opts.Connections)
		ariaOpts["max-connection-per-server"] = strconv.Itoa(opts.Connections)
	}

	// File selection for torrents/magnets
	if len(opts.SelectedFiles) > 0 {
		var indexes []string
		for _, idx := range opts.SelectedFiles {
			indexes = append(indexes, strconv.Itoa(idx))
		}
		ariaOpts["select-file"] = strings.Join(indexes, ",")
	}

	var method string
	var params []interface{}

	if opts.TorrentData != "" {
		method = "aria2.addTorrent"
		params = []interface{}{opts.TorrentData, []interface{}{}, ariaOpts}
	} else {
		method = "aria2.addUri"
		params = []interface{}{[]string{url}, ariaOpts}
	}

	res, err := e.client.Call(ctx, method, params...)
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

func (e *Engine) GetPeers(ctx context.Context, id string) ([]engine.DownloadPeer, error) {
	res, err := e.client.Call(ctx, "aria2.getPeers", id)
	if err != nil {
		return nil, err
	}

	var ariaPeers []struct {
		IP            string `json:"ip"`
		Port          string `json:"port"`
		DownloadSpeed string `json:"downloadSpeed"`
		UploadSpeed   string `json:"uploadSpeed"`
		Seeder        string `json:"seeder"`
	}
	if err := json.Unmarshal(res, &ariaPeers); err != nil {
		return nil, err
	}

	peers := make([]engine.DownloadPeer, 0, len(ariaPeers))
	for _, p := range ariaPeers {
		dSpeed, _ := strconv.ParseInt(p.DownloadSpeed, 10, 64)
		uSpeed, _ := strconv.ParseInt(p.UploadSpeed, 10, 64)
		peers = append(peers, engine.DownloadPeer{
			IP:            p.IP,
			Port:          p.Port,
			DownloadSpeed: dSpeed,
			UploadSpeed:   uSpeed,
			IsSeeder:      p.Seeder == "true",
		})
	}

	return peers, nil
}

func (e *Engine) List(ctx context.Context) ([]*engine.DownloadStatus, error) {
	results, err := e.fetchTasks(ctx)
	if err != nil {
		return nil, err
	}

	var statuses []*engine.DownloadStatus
	for _, t := range results {
		statuses = append(statuses, e.mapStatus(t))
	}
	return statuses, nil
}

func (e *Engine) Sync(ctx context.Context) error {
	return nil
}

func (e *Engine) Configure(ctx context.Context, options map[string]string) error {
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
		"listenPort":               "listen-port",
	}

	ariaOpts := make(map[string]interface{})
	for k, v := range options {
		if ariaKey, ok := keyMap[k]; ok {
			ariaOpts[ariaKey] = v
		} else {
			ariaOpts[k] = v
		}
	}
	_, err := e.client.Call(ctx, "aria2.changeGlobalOption", ariaOpts)
	return err
}

func (e *Engine) Version(ctx context.Context) (string, error) {
	res, err := e.client.Call(ctx, "aria2.getVersion")
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

type Aria2Task struct {
	Gid             string      `json:"gid"`
	Status          string      `json:"status"`
	TotalLength     string      `json:"totalLength"`
	CompletedLength string      `json:"completedLength"`
	DownloadSpeed   string      `json:"downloadSpeed"`
	Eta             string      `json:"eta,omitempty"`
	Connections     string      `json:"connections"`
	NumSeeders      string      `json:"numSeeders,omitempty"`
	ErrorMessage    string      `json:"errorMessage"`
	Dir             string      `json:"dir"`
	Files           []Aria2File `json:"files"`
}

type Aria2File struct {
	Index           string `json:"index"`
	Path            string `json:"path"`
	Length          string `json:"length"`
	CompletedLength string `json:"completedLength"`
	Selected        string `json:"selected"`
}

func (e *Engine) mapStatus(t *Aria2Task) *engine.DownloadStatus {
	total, _ := strconv.ParseInt(t.TotalLength, 10, 64)
	completed, _ := strconv.ParseInt(t.CompletedLength, 10, 64)
	speed, _ := strconv.ParseInt(t.DownloadSpeed, 10, 64)
	conn, _ := strconv.Atoi(t.Connections)
	seeders, _ := strconv.Atoi(t.NumSeeders)

	eta := 0
	if speed > 0 && total > 0 {
		remaining := total - completed
		if remaining > 0 {
			eta = int(remaining / speed)
		}
	}

	status := t.Status
	filename := ""
	var files []engine.DownloadFileStatus
	if len(t.Files) > 0 {
		filename = filepath.Base(t.Files[0].Path)
		for _, f := range t.Files {
			idx, _ := strconv.Atoi(f.Index)
			length, _ := strconv.ParseInt(f.Length, 10, 64)
			files = append(files, engine.DownloadFileStatus{
				Index:    idx,
				Path:     f.Path,
				Size:     length,
				Selected: f.Selected == "true",
			})
		}
	}

	return &engine.DownloadStatus{
		ID:          t.Gid,
		Status:      status,
		Filename:    filename,
		Dir:         t.Dir,
		Size:        total,
		Downloaded:  completed,
		Speed:       speed,
		Connections: conn,
		Seeders:     seeders,
		Peers:       conn - seeders,
		Eta:         eta,
		Error:       t.ErrorMessage,
		Files:       files,
	}
}

func (e *Engine) fetchTasks(ctx context.Context) ([]*Aria2Task, error) {
	var allTasks []*Aria2Task

	// 1. tellActive
	res, err := e.client.Call(ctx, "aria2.tellActive")
	if err == nil {
		var tasks []*Aria2Task
		if err := json.Unmarshal(res, &tasks); err == nil {
			allTasks = append(allTasks, tasks...)
		}
	}

	// 2. tellWaiting
	res, err = e.client.Call(ctx, "aria2.tellWaiting", 0, 1000)
	if err == nil {
		var tasks []*Aria2Task
		if err := json.Unmarshal(res, &tasks); err == nil {
			allTasks = append(allTasks, tasks...)
		}
	}

	// 3. tellStopped
	res, err = e.client.Call(ctx, "aria2.tellStopped", 0, 1000)
	if err == nil {
		var tasks []*Aria2Task
		if err := json.Unmarshal(res, &tasks); err == nil {
			allTasks = append(allTasks, tasks...)
		}
	}

	return allTasks, nil
}

func (e *Engine) poll() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Track seen tasks to detect transitions to stopped/complete
	activeGids := make(map[string]bool)

	for range ticker.C {
		ctx := context.Background()
		tasks, err := e.fetchTasks(ctx)
		if err != nil {
			continue
		}

		currentActive := make(map[string]*Aria2Task)
		currentStopped := make(map[string]*Aria2Task)

		for _, t := range tasks {
			if t.Status == "active" {
				currentActive[t.Gid] = t
			} else if t.Status == "complete" || t.Status == "error" || t.Status == "removed" {
				currentStopped[t.Gid] = t
			}
		}

		e.mu.RLock()
		onProgress := e.onProgress
		onComplete := e.onComplete
		onError := e.onError
		e.mu.RUnlock()

		// 1. Report progress for active tasks
		for gid, t := range currentActive {
			activeGids[gid] = true
			if onProgress != nil {
				s := e.mapStatus(t)
				onProgress(gid, engine.Progress{
					Downloaded: s.Downloaded,
					Size:       s.Size,
					Speed:      s.Speed,
					ETA:        s.Eta,
					Seeders:    s.Seeders,
					Peers:      s.Peers,
				})

				// File progress
				for _, f := range t.Files {
					if f.Selected == "true" {
						fTotal, _ := strconv.ParseInt(f.Length, 10, 64)
						fCompleted, _ := strconv.ParseInt(f.CompletedLength, 10, 64)
						onProgress(fmt.Sprintf("%s:%s", gid, f.Index), engine.Progress{
							Downloaded: fCompleted,
							Size:       fTotal,
						})
					}
				}
			}
		}

		// 2. Detect completions/errors
		for gid := range activeGids {
			if _, active := currentActive[gid]; !active {
				// Task is no longer active, check if it stopped
				if t, stopped := currentStopped[gid]; stopped {
					e.mu.Lock()
					reported := e.reportedGids[gid]
					if !reported {
						e.reportedGids[gid] = true
						e.mu.Unlock()

						if t.Status == "complete" && onComplete != nil {
							onComplete(gid, e.getDownloadPath(t))
						} else if t.Status == "error" && onError != nil {
							onError(gid, fmt.Errorf("%s", t.ErrorMessage))
						}
					} else {
						e.mu.Unlock()
					}
					delete(activeGids, gid)
				} else {
					// Disappeared completely? (removed from result)
					// We treat this as "stopped/removed"
					delete(activeGids, gid)
				}
			}
		}

		// 3. Catch tasks that completed so fast they were never seen as active
		for gid, t := range currentStopped {
			e.mu.Lock()
			reported := e.reportedGids[gid]
			if !reported {
				e.reportedGids[gid] = true
				e.mu.Unlock()

				if t.Status == "complete" && onComplete != nil {
					onComplete(gid, e.getDownloadPath(t))
				} else if t.Status == "error" && onError != nil {
					onError(gid, fmt.Errorf("%s", t.ErrorMessage))
				}
			} else {
				e.mu.Unlock()
			}
		}
	}
}

func (e *Engine) getDownloadPath(t *Aria2Task) string {
	if len(t.Files) == 0 {
		return t.Dir
	}
	path := t.Files[0].Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(t.Dir, path)
	}

	rel, err := filepath.Rel(t.Dir, path)
	if err == nil {
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) > 1 {
			return filepath.Join(t.Dir, parts[0])
		}
	}
	return path
}
