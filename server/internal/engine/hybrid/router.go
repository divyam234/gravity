package hybrid

import (
	"context"
	"strings"
	"sync"

	"gravity/internal/engine"
	"gravity/internal/model"

	"go.uber.org/zap"
)

type HybridRouter struct {
	aria2  engine.DownloadEngine
	native engine.DownloadEngine
	logger *zap.Logger

	mu       sync.RWMutex
	settings *model.Settings

	// Task routing map: TaskID -> EngineType
	taskMap map[string]string
}

func NewHybridRouter(aria2, native engine.DownloadEngine, l *zap.Logger) *HybridRouter {
	return &HybridRouter{
		aria2:   aria2,
		native:  native,
		taskMap: make(map[string]string),
		logger:  l.With(zap.String("engine", "hybrid")),
	}
}

func (h *HybridRouter) Start(ctx context.Context) error {
	if err := h.aria2.Start(ctx); err != nil {
		return err
	}
	return h.native.Start(ctx)
}

func (h *HybridRouter) Stop() error {
	h.aria2.Stop()
	return h.native.Stop()
}

func (h *HybridRouter) Add(ctx context.Context, url string, opts engine.DownloadOptions) (string, error) {
	h.mu.RLock()
	pref := "aria2"
	if h.settings != nil {
		if strings.HasPrefix(url, "magnet:") || opts.TorrentData != "" {
			pref = h.settings.Download.PreferredMagnetEngine
		} else {
			pref = h.settings.Download.PreferredEngine
		}
	}
	h.mu.RUnlock()

	var gid string
	var err error

	if pref == "native" {
		h.logger.Debug("routing task to native engine", zap.String("url", url))
		gid, err = h.native.Add(ctx, url, opts)
	} else {
		h.logger.Debug("routing task to aria2 engine", zap.String("url", url))
		gid, err = h.aria2.Add(ctx, url, opts)
	}

	if err == nil {
		h.mu.Lock()
		h.taskMap[gid] = pref
		h.mu.Unlock()
	}

	return gid, nil
}

func (h *HybridRouter) getEngine(id string) engine.DownloadEngine {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.taskMap[id] == "native" {
		return h.native
	}
	return h.aria2
}

func (h *HybridRouter) Status(ctx context.Context, id string) (*engine.DownloadStatus, error) {
	return h.getEngine(id).Status(ctx, id)
}

func (h *HybridRouter) Pause(ctx context.Context, id string) error {
	return h.getEngine(id).Pause(ctx, id)
}

func (h *HybridRouter) Resume(ctx context.Context, id string) error {
	return h.getEngine(id).Resume(ctx, id)
}

func (h *HybridRouter) Cancel(ctx context.Context, id string) error {
	return h.getEngine(id).Cancel(ctx, id)
}

func (h *HybridRouter) Remove(ctx context.Context, id string) error {
	e := h.getEngine(id)
	err := e.Remove(ctx, id)

	h.mu.Lock()
	delete(h.taskMap, id)
	h.mu.Unlock()

	return err
}

func (h *HybridRouter) List(ctx context.Context) ([]*engine.DownloadStatus, error) {
	// Combine results from both
	l1, _ := h.aria2.List(ctx)
	l2, _ := h.native.List(ctx)
	return append(l1, l2...), nil
}

func (h *HybridRouter) Sync(ctx context.Context) error {
	h.aria2.Sync(ctx)
	return h.native.Sync(ctx)
}

func (h *HybridRouter) Configure(ctx context.Context, s *model.Settings) error {
	h.mu.Lock()
	h.settings = s
	h.mu.Unlock()

	h.aria2.Configure(ctx, s)
	return h.native.Configure(ctx, s)
}

func (h *HybridRouter) Version(ctx context.Context) (string, error) {
	return h.aria2.Version(ctx)
}

func (h *HybridRouter) GetVersions(ctx context.Context) (aria2 string, native string) {
	aria2, _ = h.aria2.Version(ctx)
	native, _ = h.native.Version(ctx)
	return
}

func (h *HybridRouter) OnProgress(f func(string, engine.Progress)) {
	h.aria2.OnProgress(f)
	h.native.OnProgress(f)
}

func (h *HybridRouter) OnComplete(f func(string, string)) {
	h.aria2.OnComplete(f)
	h.native.OnComplete(f)
}

func (h *HybridRouter) OnError(f func(string, error)) {
	h.aria2.OnError(f)
	h.native.OnError(f)
}

func (h *HybridRouter) GetMagnetFiles(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
	h.mu.RLock()
	pref := "aria2"
	if h.settings != nil {
		pref = h.settings.Download.PreferredMagnetEngine
	}
	h.mu.RUnlock()

	if pref == "native" {
		return h.native.GetMagnetFiles(ctx, magnet)
	}
	return h.aria2.GetMagnetFiles(ctx, magnet)
}

func (h *HybridRouter) GetTorrentFiles(ctx context.Context, torrentBase64 string) (*model.MagnetInfo, error) {
	h.mu.RLock()
	pref := "aria2"
	if h.settings != nil {
		pref = h.settings.Download.PreferredMagnetEngine
	}
	h.mu.RUnlock()

	if pref == "native" {
		return h.native.GetTorrentFiles(ctx, torrentBase64)
	}
	return h.aria2.GetTorrentFiles(ctx, torrentBase64)
}

func (h *HybridRouter) AddMagnetWithSelection(ctx context.Context, magnet string, selectedIndexes []string, opts engine.DownloadOptions) (string, error) {
	h.mu.RLock()
	pref := "aria2"
	if h.settings != nil {
		pref = h.settings.Download.PreferredMagnetEngine
	}
	h.mu.RUnlock()

	var gid string
	var err error
	if pref == "native" {
		h.logger.Debug("routing magnet to native engine", zap.String("magnet", magnet))
		gid, err = h.native.AddMagnetWithSelection(ctx, magnet, selectedIndexes, opts)
	} else {
		h.logger.Debug("routing magnet to aria2 engine", zap.String("magnet", magnet))
		gid, err = h.aria2.AddMagnetWithSelection(ctx, magnet, selectedIndexes, opts)
	}

	if err == nil {
		h.mu.Lock()
		h.taskMap[gid] = pref
		h.mu.Unlock()
	}
	return gid, nil
}

func (h *HybridRouter) GetPeers(ctx context.Context, id string) ([]engine.DownloadPeer, error) {

	return h.getEngine(id).GetPeers(ctx, id)

}

func (h *HybridRouter) ResumePolling() {
	if aria, ok := h.aria2.(interface{ ResumePolling() }); ok {
		aria.ResumePolling()
	}
	if n, ok := h.native.(interface{ ResumePolling() }); ok {
		n.ResumePolling()
	}

}

func (h *HybridRouter) PausePolling() {
	if aria, ok := h.aria2.(interface{ PausePolling() }); ok {
		aria.PausePolling()
	}
	if n, ok := h.native.(interface{ PausePolling() }); ok {
		n.PausePolling()
	}

}
