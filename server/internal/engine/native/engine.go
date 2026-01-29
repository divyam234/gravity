package native

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"gravity/internal/engine"
	"gravity/internal/model"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	_ "github.com/rclone/rclone/backend/http"
	_ "github.com/rclone/rclone/backend/local"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/accounting"
	"github.com/rclone/rclone/fs/operations"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"
)

type NativeEngine struct {
	torrentClient *torrent.Client
	ctx           context.Context
	cancel        context.CancelFunc
	dataDir       string
	storage       *DynamicStorage
	logger        *zap.Logger

	onProgress func(id string, progress engine.Progress)
	onComplete func(id string, filePath string)
	onError    func(id string, err error)

	activeTasks sync.Map
	mu          sync.RWMutex
	settings    *model.Settings

	pollingCond *sync.Cond
	done        chan struct{}
}

type taskType string

const (
	taskTypeTorrent taskType = "torrent"
	taskTypeRclone  taskType = "rclone"
)

type task struct {
	id       string
	taskType taskType
	url      string
	dir      string
	filename string
	size     int64
	headers  map[string]string

	stats *accounting.StatsInfo

	tDownload *torrent.Torrent

	proxyURL      string
	proxyUser     string
	proxyPassword string

	split int

	lastRead    int64
	lastWrite   int64
	lastChecked time.Time
	downSpeed   int64
	upSpeed     int64

	cancel context.CancelFunc
	done   chan struct{}
}

func NewNativeEngine(dataDir string, l *zap.Logger) *NativeEngine {
	e := &NativeEngine{
		dataDir: dataDir,
		done:    make(chan struct{}),
		logger:  l.With(zap.String("engine", "native")),
	}
	e.pollingCond = sync.NewCond(&e.mu)
	return e
}

func (e *NativeEngine) Start(ctx context.Context) error {
	e.ctx, e.cancel = context.WithCancel(ctx)

	// Start Rclone accounting
	accounting.Start(e.ctx)

	cfg := torrent.NewDefaultClientConfig()

	e.mu.RLock()
	s := e.settings
	e.mu.RUnlock()

	// Set Listen Port from settings
	listenPort := 0 // Random port by default
	if s != nil && s.Torrent.ListenPort > 0 {
		listenPort = s.Torrent.ListenPort
	}
	cfg.ListenPort = listenPort

	// Use the downloads subdirectory
	cfg.DataDir = filepath.Join(e.dataDir, ".metadata")
	defaultDownloadsDir := filepath.Join(e.dataDir, "downloads")
	os.MkdirAll(defaultDownloadsDir, 0755)
	os.MkdirAll(cfg.DataDir, 0755)

	// Use .metadata dir for completion DB
	e.storage = NewDynamicStorage(defaultDownloadsDir, cfg.DataDir)
	cfg.DefaultStorage = e.storage

	if s != nil && len(s.Network.Proxies) > 0 {
		proxyURL, err := url.Parse(s.Network.Proxies[0].URL)
		if err == nil {
			dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
			if err == nil {
				cfg.HTTPProxy = http.ProxyURL(proxyURL)
				cfg.TrackerDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				}
			}
		}
	}

	tc, err := torrent.NewClient(cfg)
	if err != nil {
		e.logger.Error("failed to start torrent client", zap.Error(err))
		return fmt.Errorf("failed to start torrent client: %w", err)
	}
	e.torrentClient = tc
	go e.poll()
	return nil
}

func (e *NativeEngine) Stop() error {
	var errs []error

	if e.storage != nil {
		if err := e.storage.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close storage: %w", err))
		}
	}
	if e.done != nil {
		select {
		case <-e.done:
		default:
			close(e.done)
		}
	}
	if e.cancel != nil {
		e.cancel()
	}
	if e.torrentClient != nil {
		e.torrentClient.Close()
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (e *NativeEngine) Add(ctx context.Context, url string, opts engine.DownloadOptions) (string, error) {
	id := fmt.Sprintf("nat_%d", time.Now().UnixNano())
	t := &task{
		id:       id,
		url:      url,
		dir:      opts.DownloadDir,
		filename: opts.Filename,
		size:     opts.Size,
		headers:  opts.Headers,
		done:     make(chan struct{}),
		split:    0,
	}

	if opts.Split != nil {
		t.split = *opts.Split
	}

	if strings.HasPrefix(url, "magnet:") || strings.HasSuffix(url, ".torrent") || opts.TorrentData != "" {
		t.taskType = taskTypeTorrent

		// Register custom storage path
		if opts.DownloadDir != "" {
			if opts.TorrentData != "" {
				mi, _ := metainfo.Load(strings.NewReader(opts.TorrentData))
				if mi != nil {
					e.storage.Register(mi.HashInfoBytes().HexString(), opts.DownloadDir)
				}
			} else if strings.HasPrefix(url, "magnet:") {
				m, err := metainfo.ParseMagnetUri(url)
				if err == nil {
					e.storage.Register(m.InfoHash.HexString(), opts.DownloadDir)
				}
			}
		}

		var dl *torrent.Torrent
		var err error
		if opts.TorrentData != "" {
			mi, err := metainfo.Load(strings.NewReader(opts.TorrentData))
			if err != nil {
				return "", err
			}
			dl, err = e.torrentClient.AddTorrent(mi)
		} else if strings.HasPrefix(url, "magnet:") {
			dl, err = e.torrentClient.AddMagnet(url)
		}
		if err != nil {
			return "", err
		}

		go func(torrentTask *torrent.Torrent, taskId string) {
			timeout := 60 * time.Second
			e.mu.RLock()
			if e.settings != nil && e.settings.Download.ConnectTimeout > 0 {
				timeout = time.Duration(e.settings.Download.ConnectTimeout) * time.Second
			}
			onError := e.onError
			ctx := e.ctx // Capture context while holding lock
			e.mu.RUnlock()

			select {
			case <-torrentTask.GotInfo():
				if len(opts.SelectedFiles) > 0 {
					files := torrentTask.Files()
					for i, f := range files {
						selected := false
						// Check if current file index (1-based) is in requested list
						for _, targetIdx := range opts.SelectedFiles {
							if i+1 == targetIdx {
								selected = true
								break
							}
						}
						if selected {
							f.Download()
						}
					}
				} else {
					torrentTask.DownloadAll()
				}
			case <-time.After(timeout):
				if onError != nil {
					onError(taskId, fmt.Errorf("metadata resolution timeout"))
				}
				if ctx != nil {
					e.Remove(ctx, taskId)
				}
			}
		}(dl, id)

		t.tDownload = dl
	} else {
		t.taskType = taskTypeRclone
		taskCtx, cancel := context.WithCancel(e.ctx)
		t.cancel = cancel
		go e.runRcloneDownload(taskCtx, t)
	}

	e.activeTasks.Store(id, t)
	e.pollingCond.Broadcast()
	return id, nil
}

func (e *NativeEngine) runRcloneDownload(ctx context.Context, t *task) {
	// Capture callbacks under lock for thread-safe access
	e.mu.RLock()
	onError := e.onError
	onProgress := e.onProgress
	onComplete := e.onComplete
	s := e.settings
	e.mu.RUnlock()

	defer func() {
		if r := recover(); r != nil {
			e.logger.Error("engine panic during rclone download", zap.Any("panic", r))
			if onError != nil {
				onError(t.id, fmt.Errorf("engine panic: %v", r))
			}
		}
	}()

	e.logger.Debug("starting rclone download", zap.String("id", t.id), zap.String("url", t.url))

	// Create a fresh config for this context to support per-request headers/proxy
	ctx, ci := fs.AddConfig(ctx)

	if s != nil {
		ci.MultiThreadStreams = s.Download.Split
		ci.ConnectTimeout = fs.Duration(time.Duration(s.Download.ConnectTimeout) * time.Second)
		ci.InsecureSkipVerify = !s.Download.CheckCertificate
		if s.Download.MaxTries > 0 {
			ci.LowLevelRetries = s.Download.MaxTries
		}
	}

	if t.proxyURL != "" {
		ci.Proxy = t.proxyURL
	}

	if t.split > 0 {
		ci.MultiThreadStreams = t.split
		if t.split > 1 {
			ci.MultiThreadCutoff = 0 // Force multi-thread
		}
	}
	ci.MultiThreadResume = true
	ci.MultiThreadChunkSize = 16 * 1024 * 1024

	// Set headers in rclone config
	if len(t.headers) > 0 {
		for k, v := range t.headers {
			ci.Headers = append(ci.Headers, &fs.HTTPOption{
				Key:   k,
				Value: v,
			})
		}
	}

	accCtx := accounting.WithStatsGroup(ctx, t.id)
	t.stats = accounting.StatsGroup(accCtx, t.id)

	dstFs, err := fs.NewFs(accCtx, t.dir)
	if err != nil {
		if onError != nil {
			onError(t.id, err)
		}
		return
	}

	destObj, err := operations.CopyURLMulti(accCtx, dstFs, t.filename, t.url, false)

	if err != nil {
		e.logger.Error("rclone download failed", zap.String("id", t.id), zap.Error(err))
		if onError != nil {
			onError(t.id, err)
		}

	} else {

		finalPath := filepath.Join(destObj.Fs().Root(), destObj.Remote())

		e.logger.Debug("download complete", zap.String("path", finalPath))

		if onProgress != nil {
			onProgress(t.id, engine.Progress{Downloaded: t.size, Size: t.size, Speed: 0})
		}
		if onComplete != nil {
			onComplete(t.id, finalPath)
		}
	}

	e.Remove(e.ctx, t.id)

}

func (e *NativeEngine) poll() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		e.mu.Lock()
		empty := true
		e.activeTasks.Range(func(_, _ any) bool { empty = false; return false })
		for empty {
			select {
			case <-e.done:
				e.mu.Unlock()
				return
			default:
				e.pollingCond.Wait()
			}
			e.activeTasks.Range(func(_, _ any) bool { empty = false; return false })
		}
		e.mu.Unlock()

		select {
		case <-e.done:
			return
		case <-ticker.C:
			e.reportProgress()
		}
	}
}

func (e *NativeEngine) reportProgress() {
	e.mu.RLock()
	onProgress := e.onProgress
	onComplete := e.onComplete
	e.mu.RUnlock()

	if onProgress == nil {
		return
	}
	now := time.Now()

	e.activeTasks.Range(func(key, value any) bool {
		t := value.(*task)
		if t.taskType == taskTypeTorrent {
			if t.tDownload.Info() == nil {
				onProgress(t.id, engine.Progress{
					Peers: t.tDownload.Stats().ActivePeers,
				})
				return true
			}
			st := t.tDownload.Stats()
			if !t.lastChecked.IsZero() {
				diff := now.Sub(t.lastChecked).Seconds()
				if diff > 0 {
					read := st.BytesReadData.Int64()
					t.downSpeed = int64(float64(read-t.lastRead) / diff)
					t.lastRead = read
				}
			} else {
				t.lastRead = st.BytesReadData.Int64()
			}
			t.lastChecked = now

			onProgress(t.id, engine.Progress{
				Downloaded: t.tDownload.BytesCompleted(),
				Size:       t.tDownload.Length(),
				Speed:      t.downSpeed,
				Peers:      st.ActivePeers,
				Seeders:    st.ConnectedSeeders,
			})

			if t.tDownload.BytesCompleted() >= t.tDownload.Length() && t.tDownload.Length() > 0 {
				if onComplete != nil {
					onComplete(t.id, filepath.Join(t.dir, t.tDownload.Name()))
				}
				e.Remove(context.Background(), t.id)
			}
		} else {
			if t.stats != nil {
				current := t.stats.GetBytes()
				if !t.lastChecked.IsZero() {
					diff := now.Sub(t.lastChecked).Seconds()
					if diff > 0 {
						t.downSpeed = int64(float64(current-t.lastRead) / diff)
						t.lastRead = current
					}
				} else {
					t.lastRead = current
				}
				t.lastChecked = now

				eta := 0
				if t.downSpeed > 0 && t.size > 0 {
					remaining := t.size - current
					if remaining > 0 {
						eta = int(remaining / t.downSpeed)
					}
				}

				onProgress(t.id, engine.Progress{
					Downloaded: current,
					Size:       t.size,
					Speed:      t.downSpeed,
					ETA:        eta,
				})
			}
		}
		return true
	})
}

func (e *NativeEngine) Status(ctx context.Context, id string) (*engine.DownloadStatus, error) {
	val, ok := e.activeTasks.Load(id)
	if !ok {
		return nil, fmt.Errorf("task not found")
	}
	t := val.(*task)

	status := &engine.DownloadStatus{
		ID:       id,
		URL:      t.url,
		Filename: t.filename,
		Dir:      t.dir,
	}

	if t.taskType == taskTypeTorrent {
		if t.tDownload.Info() != nil {
			status.Status = "active"
			status.Downloaded = t.tDownload.BytesCompleted()
			status.Size = t.tDownload.Length()
			status.Speed = t.downSpeed
			status.Peers = t.tDownload.Stats().ActivePeers
			status.Seeders = t.tDownload.Stats().ConnectedSeeders

			if status.Speed > 0 && status.Size > 0 {
				rem := status.Size - status.Downloaded
				if rem > 0 {
					status.Eta = int(rem / status.Speed)
				}
			}
		} else {
			status.Status = "resolving"
		}
	} else {
		status.Status = "active"
		status.Speed = t.downSpeed
		if t.stats != nil {
			status.Downloaded = t.stats.GetBytes()
		} else {
			status.Downloaded = t.lastRead
		}
		status.Size = t.size

		if status.Speed > 0 && status.Size > 0 {
			rem := status.Size - status.Downloaded
			if rem > 0 {
				status.Eta = int(rem / status.Speed)
			}
		}
	}
	return status, nil
}

func (e *NativeEngine) List(ctx context.Context) ([]*engine.DownloadStatus, error) {
	var list []*engine.DownloadStatus
	e.activeTasks.Range(func(key, value any) bool {
		s, _ := e.Status(ctx, key.(string))
		list = append(list, s)
		return true
	})
	return list, nil
}

func (e *NativeEngine) Pause(ctx context.Context, id string) error  { return nil }
func (e *NativeEngine) Resume(ctx context.Context, id string) error { return nil }
func (e *NativeEngine) Cancel(ctx context.Context, id string) error {
	if val, ok := e.activeTasks.Load(id); ok {
		t := val.(*task)
		if t.cancel != nil {
			t.cancel()
		}
	}
	return nil
}
func (e *NativeEngine) Remove(ctx context.Context, id string) error {
	e.activeTasks.Delete(id)
	return nil
}

func (e *NativeEngine) GetPeers(ctx context.Context, id string) ([]engine.DownloadPeer, error) {
	val, ok := e.activeTasks.Load(id)
	if !ok {
		return nil, nil
	}
	t := val.(*task)
	if t.taskType != taskTypeTorrent || t.tDownload == nil || t.tDownload.Info() == nil {
		return nil, nil
	}

	var peers []engine.DownloadPeer
	for _, pc := range t.tDownload.PeerConns() {
		peers = append(peers, engine.DownloadPeer{
			IP: pc.RemoteAddr.String(),
		})
	}
	return peers, nil
}

func (e *NativeEngine) Sync(ctx context.Context) error { return nil }
func (e *NativeEngine) Configure(ctx context.Context, s *model.Settings) error {
	e.mu.Lock()
	e.settings = s
	e.mu.Unlock()
	return nil
}
func (e *NativeEngine) Version(ctx context.Context) (string, error) {
	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/anacrolix/torrent" {
				return dep.Version, nil
			}
		}
	}
	return "native-hybrid-1.0", nil
}
func (e *NativeEngine) OnProgress(h func(string, engine.Progress)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onProgress = h
}
func (e *NativeEngine) OnComplete(h func(string, string)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onComplete = h
}
func (e *NativeEngine) OnError(h func(string, error)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onError = h
}

func (e *NativeEngine) GetMagnetFiles(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
	if e.torrentClient == nil {
		return nil, fmt.Errorf("torrent client not initialized")
	}

	t, err := e.torrentClient.AddMagnet(magnet)
	if err != nil {
		return nil, err
	}
	defer t.Drop()

	select {
	case <-t.GotInfo():
		// Got info, continue
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(60 * time.Second):
		return nil, fmt.Errorf("timeout waiting for magnet metadata")
	}

	info := &model.MagnetInfo{
		Source: "native",
		Name:   t.Name(),
		Hash:   t.InfoHash().String(),
		Size:   t.Length(),
	}
	for i, f := range t.Files() {
		info.Files = append(info.Files, &model.MagnetFile{
			ID:    fmt.Sprintf("%d", i),
			Name:  filepath.Base(f.Path()),
			Path:  f.Path(),
			Size:  f.Length(),
			Index: i,
		})
	}
	return info, nil
}

func (e *NativeEngine) GetTorrentFiles(ctx context.Context, torrentBase64 string) (*model.MagnetInfo, error) {
	mi, err := metainfo.Load(strings.NewReader(torrentBase64))
	if err != nil {
		return nil, fmt.Errorf("failed to parse torrent data: %w", err)
	}

	info, err := mi.UnmarshalInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal torrent info: %w", err)
	}

	magnetInfo := &model.MagnetInfo{
		Source: "native",
		Name:   info.Name,
		Hash:   mi.HashInfoBytes().String(),
		Size:   info.TotalLength(),
	}

	for i, f := range info.UpvertedFiles() {
		magnetInfo.Files = append(magnetInfo.Files, &model.MagnetFile{
			ID:    fmt.Sprintf("%d", i),
			Name:  filepath.Base(strings.Join(f.Path, "/")),
			Path:  strings.Join(f.Path, "/"),
			Size:  f.Length,
			Index: i,
		})
	}

	return magnetInfo, nil
}

func (e *NativeEngine) AddMagnetWithSelection(ctx context.Context, magnet string, selectedIndexes []string, opts engine.DownloadOptions) (string, error) {
	return e.Add(ctx, magnet, opts)
}
