package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"gravity/internal/engine"
	"gravity/internal/logger"
	"gravity/internal/model"
	"gravity/internal/store"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SearchService struct {
	repo          *store.SearchRepo
	settingsRepo  *store.SettingsRepo
	storageEngine engine.StorageEngine
	mu            sync.Mutex
	isIndexing    map[string]bool
	ctx           context.Context
	logger        *zap.Logger
}

func NewSearchService(repo *store.SearchRepo, settingsRepo *store.SettingsRepo, storage engine.StorageEngine) *SearchService {
	return &SearchService{
		repo:          repo,
		settingsRepo:  settingsRepo,
		storageEngine: storage,
		isIndexing:    make(map[string]bool),
		logger:        logger.Component("SEARCH"),
	}
}

func (s *SearchService) Start(ctx context.Context) {
	if s == nil {
		return
	}
	s.ctx = ctx
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.checkAutoIndexing()
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

func (s *SearchService) checkAutoIndexing() {
	if s == nil || s.settingsRepo == nil {
		return
	}
	ctx := s.ctx
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil || settings == nil {
		return
	}

	for _, c := range settings.Search.Configs {
		if c.AutoIndexIntervalMin <= 0 {
			continue
		}

		shouldIndex := false
		if c.LastIndexedAt == nil {
			shouldIndex = true
		} else {
			if time.Since(*c.LastIndexedAt) > time.Duration(c.AutoIndexIntervalMin)*time.Minute {
				shouldIndex = true
			}
		}

		if shouldIndex && c.Status != "indexing" {
			s.mu.Lock()
			indexing := s.isIndexing[c.Remote]
			s.mu.Unlock()

			if !indexing {
				go s.IndexRemote(ctx, c.Remote)
			}
		}
	}
}

func (s *SearchService) IndexRemote(ctx context.Context, remote string) error {
	s.mu.Lock()
	if s.isIndexing[remote] {
		s.mu.Unlock()
		return fmt.Errorf("indexing already in progress for %s", remote)
	}
	s.isIndexing[remote] = true
	s.mu.Unlock()

	s.logger.Info("starting remote indexing", zap.String("remote", remote))

	defer func() {
		s.mu.Lock()
		delete(s.isIndexing, remote)
		s.mu.Unlock()
	}()

	s.updateConfigStatus(ctx, remote, "indexing", "")

	// Fetch current config for filtering
	configs, err := s.GetConfigs(ctx)
	var config model.RemoteIndexConfig
	if err == nil {
		for _, c := range configs {
			if c.Remote == remote {
				config = c
				break
			}
		}
	}

	files, err := s.listRecursive(ctx, remote, "", config)
	if err != nil {
		s.logger.Error("failed to list files for indexing", zap.String("remote", remote), zap.Error(err))
		s.updateConfigStatus(ctx, remote, "error", err.Error())
		return err
	}

	s.logger.Debug("files listed for indexing", zap.String("remote", remote), zap.Int("count", len(files)))

	// Convert to model format
	indexedFiles := make([]model.IndexedFile, len(files))
	for i, f := range files {
		indexedFiles[i] = model.IndexedFile{
			ID:      uuid.New().String(),
			Remote:  remote,
			Path:    f.Path,
			Name:    f.Name,
			Size:    f.Size,
			ModTime: f.ModTime,
			IsDir:   f.IsDir,
		}
	}

	if err := s.repo.SaveFiles(ctx, remote, indexedFiles); err != nil {
		s.logger.Error("failed to save indexed files to DB", zap.String("remote", remote), zap.Error(err))
		s.updateConfigStatus(ctx, remote, "error", err.Error())
		return err
	}

	s.logger.Info("remote indexing completed", zap.String("remote", remote), zap.Int("count", len(indexedFiles)))
	return s.updateLastIndexed(ctx, remote)
}

func (s *SearchService) listRecursive(ctx context.Context, remote, path string, config model.RemoteIndexConfig) ([]engine.FileInfo, error) {
	// Root of remote
	virtualPath := "/" + remote + "/" + path
	items, err := s.storageEngine.List(ctx, virtualPath)
	if err != nil {
		return nil, err
	}

	var excludeReg *regexp.Regexp
	if config.ExcludedPatterns != "" {
		excludeReg, _ = regexp.Compile(config.ExcludedPatterns)
	}

	var results []engine.FileInfo
	for _, item := range items {
		if item.IsDir {
			results = append(results, item)
			// Subdirectory listing
			subPath := strings.TrimPrefix(item.Path, "/"+remote+"/")
			subItems, err := s.listRecursive(ctx, remote, subPath, config)
			if err == nil {
				results = append(results, subItems...)
			}
			continue
		}

		// Apply Filters for Files
		if config.MinSizeBytes > 0 && item.Size < config.MinSizeBytes {
			continue
		}

		if config.IncludedExtensions != "" {
			exts := strings.Split(strings.ToLower(config.IncludedExtensions), ",")
			match := false
			for _, ext := range exts {
				ext = "." + strings.TrimPrefix(strings.TrimSpace(ext), ".")
				if strings.HasSuffix(strings.ToLower(item.Name), ext) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		if excludeReg != nil && excludeReg.MatchString(item.Path) {
			continue
		}

		results = append(results, item)
	}
	return results, nil
}

func (s *SearchService) Search(ctx context.Context, query string, limit, offset int) ([]model.IndexedFile, int, error) {
	return s.repo.Search(ctx, query, limit, offset)
}

func (s *SearchService) GetConfigs(ctx context.Context) ([]model.RemoteIndexConfig, error) {
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	dbConfigs := settings.Search.Configs

	remotes, err := s.storageEngine.ListRemotes(ctx)
	if err != nil {
		return dbConfigs, nil
	}

	configMap := make(map[string]model.RemoteIndexConfig)
	for _, c := range dbConfigs {
		configMap[c.Remote] = c
	}

	var results []model.RemoteIndexConfig
	for _, r := range remotes {
		if cfg, ok := configMap[r.Name]; ok {
			results = append(results, cfg)
		} else {
			results = append(results, model.RemoteIndexConfig{
				Remote:               r.Name,
				AutoIndexIntervalMin: 0,
				Status:               "idle",
			})
		}
	}

	return results, nil
}

func (s *SearchService) UpdateConfig(ctx context.Context, remote string, interval int, excludedPatterns, includedExtensions string, minSize int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return err
	}

	updated := false
	for i, c := range settings.Search.Configs {
		if c.Remote == remote {
			settings.Search.Configs[i].AutoIndexIntervalMin = interval
			settings.Search.Configs[i].ExcludedPatterns = excludedPatterns
			settings.Search.Configs[i].IncludedExtensions = includedExtensions
			settings.Search.Configs[i].MinSizeBytes = minSize
			updated = true
			break
		}
	}

	if !updated {
		settings.Search.Configs = append(settings.Search.Configs, model.RemoteIndexConfig{
			Remote:               remote,
			AutoIndexIntervalMin: interval,
			Status:               "idle",
			ExcludedPatterns:     excludedPatterns,
			IncludedExtensions:   includedExtensions,
			MinSizeBytes:         minSize,
		})
	}

	return s.settingsRepo.Save(ctx, settings)
}

func (s *SearchService) BatchUpdateConfig(ctx context.Context, configs map[string]model.RemoteIndexConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return err
	}

	configMap := make(map[string]int)
	for i, c := range settings.Search.Configs {
		configMap[c.Remote] = i
	}

	for remote, cfg := range configs {
		if idx, ok := configMap[remote]; ok {
			settings.Search.Configs[idx].AutoIndexIntervalMin = cfg.AutoIndexIntervalMin
			settings.Search.Configs[idx].ExcludedPatterns = cfg.ExcludedPatterns
			settings.Search.Configs[idx].IncludedExtensions = cfg.IncludedExtensions
			settings.Search.Configs[idx].MinSizeBytes = cfg.MinSizeBytes
		} else {
			cfg.Remote = remote
			cfg.Status = "idle"
			settings.Search.Configs = append(settings.Search.Configs, cfg)
		}
	}

	return s.settingsRepo.Save(ctx, settings)
}

func (s *SearchService) updateConfigStatus(ctx context.Context, remote, status, errorMsg string) error {
	// Note: We use a separate mutex or short lock for settings update to avoid contention
	// But SearchService.mu is already held during IndexRemote for specific remote.
	// Save() locks the DB transaction.

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return err
	}

	updated := false
	for i, c := range settings.Search.Configs {
		if c.Remote == remote {
			settings.Search.Configs[i].Status = status
			settings.Search.Configs[i].ErrorMsg = errorMsg
			updated = true
			break
		}
	}

	if !updated {
		// Should not happen for existing config, but handle it
		settings.Search.Configs = append(settings.Search.Configs, model.RemoteIndexConfig{
			Remote:   remote,
			Status:   status,
			ErrorMsg: errorMsg,
		})
	}

	return s.settingsRepo.Save(ctx, settings)
}

func (s *SearchService) updateLastIndexed(ctx context.Context, remote string) error {
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for i, c := range settings.Search.Configs {
		if c.Remote == remote {
			settings.Search.Configs[i].LastIndexedAt = &now
			settings.Search.Configs[i].Status = "idle"
			settings.Search.Configs[i].ErrorMsg = ""
			break
		}
	}

	return s.settingsRepo.Save(ctx, settings)
}
