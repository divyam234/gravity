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
	"gravity/internal/model"

	"github.com/anacrolix/torrent"
	"go.uber.org/zap"
)

type Engine struct {
	runner         *Runner
	client         *Client
	metadataClient *torrent.Client
	dataDir        string
	logger         *zap.Logger

	settings *model.Settings
	appCtx   context.Context

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

func NewEngine(port int, dataDir string, l *zap.Logger) *Engine {
	logger := l.With(zap.String("engine", "aria2"))
	runner := NewRunner(port, dataDir, logger)
	// WebSocket URL for local aria2 instance
	wsUrl := fmt.Sprintf("ws://localhost:%d/jsonrpc", port)
	client := NewClient(wsUrl)

	e := &Engine{
		dataDir:       dataDir,
		runner:        runner,
		client:        client,
		logger:        logger,
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
	e.appCtx = ctx
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
	if _, err := e.client.Call(ctx, "system.multicall", []any{
		[]any{"aria2.changeGlobalOption", map[string]any{"listen-port": strconv.Itoa(e.runner.port)}},
	}); err != nil {
		e.logger.Warn("failed to set options via multicall", zap.Error(err))
	}

	// Start metadata client
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = filepath.Join(e.dataDir, ".metadata")
	cfg.NoUpload = true
	cfg.ListenPort = 0 // Random port
	tc, err := torrent.NewClient(cfg)
	if err != nil {
		e.logger.Warn("failed to start metadata client", zap.Error(err))
	} else {
		e.metadataClient = tc
	}

	// Start optimized poller for active downloads only
	e.mu.Lock()
	e.pollingPaused = false
	e.mu.Unlock()
	go e.poll()

	return nil
}

func (e *Engine) Stop() error {
	if e.metadataClient != nil {
		e.metadataClient.Close()
	}
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
		e.logger.Debug("progress polling paused")
	}
}

func (e *Engine) ResumePolling() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.pollingPaused {
		e.pollingPaused = false
		e.pollingCond.Broadcast() // Wake up poller
		e.logger.Debug("progress polling resumed")
	}
}

func (e *Engine) Add(ctx context.Context, url string, opts engine.DownloadOptions) (string, error) {
	ariaOpts := make(map[string]any)
	if opts.ID != "" {
		ariaOpts["gid"] = opts.ID
	}
	if opts.Filename != "" {
		ariaOpts["out"] = opts.Filename
	}
	if opts.DownloadDir != "" {
		ariaOpts["dir"] = opts.DownloadDir
	}
	if len(opts.Headers) > 0 {
		var headers []string
		for k, v := range opts.Headers {
			headers = append(headers, fmt.Sprintf("%s: %s", k, v))
		}
		ariaOpts["header"] = headers
	}
	if opts.Split != nil && *opts.Split > 0 {
		ariaOpts["split"] = strconv.Itoa(*opts.Split)
	}
	if opts.MaxTries != nil && *opts.MaxTries > 0 {
		ariaOpts["max-tries"] = strconv.Itoa(*opts.MaxTries)
	}
	if opts.UserAgent != nil && *opts.UserAgent != "" {
		ariaOpts["user-agent"] = *opts.UserAgent
	}

	// Proxies
	if len(opts.Proxies) > 0 {
		// aria2 takes a single proxy via all-proxy
		ariaOpts["all-proxy"] = opts.Proxies[0].URL
	} else {
		// Auto-configure proxy based on settings
		e.mu.RLock()
		s := e.settings
		e.mu.RUnlock()

		if s != nil && len(s.Network.Proxies) > 0 {
			ariaOpts["all-proxy"] = s.Network.Proxies[0].URL
		}
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
	var params []any

	if strings.HasPrefix(url, "magnet:") && opts.TorrentData == "" {
		// Resolve metadata via native lib to avoid Aria2 dual-ID
		e.logger.Debug("resolving magnet metadata via native lib", zap.String("url", url))
		b64, err := e.resolveMetadata(ctx, url)
		if err == nil {
			opts.TorrentData = b64
		} else {
			e.logger.Warn("failed to resolve metadata natively, falling back to aria2", zap.Error(err))
		}
	}

	if opts.TorrentData != "" {
		method = "aria2.addTorrent"
		params = []any{opts.TorrentData, []any{}, ariaOpts}
	} else {
		method = "aria2.addUri"
		params = []any{[]string{url}, ariaOpts}
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
	e.pollingCond.Broadcast()
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

func (e *Engine) Configure(ctx context.Context, settings *model.Settings) error {
	e.mu.Lock()
	e.settings = settings
	e.mu.Unlock()

	if settings == nil {
		return nil
	}

	ariaOpts := make(map[string]any)

	// Download
	if settings.Download.DownloadDir != "" {
		ariaOpts["dir"] = settings.Download.DownloadDir
	}
	if settings.Download.MaxConcurrentDownloads > 0 {
		ariaOpts["max-concurrent-downloads"] = strconv.Itoa(settings.Download.MaxConcurrentDownloads)
	}
	if settings.Download.MaxDownloadSpeed != "" {
		ariaOpts["max-overall-download-limit"] = settings.Download.MaxDownloadSpeed
	}
	if settings.Download.MaxUploadSpeed != "" {
		ariaOpts["max-overall-upload-limit"] = settings.Download.MaxUploadSpeed
	}
	if settings.Download.MaxConnectionPerServer > 0 {
		ariaOpts["max-connection-per-server"] = strconv.Itoa(settings.Download.MaxConnectionPerServer)
	}
	if settings.Download.Split > 0 {
		ariaOpts["split"] = strconv.Itoa(settings.Download.Split)
	}
	if settings.Download.UserAgent != "" {
		ariaOpts["user-agent"] = settings.Download.UserAgent
	}
	if settings.Download.ConnectTimeout > 0 {
		ariaOpts["connect-timeout"] = strconv.Itoa(settings.Download.ConnectTimeout)
	}
	if settings.Download.MaxTries > 0 {
		ariaOpts["max-tries"] = strconv.Itoa(settings.Download.MaxTries)
	}
	ariaOpts["check-integrity"] = strconv.FormatBool(settings.Download.CheckCertificate)
	ariaOpts["file-allocation"] = "falloc" // Default to falloc for performance
	if settings.Download.PreAllocateSpace {
		ariaOpts["file-allocation"] = "prealloc"
	}
	if settings.Download.DiskCache != "" {
		ariaOpts["disk-cache"] = settings.Download.DiskCache
	}
	if settings.Download.MinSplitSize != "" {
		ariaOpts["min-split-size"] = settings.Download.MinSplitSize
	}

	// Network
	if len(settings.Network.Proxies) > 0 {
		ariaOpts["all-proxy"] = settings.Network.Proxies[0].URL
	} else {
		ariaOpts["all-proxy"] = ""
	}
	if settings.Network.InterfaceBinding != "" {
		ariaOpts["interface"] = settings.Network.InterfaceBinding
	}

	// Torrent
	if settings.Torrent.SeedRatio != "" {
		ariaOpts["seed-ratio"] = settings.Torrent.SeedRatio
	}
	if settings.Torrent.SeedTime > 0 {
		ariaOpts["seed-time"] = strconv.Itoa(settings.Torrent.SeedTime)
	}
	if settings.Torrent.ListenPort > 0 {
		ariaOpts["listen-port"] = strconv.Itoa(settings.Torrent.ListenPort)
	}
	if settings.Network.TCPPortRange != "" {
		ariaOpts["peer-id-prefix"] = "A2-" // Just a marker
		// aria2 uses --listen-port for range
		ariaOpts["listen-port"] = settings.Network.TCPPortRange
	}
	ariaOpts["enable-dht"] = strconv.FormatBool(settings.Torrent.EnableDht)
	ariaOpts["bt-enable-pex"] = strconv.FormatBool(settings.Torrent.EnablePex)
	ariaOpts["bt-enable-lpd"] = strconv.FormatBool(settings.Torrent.EnableLpd)
	ariaOpts["bt-encryption"] = settings.Torrent.Encryption
	if settings.Torrent.MaxPeers > 0 {
		ariaOpts["bt-max-peers"] = strconv.Itoa(settings.Torrent.MaxPeers)
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

func (e *Engine) GetRunner() *Runner {
	return e.runner
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
	FollowedBy      []string    `json:"followedBy"`
	Seeder          string      `json:"seeder"`
	BitTorrent      *struct {
		Info *struct {
			Name string `json:"name"`
		} `json:"info"`
	} `json:"bittorrent,omitempty"`
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

	if t.BitTorrent != nil && (t.BitTorrent.Info == nil || t.BitTorrent.Info.Name == "") {
		status = "resolving"
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
		FollowedBy:  t.FollowedBy,
		IsSeeder:    t.Seeder == "true",
	}
}

// handleNotification receives events from Aria2 via WebSocket
func (e *Engine) handleNotification(method string, params []any) {
	if len(params) == 0 {
		return
	}

	// Format: [{"gid": "..."}]
	eventData, ok := params[0].(map[string]any)
	if !ok {
		return
	}

	gid, ok := eventData["gid"].(string)
	if !ok {
		return
	}

	ctx := e.appCtx
	if ctx == nil {
		ctx = context.Background()
	}

	switch method {
	case "aria2.onDownloadStart":
		e.mu.Lock()
		e.activeGids[gid] = true
		e.pollingCond.Broadcast()
		e.mu.Unlock()
		e.logger.Debug("download started", zap.String("gid", gid))

	case "aria2.onDownloadPause":
		e.mu.Lock()
		delete(e.activeGids, gid)
		e.mu.Unlock()
		e.logger.Debug("download paused", zap.String("gid", gid))

	case "aria2.onDownloadStop":
		e.mu.Lock()
		delete(e.activeGids, gid)
		e.mu.Unlock()
		e.logger.Debug("download stopped", zap.String("gid", gid))
		// Service layer must handle removal

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
		e.logger.Debug("download complete", zap.String("gid", gid))
		// Service layer must handle removal

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
		e.logger.Debug("download error", zap.String("gid", gid))
		// Service layer must handle removal
	}
}

// poll only active downloads for progress updates
func (e *Engine) poll() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var emptyCycles int

	for {
		e.mu.Lock()
		for e.pollingPaused || len(e.activeGids) == 0 {
			// Reset empty cycles when we go to sleep
			emptyCycles = 0
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
		onProgress := e.onProgress
		e.mu.RUnlock()

		if onProgress == nil {
			continue
		}

		ctx := e.appCtx
		if ctx == nil {
			ctx = context.Background()
		}

		// Optimize: Use tellActive to get all active downloads in one call
		res, err := e.client.Call(ctx, "aria2.tellActive")
		if err != nil {
			e.logger.Error("failed to poll active tasks", zap.Error(err))
			continue
		}

		var activeTasks []*Aria2Task
		if err := json.Unmarshal(res, &activeTasks); err != nil {
			e.logger.Error("failed to unmarshal active tasks", zap.Error(err))
			continue
		}

		// Cleanup Ghost Tasks:
		// If aria2 reports NO active tasks, but we think we have some (activeGids > 0),
		// it might mean we missed a Stop/Complete event.
		// We allow a few empty cycles (6 seconds) to account for startup race conditions
		// before forcibly clearing our active list to allow sleeping.
		if len(activeTasks) == 0 {
			emptyCycles++
			if emptyCycles >= 3 {
				e.mu.Lock()
				if len(e.activeGids) > 0 {
					e.logger.Warn("detected ghost tasks (idle for 6s), resetting active list")
					e.activeGids = make(map[string]bool)
				}
				e.mu.Unlock()
			}
		} else {
			emptyCycles = 0
		}

		for _, t := range activeTasks {
			// Convert Aria2 task to engine status
			status := e.mapStatus(t)

			onProgress(status.ID, engine.Progress{
				Downloaded: status.Downloaded,
				Size:       status.Size,
				Speed:      status.Speed,
				ETA:        status.Eta,
				Seeders:    status.Seeders,
				Peers:      status.Peers,
				IsSeeder:   status.IsSeeder,
			})

			// File progress
			for _, f := range t.Files {
				if f.Selected == "true" {
					completed, _ := strconv.ParseInt(f.CompletedLength, 10, 64)
					length, _ := strconv.ParseInt(f.Length, 10, 64)
					onProgress(fmt.Sprintf("%s:%s", status.ID, f.Index), engine.Progress{
						Downloaded: completed,
						Size:       length,
					})
				}
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
