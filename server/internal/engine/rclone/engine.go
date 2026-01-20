package rclone

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gravity/internal/engine"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeletePrefix(ctx context.Context, prefix string) error
}

type Engine struct {
	runner *Runner
	client *Client

	onProgress func(jobID string, progress engine.UploadProgress)
	onComplete func(jobID string)
	onError    func(jobID string, err error)

	activeJobs map[string]struct {
		trackID string
		size    int64
		srcPath string
		dstPath string
	}
	cache Cache
	mu    sync.RWMutex
}

func NewEngine(port int, cache Cache) *Engine {
	runner := NewRunner(port)
	client := NewClient(fmt.Sprintf("http://localhost:%d", port))

	return &Engine{
		runner: runner,
		client: client,
		activeJobs: make(map[string]struct {
			trackID string
			size    int64
			srcPath string
			dstPath string
		}),
		cache: cache,
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
		srcPath string
		dstPath string
	}{
		trackID: opts.TrackingID,
		size:    info.Size(),
		srcPath: src,
		dstPath: dst,
	}
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
			srcPath string
			dstPath string
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
				// Invalidate cache for destination
				if data.dstPath != "" {
					e.invalidateListCache(context.Background(), data.dstPath)
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

func (e *Engine) parseVirtualPath(path string) (string, string) {
	path = strings.Trim(path, "/")
	if path == "" {
		return "", ""
	}
	parts := strings.SplitN(path, "/", 2)
	remote := parts[0]
	remotePath := ""
	if len(parts) > 1 {
		remotePath = parts[1]
	}
	return remote, remotePath
}

func (e *Engine) invalidateListCache(ctx context.Context, virtualPath string) {
	// Invalidate the directory listing where this item resides
	parent := filepath.Dir(virtualPath)
	if parent == "." {
		parent = "/"
	}
	e.cache.Delete(ctx, "list:"+parent)

	// Also invalidate the item itself if it's a directory
	e.cache.Delete(ctx, "list:"+virtualPath)
}

func (e *Engine) List(ctx context.Context, virtualPath string) ([]engine.FileInfo, error) {
	// Check cache
	cacheKey := "list:" + virtualPath
	if cached, err := e.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		var files []engine.FileInfo
		if err := gob.NewDecoder(bytes.NewReader(cached)).Decode(&files); err == nil {
			return files, nil
		}
	}

	remote, remotePath := e.parseVirtualPath(virtualPath)

	var files []engine.FileInfo
	var err error

	// Root: List Remotes
	if remote == "" {
		files, err = e.listRemotesAsFiles(ctx)
	} else {
		// Remote: List Files
		files, err = e.listFiles(ctx, remote, remotePath)
	}

	if err != nil {
		return nil, err
	}

	// Save to cache
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(files); err == nil {
		e.cache.Set(ctx, cacheKey, buf.Bytes(), 1*time.Minute)
	}

	return files, nil
}

func (e *Engine) listRemotesAsFiles(ctx context.Context) ([]engine.FileInfo, error) {
	remotes, err := e.ListRemotes(ctx)
	if err != nil {
		return nil, err
	}
	files := make([]engine.FileInfo, len(remotes))
	for i, r := range remotes {
		files[i] = engine.FileInfo{
			Path:     "/" + r.Name,
			Name:     r.Name,
			Type:     engine.FileTypeFolder,
			IsDir:    true,
			MimeType: "inode/directory",
		}
	}
	return files, nil
}

func (e *Engine) listFiles(ctx context.Context, remote, remotePath string) ([]engine.FileInfo, error) {
	params := map[string]interface{}{
		"fs":     remote + ":",
		"remote": remotePath,
		"opt": map[string]interface{}{
			"showHash":    false,
			"showModTime": true,
		},
	}

	res, err := e.client.Call(ctx, "operations/list", params)
	if err != nil {
		return nil, err
	}

	var list struct {
		List []struct {
			Path     string    `json:"Path"`
			Name     string    `json:"Name"`
			Size     int64     `json:"Size"`
			MimeType string    `json:"MimeType"`
			ModTime  time.Time `json:"ModTime"`
			IsDir    bool      `json:"IsDir"`
		} `json:"list"`
	}

	if err := json.Unmarshal(res, &list); err != nil {
		return nil, err
	}

	files := make([]engine.FileInfo, len(list.List))
	for i, item := range list.List {
		fType := engine.FileTypeFile
		if item.IsDir {
			fType = engine.FileTypeFolder
		}

		files[i] = engine.FileInfo{
			Path:     "/" + remote + "/" + item.Path,
			Name:     item.Name,
			Size:     item.Size,
			MimeType: item.MimeType,
			ModTime:  item.ModTime,
			Type:     fType,
			IsDir:    item.IsDir,
		}
	}

	return files, nil
}

func (e *Engine) Mkdir(ctx context.Context, virtualPath string) error {
	remote, remotePath := e.parseVirtualPath(virtualPath)
	if remote == "" {
		return fmt.Errorf("cannot create folder in root")
	}

	params := map[string]interface{}{
		"fs":     remote + ":",
		"remote": remotePath,
	}
	_, err := e.client.Call(ctx, "operations/mkdir", params)
	if err == nil {
		e.invalidateListCache(ctx, virtualPath)
	}
	return err
}

func (e *Engine) Delete(ctx context.Context, virtualPath string) error {
	remote, remotePath := e.parseVirtualPath(virtualPath)
	if remote == "" {
		return fmt.Errorf("cannot delete root items")
	}

	params := map[string]interface{}{
		"fs":     remote + ":",
		"remote": remotePath,
	}
	_, err := e.client.Call(ctx, "operations/deletefile", params)
	if err == nil {
		e.invalidateListCache(ctx, virtualPath)
	}
	return err
}

func (e *Engine) Rename(ctx context.Context, virtualPath, newName string) error {
	remote, remotePath := e.parseVirtualPath(virtualPath)
	if remote == "" {
		return fmt.Errorf("cannot rename root items")
	}

	params := map[string]interface{}{
		"srcFs":     remote + ":",
		"srcRemote": remotePath,
		"dstFs":     remote + ":",
		"dstRemote": filepath.Join(filepath.Dir(remotePath), newName),
	}
	_, err := e.client.Call(ctx, "operations/movefile", params)
	if err == nil {
		e.invalidateListCache(ctx, virtualPath)
		// Invalidate destination as well
		parent := filepath.Dir(virtualPath)
		if parent == "." {
			parent = "/"
		}
		e.invalidateListCache(ctx, filepath.Join(parent, newName))
	}
	return err
}

func (e *Engine) Copy(ctx context.Context, srcPath, dstPath string) (string, error) {
	info, err := e.Stat(ctx, srcPath)
	if err != nil {
		return "", err
	}

	srcRemote, srcRemotePath := e.parseVirtualPath(srcPath)
	dstRemote, dstRemotePath := e.parseVirtualPath(dstPath)

	params := map[string]interface{}{
		"_async": true,
	}
	method := ""

	if info.IsDir {
		method = "sync/copy"
		params["srcFs"] = srcRemote + ":" + srcRemotePath
		params["dstFs"] = dstRemote + ":" + dstRemotePath
	} else {
		method = "operations/copyfile"
		params["srcFs"] = srcRemote + ":"
		params["srcRemote"] = srcRemotePath
		params["dstFs"] = dstRemote + ":"
		params["dstRemote"] = dstRemotePath
	}

	res, err := e.client.Call(ctx, method, params)
	if err != nil {
		return "", err
	}

	var jobRes struct {
		JobID int64 `json:"jobid"`
	}
	if err := json.Unmarshal(res, &jobRes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", jobRes.JobID), nil
}

func (e *Engine) Move(ctx context.Context, srcPath, dstPath string) (string, error) {
	info, err := e.Stat(ctx, srcPath)
	if err != nil {
		return "", err
	}

	srcRemote, srcRemotePath := e.parseVirtualPath(srcPath)
	dstRemote, dstRemotePath := e.parseVirtualPath(dstPath)

	params := map[string]interface{}{
		"_async": true,
	}
	method := ""

	if info.IsDir {
		method = "sync/move"
		params["srcFs"] = srcRemote + ":" + srcRemotePath
		params["dstFs"] = dstRemote + ":" + dstRemotePath
	} else {
		method = "operations/movefile"
		params["srcFs"] = srcRemote + ":"
		params["srcRemote"] = srcRemotePath
		params["dstFs"] = dstRemote + ":"
		params["dstRemote"] = dstRemotePath
	}

	res, err := e.client.Call(ctx, method, params)
	if err != nil {
		return "", err
	}

	var jobRes struct {
		JobID int64 `json:"jobid"`
	}
	if err := json.Unmarshal(res, &jobRes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", jobRes.JobID), nil
}

func (e *Engine) Stat(ctx context.Context, virtualPath string) (*engine.FileInfo, error) {
	remote, remotePath := e.parseVirtualPath(virtualPath)
	if remote == "" {
		return &engine.FileInfo{
			Path:     "/",
			Name:     "Root",
			Type:     engine.FileTypeFolder,
			IsDir:    true,
			MimeType: "inode/directory",
		}, nil
	}

	// List parent to find the item
	parent := filepath.Dir(remotePath)
	name := filepath.Base(remotePath)
	if remotePath == "" || remotePath == "." {
		// It's the remote root
		return &engine.FileInfo{
			Path:     "/" + remote,
			Name:     remote,
			Type:     engine.FileTypeFolder,
			IsDir:    true,
			MimeType: "inode/directory",
		}, nil
	}

	// If parent is "." or "/", rclone expects "" for root
	if parent == "." || parent == "/" {
		parent = ""
	}

	params := map[string]interface{}{
		"fs":     remote + ":",
		"remote": parent,
		"opt": map[string]interface{}{
			"showHash":    false,
			"showModTime": true,
		},
	}

	res, err := e.client.Call(ctx, "operations/list", params)
	if err != nil {
		return nil, err
	}

	var list struct {
		List []struct {
			Path     string    `json:"Path"`
			Name     string    `json:"Name"`
			Size     int64     `json:"Size"`
			MimeType string    `json:"MimeType"`
			ModTime  time.Time `json:"ModTime"`
			IsDir    bool      `json:"IsDir"`
		} `json:"list"`
	}

	if err := json.Unmarshal(res, &list); err != nil {
		return nil, err
	}

	for _, item := range list.List {
		if item.Name == name {
			fType := engine.FileTypeFile
			if item.IsDir {
				fType = engine.FileTypeFolder
			}
			return &engine.FileInfo{
				Path:     "/" + remote + "/" + item.Path,
				Name:     item.Name,
				Size:     item.Size,
				MimeType: item.MimeType,
				ModTime:  item.ModTime,
				Type:     fType,
				IsDir:    item.IsDir,
			}, nil
		}
	}

	return nil, fmt.Errorf("file not found")
}
