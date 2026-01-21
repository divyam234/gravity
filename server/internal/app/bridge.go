package app

import (
	"context"
	"fmt"
	"gravity/internal/model"
	"gravity/internal/service"
)

type Bridge struct {
	app *App
}

func NewBridge(a *App) *Bridge {
	return &Bridge{app: a}
}

// Downloads
func (b *Bridge) GetDownloads(status []string, limit, offset int) (any, error) {
	downloads, total, err := b.app.downloadService.List(context.Background(), status, limit, offset)
	return map[string]any{
		"data": downloads,
		"meta": map[string]any{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	}, err
}

func (b *Bridge) GetDownload(id string) (*model.Download, error) {
	return b.app.downloadService.Get(context.Background(), id)
}

func (b *Bridge) CreateDownload(url, destination, filename string) (*model.Download, error) {
	return b.app.downloadService.Create(context.Background(), url, filename, destination)
}

func (b *Bridge) PauseDownload(id string) error {
	return b.app.downloadService.Pause(context.Background(), id)
}

func (b *Bridge) ResumeDownload(id string) error {
	return b.app.downloadService.Resume(context.Background(), id)
}

func (b *Bridge) DeleteDownload(id string, deleteFiles bool) error {
	return b.app.downloadService.Delete(context.Background(), id, deleteFiles)
}

// Providers
func (b *Bridge) GetProviders() (any, error) {
	providers, err := b.app.providerService.List(context.Background())
	return map[string]any{"data": providers}, err
}

func (b *Bridge) ConfigureProvider(name string, config map[string]string, enabled bool) error {
	return b.app.providerService.Configure(context.Background(), name, config, enabled)
}

func (b *Bridge) ResolveUrl(url string) (any, error) {
	res, name, err := b.app.providerService.Resolve(context.Background(), url)
	return map[string]any{"resource": res, "provider": name}, err
}

// Remotes
func (b *Bridge) GetRemotes() (any, error) {
	remotes, err := b.app.uploadEngine.ListRemotes(context.Background())
	return map[string]any{"data": remotes}, err
}

// Stats
func (b *Bridge) GetStats() (*model.Stats, error) {
	return b.app.statsService.GetCurrent(context.Background())
}

// Settings
func (b *Bridge) GetSettings() (map[string]string, error) {
	repo := b.app.downloadService.SettingsRepo()
	return repo.Get(context.Background())
}

func (b *Bridge) UpdateSettings(settings map[string]string) error {
	ctx := context.Background()
	repo := b.app.downloadService.SettingsRepo()
	for k, v := range settings {
		if err := repo.Set(ctx, k, v); err != nil {
			return err
		}
	}
	// Apply to engines
	b.app.downloadEngine.Configure(ctx, settings)
	b.app.uploadEngine.Configure(ctx, settings)
	return nil
}

// Files
func (b *Bridge) ListFiles(path string) (any, error) {
	files, err := b.app.uploadEngine.List(context.Background(), path)
	return map[string]any{"data": files}, err
}

func (b *Bridge) Mkdir(path string) error {
	return b.app.uploadEngine.Mkdir(context.Background(), path)
}

func (b *Bridge) DeleteFile(path string) error {
	return b.app.uploadEngine.Delete(context.Background(), path)
}

func (b *Bridge) OperateFile(op, src, dst string) (any, error) {
	// Rclone engine has Move/Copy/Rename methods but maybe not a unified 'Operate'
	// Wait, I saw Operate in FileHandler. Let's check Rclone engine methods again.
	var jobID string
	var err error
	ctx := context.Background()
	switch op {
	case "move":
		jobID, err = b.app.uploadEngine.Move(ctx, src, dst)
	case "copy":
		jobID, err = b.app.uploadEngine.Copy(ctx, src, dst)
	case "rename":
		err = b.app.uploadEngine.Rename(ctx, src, dst)
	default:
		return nil, fmt.Errorf("unknown operation: %s", op)
	}
	return map[string]any{"jobId": jobID}, err
}

// Magnets
func (b *Bridge) CheckMagnet(magnet string) (*model.MagnetInfo, error) {
	return b.app.magnetService.CheckMagnet(context.Background(), magnet)
}

func (b *Bridge) CheckTorrent(torrentBase64 string) (*model.MagnetInfo, error) {
	return b.app.magnetService.CheckTorrent(context.Background(), torrentBase64)
}

func (b *Bridge) DownloadMagnet(req service.MagnetDownloadRequest) (*model.Download, error) {
	return b.app.magnetService.DownloadMagnet(context.Background(), req)
}

// Search
func (b *Bridge) Search(q string, limit, offset int) (any, error) {
	results, total, err := b.app.searchService.Search(context.Background(), q, limit, offset)
	return map[string]any{
		"data": results,
		"meta": map[string]any{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	}, err
}
