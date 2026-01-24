package rclone

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	stdSync "sync"
	"time"

	"gravity/internal/engine"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/accounting"
	rclConfig "github.com/rclone/rclone/fs/config"
	_ "github.com/rclone/rclone/fs/operations"
	"github.com/rclone/rclone/fs/rc"
	_ "github.com/rclone/rclone/fs/sync"
	"github.com/rclone/rclone/vfs"
	"github.com/rclone/rclone/vfs/vfscommon"

	_ "github.com/rclone/rclone/backend/all"
)

type Engine struct {
	vfs        *vfs.VFS
	onProgress func(jobID string, progress engine.UploadProgress)
	onComplete func(jobID string)
	onError    func(jobID string, err error)

	activeJobs map[string]*job
	mu         stdSync.RWMutex
	cacheTTL   time.Duration
	appCtx     context.Context
}

type job struct {
	id      string
	trackID string
	size    int64
	done    chan struct{}
	cancel  context.CancelFunc
}

func NewEngine(ctx context.Context) *Engine {
	return &Engine{
		activeJobs: make(map[string]*job),
		cacheTTL:   5 * time.Minute,
		appCtx:     ctx,
	}
}

func (e *Engine) Start(ctx context.Context) error {
	log.Printf("Rclone: Using config file: %s", rclConfig.GetConfigPath())
	if err := SyncGravityRoot(); err != nil {
		return fmt.Errorf("failed to sync gravity root: %w", err)
	}

	f, err := fs.NewFs(ctx, GravityRootRemote+":")
	if err != nil {
		return fmt.Errorf("failed to create root fs: %w", err)
	}

	e.vfs = vfs.New(f, &vfscommon.Opt)

	go e.pollAccounting()

	return nil
}

func (e *Engine) Stop() error {
	return nil
}

func (e *Engine) List(ctx context.Context, virtualPath string) ([]engine.FileInfo, error) {
	node, err := e.vfs.Stat(virtualPath)
	if err != nil {
		return nil, err
	}

	if !node.IsDir() {
		return nil, fmt.Errorf("not a directory")
	}

	dir := node.(*vfs.Dir)
	entries, err := dir.ReadDirAll()
	if err != nil {
		return nil, err
	}

	var results []engine.FileInfo
	for _, entry := range entries {
		fType := engine.FileTypeFile
		if entry.IsDir() {
			fType = engine.FileTypeFolder
		}

		results = append(results, engine.FileInfo{
			Path:     path.Join(virtualPath, entry.Name()),
			Name:     entry.Name(),
			Size:     entry.Size(),
			ModTime:  entry.ModTime(),
			IsDir:    entry.IsDir(),
			Type:     fType,
			MimeType: fs.MimeTypeFromName(entry.Name()),
		})
	}

	return results, nil
}

func (e *Engine) Stat(ctx context.Context, virtualPath string) (*engine.FileInfo, error) {
	node, err := e.vfs.Stat(virtualPath)
	if err != nil {
		return nil, err
	}

	fType := engine.FileTypeFile
	if node.IsDir() {
		fType = engine.FileTypeFolder
	}

	return &engine.FileInfo{
		Path:     virtualPath,
		Name:     node.Name(),
		Size:     node.Size(),
		ModTime:  node.ModTime(),
		IsDir:    node.IsDir(),
		Type:     fType,
		MimeType: fs.MimeTypeFromName(node.Name()),
	}, nil
}

func (e *Engine) Open(ctx context.Context, virtualPath string) (engine.ReadSeekCloser, error) {
	file, err := e.vfs.OpenFile(virtualPath, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (e *Engine) Mkdir(ctx context.Context, virtualPath string) error {
	parent := path.Dir(virtualPath)
	name := path.Base(virtualPath)

	node, err := e.vfs.Stat(parent)
	if err != nil {
		return err
	}

	dir, ok := node.(*vfs.Dir)
	if !ok {
		return fmt.Errorf("parent is not a directory")
	}
	_, err = dir.Mkdir(name)
	return err
}

func (e *Engine) Delete(ctx context.Context, virtualPath string) error {
	node, err := e.vfs.Stat(virtualPath)
	if err != nil {
		return err
	}
	return node.Remove()
}

func (e *Engine) Rename(ctx context.Context, virtualPath, newName string) error {
	oldParent := path.Dir(virtualPath)
	newPath := path.Join(oldParent, newName)
	return e.vfs.Rename(virtualPath, newPath)
}

func (e *Engine) Upload(ctx context.Context, src, dst string, opts engine.UploadOptions) (string, error) {
	dstTrim := strings.TrimPrefix(dst, "/")
	parts := strings.SplitN(dstTrim, "/", 2)
	remoteName := strings.TrimSuffix(parts[0], ":")
	remotePath := ""
	if len(parts) > 1 {
		remotePath = parts[1]
	}

	info, err := os.Stat(src)
	if err != nil {
		return "", fmt.Errorf("stat src: %w", err)
	}
	isDir := info.IsDir()

	jobID := fmt.Sprintf("job-%d", opts.JobID)
	if opts.TrackingID != "" {
		jobID = opts.TrackingID
	}

	var size int64
	if isDir {
		filepath.Walk(src, func(_ string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				size += info.Size()
			}
			return nil
		})
	} else {
		size = info.Size()
	}

	jobCtx, cancel := context.WithCancel(e.appCtx)
	j := &job{
		id:      jobID,
		trackID: opts.TrackingID,
		size:    size,
		done:    make(chan struct{}),
		cancel:  cancel,
	}

	e.mu.Lock()
	e.activeJobs[jobID] = j
	e.mu.Unlock()

	go func() {
		defer close(j.done)
		defer e.removeJob(jobID)

		// Ensure stats are tracked for this group
		jobCtx = accounting.WithStatsGroup(jobCtx, jobID)

		var err error
		params := rc.Params{
			"_group": jobID,
		}

		if isDir {
			params["srcFs"] = src
			params["dstFs"] = remoteName + ":" + remotePath
			_, err = e.Call(jobCtx, "sync/copy", params)
		} else {
			params["srcFs"] = filepath.Dir(src)
			params["srcRemote"] = filepath.Base(src)
			params["dstFs"] = remoteName + ":" + remotePath
			params["dstRemote"] = filepath.Base(src)
			_, err = e.Call(jobCtx, "operations/copyfile", params)
		}

		if err != nil {
			if e.onError != nil {
				e.onError(j.trackID, err)
			}
		} else {
			if e.onComplete != nil {
				e.onComplete(j.trackID)
			}
		}
	}()

	return jobID, nil
}

func (e *Engine) removeJob(id string) {
	e.mu.Lock()
	delete(e.activeJobs, id)
	e.mu.Unlock()
}

func (e *Engine) pollAccounting() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		e.mu.RLock()
		if len(e.activeJobs) == 0 {
			e.mu.RUnlock()
			continue
		}

		jobs := make([]*job, 0, len(e.activeJobs))
		for _, j := range e.activeJobs {
			jobs = append(jobs, j)
		}
		e.mu.RUnlock()

		// Get core/stats function
		call := rc.Calls.Get("core/stats")
		if call == nil {
			log.Println("Rclone: core/stats RC command not found")
			continue
		}

		for _, j := range jobs {
			params := rc.Params{"group": j.id}
			res, err := call.Fn(e.appCtx, params)
			if err != nil {
				// Job might have finished or group not found
				continue
			}

			// Helper to get number from params
			getNumber := func(key string) int64 {
				if v, ok := res[key]; ok {
					switch val := v.(type) {
					case int64:
						return val
					case float64:
						return int64(val)
					case int:
						return int64(val)
					}
				}
				return 0
			}

			if e.onProgress != nil {
				e.onProgress(j.trackID, engine.UploadProgress{
					Uploaded: getNumber("bytes"),
					Size:     j.size,
					Speed:    getNumber("speed"),
				})
			}
		}
	}
}

func (e *Engine) Status(ctx context.Context, jobID string) (*engine.UploadStatus, error) {
	e.mu.RLock()
	j, ok := e.activeJobs[jobID]
	e.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("job not found")
	}

	stats := accounting.StatsGroup(e.appCtx, jobID)
	status := "running"
	if stats.Errored() {
		status = "error"
	}

	return &engine.UploadStatus{
		JobID:    jobID,
		Status:   status,
		Uploaded: stats.GetBytes(),
		Size:     j.size,
	}, nil
}

func (e *Engine) GetGlobalStats(ctx context.Context) (*engine.GlobalStats, error) {
	call := rc.Calls.Get("core/stats")
	if call == nil {
		return nil, fmt.Errorf("core/stats command not found")
	}

	res, err := call.Fn(ctx, nil)
	if err != nil {
		return nil, err
	}

	getNumber := func(key string) int64 {
		if v, ok := res[key]; ok {
			switch val := v.(type) {
			case int64:
				return val
			case float64:
				return int64(val)
			case int:
				return int64(val)
			}
		}
		return 0
	}

	return &engine.GlobalStats{
		Speed:           getNumber("speed"),
		ActiveTransfers: int(getNumber("transfers")),
	}, nil
}

func (e *Engine) Version(ctx context.Context) (string, error) {
	return fs.Version, nil
}

func (e *Engine) Call(ctx context.Context, method string, params rc.Params) (rc.Params, error) {
	call := rc.Calls.Get(method)
	if call == nil {
		return nil, fmt.Errorf("method %q not found", method)
	}
	return call.Fn(ctx, params)
}

func (e *Engine) ListRemotes(ctx context.Context) ([]engine.Remote, error) {
	remotes := rclConfig.GetRemoteNames()
	results := make([]engine.Remote, 0, len(remotes))
	for _, r := range remotes {
		if r == GravityRootRemote {
			continue
		}
		results = append(results, engine.Remote{
			Name:      r,
			Connected: true,
		})
	}
	return results, nil
}

func (e *Engine) CreateRemote(ctx context.Context, name, rtype string, config map[string]string) error {
	for k, v := range config {
		rclConfig.Data().SetValue(name, k, v)
	}
	rclConfig.Data().SetValue(name, "type", rtype)
	err := rclConfig.Data().Save()
	if err == nil {
		SyncGravityRoot()
	}
	return err
}

func (e *Engine) DeleteRemote(ctx context.Context, name string) error {
	rclConfig.Data().DeleteSection(name)
	err := rclConfig.Data().Save()
	if err == nil {
		SyncGravityRoot()
	}
	return err
}

func (e *Engine) TestRemote(ctx context.Context, name string) error {
	_, err := fs.NewFs(e.appCtx, name+":")
	return err
}

func (e *Engine) Copy(ctx context.Context, srcPath, dstPath string) (string, error) {
	jobID := "copy-" + strings.ReplaceAll(srcPath, "/", "-") + "-" + fmt.Sprint(time.Now().Unix())
	jobCtx, cancel := context.WithCancel(e.appCtx)
	j := &job{
		id:      jobID,
		trackID: jobID,
		done:    make(chan struct{}),
		cancel:  cancel,
	}

	e.mu.Lock()
	e.activeJobs[jobID] = j
	e.mu.Unlock()

	go func() {
		defer close(j.done)
		defer e.removeJob(jobID)

		// Ensure stats are tracked for this group
		jobCtx = accounting.WithStatsGroup(jobCtx, jobID)

		// Parse paths
		// srcPath: /remote/path/to/file -> remote, path/to/file
		srcClean := strings.TrimPrefix(srcPath, "/")
		srcParts := strings.SplitN(srcClean, "/", 2)
		srcRemote := strings.TrimSuffix(srcParts[0], ":")
		srcRPath := ""
		if len(srcParts) > 1 {
			srcRPath = srcParts[1]
		}

		dstClean := strings.TrimPrefix(dstPath, "/")
		dstParts := strings.SplitN(dstClean, "/", 2)
		dstRemote := strings.TrimSuffix(dstParts[0], ":")
		dstRPath := ""
		if len(dstParts) > 1 {
			dstRPath = dstParts[1]
		}

		params := rc.Params{
			"srcFs":     srcRemote + ":",
			"srcRemote": srcRPath,
			"dstFs":     dstRemote + ":",
			"dstRemote": dstRPath,
			"_group":    jobID,
		}

		_, err := e.Call(jobCtx, "operations/copyfile", params)

		if err != nil {
			if e.onError != nil {
				e.onError(j.trackID, err)
			}
		} else {
			if e.onComplete != nil {
				e.onComplete(j.trackID)
			}
		}
	}()

	return jobID, nil
}

func (e *Engine) Move(ctx context.Context, srcPath, dstPath string) (string, error) {
	jobID := "move-" + strings.ReplaceAll(srcPath, "/", "-") + "-" + fmt.Sprint(time.Now().Unix())
	jobCtx, cancel := context.WithCancel(e.appCtx)
	j := &job{
		id:      jobID,
		trackID: jobID,
		done:    make(chan struct{}),
		cancel:  cancel,
	}

	e.mu.Lock()
	e.activeJobs[jobID] = j
	e.mu.Unlock()

	go func() {
		defer close(j.done)
		defer e.removeJob(jobID)

		// Ensure stats are tracked for this group
		jobCtx = accounting.WithStatsGroup(jobCtx, jobID)

		// Parse paths
		srcClean := strings.TrimPrefix(srcPath, "/")
		srcParts := strings.SplitN(srcClean, "/", 2)
		srcRemote := strings.TrimSuffix(srcParts[0], ":")
		srcRPath := ""
		if len(srcParts) > 1 {
			srcRPath = srcParts[1]
		}

		dstClean := strings.TrimPrefix(dstPath, "/")
		dstParts := strings.SplitN(dstClean, "/", 2)
		dstRemote := strings.TrimSuffix(dstParts[0], ":")
		dstRPath := ""
		if len(dstParts) > 1 {
			dstRPath = dstParts[1]
		}

		params := rc.Params{
			"srcFs":     srcRemote + ":",
			"srcRemote": srcRPath,
			"dstFs":     dstRemote + ":",
			"dstRemote": dstRPath,
			"_group":    jobID,
		}

		_, err := e.Call(jobCtx, "operations/movefile", params)

		if err != nil {
			if e.onError != nil {
				e.onError(j.trackID, err)
			}
		} else {
			if e.onComplete != nil {
				e.onComplete(j.trackID)
			}
		}
	}()

	return jobID, nil
}

func (e *Engine) Cancel(ctx context.Context, jobID string) error {
	e.mu.RLock()
	j, ok := e.activeJobs[jobID]
	e.mu.RUnlock()

	if ok {
		j.cancel()
	}
	return nil
}

func (e *Engine) Configure(ctx context.Context, options map[string]string) error {
	// Helper for Booleans
	parseBool := func(key string, target *bool) {
		if val, ok := options[key]; ok {
			*target = val == "true" || val == "1" || val == "yes"
		}
	}

	// Helper for Durations
	parseDuration := func(key string, target *fs.Duration) {
		if val, ok := options[key]; ok {
			if d, err := fs.ParseDuration(val); err == nil {
				*target = fs.Duration(d)
			}
		}
	}

	// Helper for Sizes
	parseSize := func(key string, target *fs.SizeSuffix) {
		if val, ok := options[key]; ok {
			var size fs.SizeSuffix
			if err := size.Set(val); err == nil {
				*target = size
			}
		}
	}

	// 1. Cache Mode
	if val, ok := options["vfsCacheMode"]; ok {
		var mode vfscommon.CacheMode
		if err := mode.Set(val); err == nil {
			vfscommon.Opt.CacheMode = mode
		}
	}

	// 2. Flags / Booleans
	parseBool("vfsNoChecksum", &vfscommon.Opt.NoChecksum)
	parseBool("vfsRefresh", &vfscommon.Opt.Refresh)
	parseBool("vfsCaseInsensitive", &vfscommon.Opt.CaseInsensitive)

	// 3. Durations
	parseDuration("vfsDirCacheTime", &vfscommon.Opt.DirCacheTime)
	parseDuration("vfsPollInterval", &vfscommon.Opt.PollInterval)
	parseDuration("vfsWriteBack", &vfscommon.Opt.WriteBack)
	parseDuration("vfsCacheMaxAge", &vfscommon.Opt.CacheMaxAge)
	parseDuration("vfsCachePollInterval", &vfscommon.Opt.CachePollInterval)
	parseDuration("vfsWriteWait", &vfscommon.Opt.WriteWait)
	parseDuration("vfsReadWait", &vfscommon.Opt.ReadWait)

	// 4. Sizes
	parseSize("vfsCacheMaxSize", &vfscommon.Opt.CacheMaxSize)
	parseSize("vfsCacheMinFreeSpace", &vfscommon.Opt.CacheMinFreeSpace)
	parseSize("vfsReadChunkSize", &vfscommon.Opt.ChunkSize)
	parseSize("vfsReadChunkSizeLimit", &vfscommon.Opt.ChunkSizeLimit)
	parseSize("vfsReadAhead", &vfscommon.Opt.ReadAhead)
	parseSize("vfsDiskSpaceTotalSize", &vfscommon.Opt.DiskSpaceTotalSize)

	// 5. Integers
	if val, ok := options["vfsReadChunkStreams"]; ok {
		if i, err := strconv.Atoi(val); err == nil {
			vfscommon.Opt.ChunkStreams = i
		}
	}

	log.Printf("Rclone: VFS configured (CacheMode=%v, WriteBack=%v, ReadAhead=%v)",
		vfscommon.Opt.CacheMode, vfscommon.Opt.WriteBack, vfscommon.Opt.ReadAhead)

	return e.Restart(ctx)
}

func (e *Engine) Restart(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Shutdown existing VFS if running
	if e.vfs != nil {
		e.vfs.Shutdown()
		e.vfs = nil
	}

	// Re-initialize Root FS
	f, err := fs.NewFs(ctx, GravityRootRemote+":")
	if err != nil {
		return fmt.Errorf("failed to recreate root fs: %w", err)
	}

	// Create new VFS with updated global options
	e.vfs = vfs.New(f, &vfscommon.Opt)

	log.Println("Rclone: VFS restarted successfully")
	return nil
}

func (e *Engine) OnProgress(h func(string, engine.UploadProgress)) { e.onProgress = h }
func (e *Engine) OnComplete(h func(string))                        { e.onComplete = h }
func (e *Engine) OnError(h func(string, error))                    { e.onError = h }
