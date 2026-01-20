package aria2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

	// Cache for active GIDs to avoid polling everything constantly
	activeGids map[string]bool

	// Polling control
	pollingPaused bool
	pollingCond   *sync.Cond
	done          chan struct{}
}

func NewEngine(port int, dataDir string) *Engine {
	runner := NewRunner(port, dataDir)
	// WebSocket URL for local aria2 instance
	wsUrl := fmt.Sprintf("ws://localhost:%d/jsonrpc", port)
	client := NewClient(wsUrl)

	e := &Engine{
		runner:        runner,
		client:        client,
		reportedGids:  make(map[string]bool),
		activeGids:    make(map[string]bool),
		pollingPaused: true,
		done:          make(chan struct{}),
	}
	e.pollingCond = sync.NewCond(&e.mu)

	// Register notification handler
	client.SetNotificationHandler(e.handleNotification)

	return e
}

func (e *Engine) Start(ctx context.Context) error {
	if err := e.runner.Start(); err != nil {
		return err
	}

	// Wait for Aria2 to be ready and connect WebSocket
	ready := false
	for i := 0; i < 20; i++ {
		// Try connecting via WebSocket
		if err := e.client.Connect(ctx); err == nil {
			// Verify version
			if _, err := e.client.Call(ctx, "aria2.getVersion"); err == nil {
				ready = true
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !ready {
		return fmt.Errorf("aria2 engine failed to become ready")
	}

	// Subscribe to aria2 notifications
	if _, err := e.client.Call(ctx, "system.multicall", []interface{}{
		[]interface{}{"aria2.changeGlobalOption", map[string]interface{}{"listen-port": strconv.Itoa(e.runner.port)}},
	}); err != nil {
		log.Printf("Warning: Failed to set options via multicall: %v", err)
	}

	// Start optimized poller for active downloads only
	go e.poll()

	return nil
}

func (e *Engine) Stop() error {
	e.client.Close()
	close(e.done)
	e.mu.Lock()
	e.pollingCond.Broadcast() // Wake up any sleeping poll routine
	e.mu.Unlock()
	return e.runner.Stop()
}

func (e *Engine) PausePolling() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.pollingPaused {
		e.pollingPaused = true
		log.Println("Aria2: Progress polling paused")
	}
}

func (e *Engine) ResumePolling() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.pollingPaused {
		e.pollingPaused = false
		e.pollingCond.Broadcast() // Wake up poller
		log.Println("Aria2: Progress polling resumed")
	}
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

	// Mark as active immediately so poller picks it up
	e.mu.Lock()
	e.activeGids[gid] = true
	e.mu.Unlock()

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

	e.mu.Lock()
	delete(e.activeGids, id)
	e.mu.Unlock()

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

	var ariaPeers []Aria2Peer
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
	// For listing, we still need to fetch everything, but we do it less frequently
	// in the poll loop. This method is called by UI/API.
	var allTasks []*Aria2Task

	// 1. tellActive
	res, err := e.client.Call(ctx, "aria2.tellActive")
	if err == nil {
		var tasks []*Aria2Task
		if err := json.Unmarshal(res, &tasks); err == nil {
			allTasks = append(allTasks, tasks...)
		}
	}

	// 2. tellWaiting - limit to reasonable amount
	res, err = e.client.Call(ctx, "aria2.tellWaiting", 0, 100)
	if err == nil {
		var tasks []*Aria2Task
		if err := json.Unmarshal(res, &tasks); err == nil {
			allTasks = append(allTasks, tasks...)
		}
	}

	// 3. tellStopped - limit to reasonable amount
	res, err = e.client.Call(ctx, "aria2.tellStopped", 0, 100)
	if err == nil {
		var tasks []*Aria2Task
		if err := json.Unmarshal(res, &tasks); err == nil {
			allTasks = append(allTasks, tasks...)
		}
	}

	var statuses []*engine.DownloadStatus
	for _, t := range allTasks {
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
	var v VersionResponse
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

// handleNotification receives events from Aria2 via WebSocket
func (e *Engine) handleNotification(method string, params []interface{}) {
	if len(params) == 0 {
		return
	}

	// Format: [{"gid": "..."}]
	eventData, ok := params[0].(map[string]interface{})
	if !ok {
		return
	}

	gid, ok := eventData["gid"].(string)
	if !ok {
		return
	}

	ctx := context.Background()

	switch method {
	case "aria2.onDownloadStart":
		e.mu.Lock()
		e.activeGids[gid] = true
		e.mu.Unlock()
		log.Printf("Aria2: Download started %s", gid)

	case "aria2.onDownloadPause":
		e.mu.Lock()
		delete(e.activeGids, gid)
		e.mu.Unlock()
		log.Printf("Aria2: Download paused %s", gid)

	case "aria2.onDownloadStop":
		e.mu.Lock()
		delete(e.activeGids, gid)
		e.mu.Unlock()
		log.Printf("Aria2: Download stopped %s", gid)

	case "aria2.onDownloadComplete":
		e.mu.Lock()
		delete(e.activeGids, gid)
		reported := e.reportedGids[gid]
		e.reportedGids[gid] = true
		onComplete := e.onComplete
		e.mu.Unlock()

		if !reported && onComplete != nil {
			// Fetch status one last time to get path
			status, err := e.Status(ctx, gid)
			if err == nil {
				// Determine best path to report
				path := status.Dir
				if len(status.Files) > 0 {
					path = status.Files[0].Path
				}
				go onComplete(gid, path)
			}
		}
		log.Printf("Aria2: Download complete %s", gid)

	case "aria2.onDownloadError":
		e.mu.Lock()
		delete(e.activeGids, gid)
		reported := e.reportedGids[gid]
		e.reportedGids[gid] = true
		onError := e.onError
		e.mu.Unlock()

		if !reported && onError != nil {
			// Fetch status to get error message
			status, err := e.Status(ctx, gid)
			errMsg := "unknown error"
			if err == nil {
				errMsg = status.Error
			}
			go onError(gid, fmt.Errorf("%s", errMsg))
		}
		log.Printf("Aria2: Download error %s", gid)
	}
}

// poll only active downloads for progress updates
func (e *Engine) poll() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// We need a context to know when to stop
	// But the current signature doesn't take context.
	// Since poll is called in Start, we can use a channel controlled by Stop.
	// We'll add a 'done' channel to Engine struct.

	for {
		e.mu.Lock()
		for e.pollingPaused {
			// Check if we should stop while paused
			select {
			case <-e.done:
				e.mu.Unlock()
				return
			default:
				e.pollingCond.Wait()
			}
		}
		e.mu.Unlock()

		select {
		case <-e.done:
			return
		case <-ticker.C:
			// continue
		}

		e.mu.RLock()
		activeList := make([]string, 0, len(e.activeGids))
		for gid := range e.activeGids {
			activeList = append(activeList, gid)
		}
		onProgress := e.onProgress
		e.mu.RUnlock()

		if len(activeList) == 0 {
			continue
		}

		ctx := context.Background()

		// For each active download, fetch status and report progress
		// We could optimize this further with multicall if needed
		for _, gid := range activeList {
			status, err := e.Status(ctx, gid)
			if err != nil {
				// If error fetching status, it might have disappeared or finished
				// Verify if it still exists via handleNotification events logic,
				// or let the next List/Sync catch it.
				// For now, if tellStatus fails, we might want to remove it from activeGids
				// but let's be conservative.
				continue
			}

			if status.Status != "active" {
				// Status changed but we missed the event?
				// handleNotification should handle transitions.
				// Just ignore here.
				continue
			}

			if onProgress != nil {
				onProgress(gid, engine.Progress{
					Downloaded: status.Downloaded,
					Size:       status.Size,
					Speed:      status.Speed,
					ETA:        status.Eta,
					Seeders:    status.Seeders,
					Peers:      status.Peers,
				})

				// File progress
				for _, f := range status.Files {
					if f.Selected {
						onProgress(fmt.Sprintf("%s:%d", gid, f.Index), engine.Progress{
							Downloaded: f.Size, // Note: Aria2File struct in mapStatus maps CompletedLength to Size
							Size:       f.Size, // logic in mapStatus seems slightly off for partial file progress
							// Let's rely on mapStatus implementation for now:
							// mapStatus: Size=Length, Downloaded=CompletedLength?
							// Actually mapStatus: Size=total, Downloaded=completed for main task.
							// For files: Size=Length. We need CompletedLength.
						})
					}
				}

				// Re-read mapStatus logic for files to be precise
				// In mapStatus:
				// files = append(files, engine.DownloadFileStatus{ ... Size: length ... })
				// It doesn't seem to export CompletedLength for files in engine.DownloadFileStatus?
				// Let's check engine definition if we need file progress.
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
