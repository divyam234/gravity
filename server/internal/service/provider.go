package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gravity/internal/engine"
	"gravity/internal/model"
	"gravity/internal/provider"
	"gravity/internal/store"

	"go.uber.org/zap"
)

type ProviderService struct {
	repo     *store.ProviderRepo
	registry *provider.Registry
	resolver *provider.Resolver
	engine   engine.DownloadEngine
	logger   *zap.Logger
}

func NewProviderService(repo *store.ProviderRepo, registry *provider.Registry, engine engine.DownloadEngine, l *zap.Logger) *ProviderService {
	return &ProviderService{
		repo:     repo,
		registry: registry,
		resolver: provider.NewResolver(registry),
		engine:   engine,
		logger:   l.With(zap.String("service", "provider")),
	}
}

func (s *ProviderService) Init(ctx context.Context) error {
	// Load configurations from DB and apply to providers in registry
	stored, err := s.repo.List(ctx)
	if err != nil {
		return err
	}

	for _, p := range stored {
		impl := s.registry.Get(p.Name)
		if impl != nil {
			impl.Configure(ctx, p.Config)
		}
	}

	return nil
}

func (s *ProviderService) List(ctx context.Context) ([]model.ProviderSummary, error) {
	// Combine implementation info with stored config/status
	results := []model.ProviderSummary{}

	for _, impl := range s.registry.List() {
		summary := model.ProviderSummary{
			Name:        impl.Name(),
			DisplayName: impl.DisplayName(),
			Type:        impl.Type(),
			Priority:    impl.Priority(),
			Configured:  impl.IsConfigured(),
		}

		stored, err := s.repo.Get(ctx, impl.Name())
		if err == nil {
			summary.Enabled = stored.Enabled
			summary.Account = stored.CachedAccount
		} else {
			summary.Enabled = false
		}

		results = append(results, summary)
	}

	return results, nil
}

func (s *ProviderService) Configure(ctx context.Context, name string, config map[string]string, enabled bool) error {
	s.logger.Info("configuring provider", zap.String("name", name), zap.Bool("enabled", enabled))
	impl := s.registry.Get(name)
	if impl == nil {
		return fmt.Errorf("provider not found")
	}

	if err := impl.Configure(ctx, config); err != nil {
		s.logger.Error("failed to configure provider", zap.String("name", name), zap.Error(err))
		return err
	}

	// Test connection and cache account info
	account, _ := impl.Test(ctx)

	p := &model.Provider{
		Name:          name,
		Enabled:       enabled,
		Priority:      impl.Priority(),
		Config:        config,
		CachedAccount: account,
	}

	return s.repo.Save(ctx, p)
}

func (s *ProviderService) Resolve(ctx context.Context, url string, headers map[string]string, torrentBase64 string) (*provider.ResolveResult, string, error) {
	// 1. Handle Torrent/Magnet
	if torrentBase64 != "" || strings.HasPrefix(url, "magnet:") {
		info, err := s.checkMetadata(ctx, url, torrentBase64)
		if err != nil {
			return nil, "", err
		}

		files := flattenMagnetFiles(info.Files)
		mode := model.ExecutionModeMagnet
		if info.Cached && hasDebridLinks(files) {
			mode = model.ExecutionModeDebridFiles
		}

		return &provider.ResolveResult{
			Name:          info.Name,
			Size:          info.Size,
			IsMagnet:      true,
			Hash:          info.Hash,
			Files:         files,
			ExecutionMode: mode,
		}, info.Source, nil
	}

	// 2. Handle Regular URL
	res, providerName, err := s.resolver.Resolve(ctx, url, headers)
	if err == nil && res != nil {
		res.ExecutionMode = model.ExecutionModeDirect
	}
	return res, providerName, err
}

// checkMetadata resolves magnet/torrent metadata from providers or engine
func (s *ProviderService) checkMetadata(ctx context.Context, magnet string, torrentBase64 string) (*model.MagnetInfo, error) {
	var info *model.MagnetInfo
	var err error

	// 1. If torrent, parse it first to get hash/info
	if torrentBase64 != "" {
		s.logger.Debug("parsing torrent file")
		info, err = s.engine.GetTorrentFiles(ctx, torrentBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse torrent: %w", err)
		}
		if info.Hash != "" {
			magnet = fmt.Sprintf("magnet:?xt=urn:btih:%s", info.Hash)
		}
	}

	// 2. Check providers (if we have a magnet link now)
	if magnet != "" {
		s.logger.Debug("checking magnet availability", zap.String("magnet", magnet))

		// Create a separate context for the provider checks to ensure timeout
		checkCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		for _, p := range s.registry.List() {
			if mp, ok := p.(provider.MagnetProvider); ok && p.IsConfigured() {
				s.logger.Debug("trying magnet provider check", zap.String("provider", p.Name()))
				cached, err := mp.CheckMagnet(checkCtx, magnet)
				if err == nil && cached != nil && cached.Cached {
					s.logger.Debug("found cached magnet on provider", zap.String("provider", p.Name()), zap.String("name", cached.Name))
					cached.Source = p.Name()
					return cached, nil
				}
			}
		}
	}

	// 3. If we already have info from torrent, return it (it wasn't cached)
	if info != nil {
		// info.Source is set by engine, don't overwrite if it's generic
		return info, nil
	}

	// 4. Fallback: Resolve magnet via engine
	if magnet != "" {
		s.logger.Debug("falling back to engine metadata fetch")
		info, err := s.engine.GetMagnetFiles(ctx, magnet)
		if err != nil {
			s.logger.Error("engine metadata fetch failed", zap.Error(err))
			return nil, fmt.Errorf("failed to get magnet files: %w", err)
		}
		// info.Source is set by engine to "aria2" or "native"
		s.logger.Debug("successfully fetched metadata via engine", zap.String("name", info.Name))
		return info, nil
	}

	return nil, fmt.Errorf("no magnet or torrent provided")
}

func (s *ProviderService) GetConfigSchema(name string) ([]provider.ConfigField, error) {
	impl := s.registry.Get(name)
	if impl == nil {
		return nil, fmt.Errorf("provider not found")
	}
	return impl.ConfigSchema(), nil
}

func (s *ProviderService) Delete(ctx context.Context, name string) error {
	// Reset config and disable
	return s.Configure(ctx, name, map[string]string{}, false)
}

func (s *ProviderService) GetStatus(ctx context.Context, name string) (*model.AccountInfo, error) {
	impl := s.registry.Get(name)
	if impl == nil {
		return nil, fmt.Errorf("provider not found")
	}
	// Force a test to get fresh status
	return impl.Test(ctx)
}

func (s *ProviderService) GetHosts(ctx context.Context, name string) ([]string, error) {
	impl := s.registry.Get(name)
	if impl == nil {
		return nil, fmt.Errorf("provider not found")
	}

	if debrid, ok := impl.(provider.DebridProvider); ok {
		return debrid.GetHosts(ctx)
	}

	return []string{}, nil
}

// flattenMagnetFiles converts nested MagnetFile tree to flat array of DownloadFile
func flattenMagnetFiles(files []*model.MagnetFile) []*model.DownloadFile {
	var result []*model.DownloadFile

	var traverse func(items []*model.MagnetFile)
	traverse = func(items []*model.MagnetFile) {
		for _, file := range items {
			if !file.IsFolder {
				result = append(result, &model.DownloadFile{
					ID:    file.ID,
					Name:  file.Name,
					Path:  file.Path,
					Size:  file.Size,
					URL:   file.Link,
					Index: file.Index,
				})
			}
			if len(file.Children) > 0 {
				traverse(file.Children)
			}
		}
	}

	traverse(files)
	return result
}

func hasDebridLinks(files []*model.DownloadFile) bool {
	for _, f := range files {
		if f.URL != "" {
			return true
		}
	}
	return false
}
