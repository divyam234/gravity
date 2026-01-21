package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"gravity/internal/engine"
	"gravity/internal/store"

	"github.com/google/uuid"
)

type SearchService struct {
	repo          *store.SearchRepo
	storageEngine engine.StorageEngine
	mu            sync.Mutex
	isIndexing    map[string]bool
}

func NewSearchService(repo *store.SearchRepo, storage engine.StorageEngine) *SearchService {
	return &SearchService{
		repo:          repo,
		storageEngine: storage,
		isIndexing:    make(map[string]bool),
	}
}

func (s *SearchService) Start() {
	if s == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			s.checkAutoIndexing()
		}
	}()
}

func (s *SearchService) checkAutoIndexing() {
	if s == nil || s.repo == nil {
		return
	}
	ctx := context.Background()
	configs, err := s.repo.GetConfigs(ctx)
	if err != nil {
		return
	}

	for _, c := range configs {
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
			go s.IndexRemote(ctx, c.Remote)
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

	defer func() {
		s.mu.Lock()
		delete(s.isIndexing, remote)
		s.mu.Unlock()
	}()

	s.repo.UpdateStatus(ctx, remote, "indexing", "")

	// Fetch current config for filtering
	configs, err := s.GetConfigs(ctx)
	var config store.RemoteIndexConfig
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
		s.repo.UpdateStatus(ctx, remote, "error", err.Error())
		return err
	}

	// Convert to store format
	indexedFiles := make([]store.IndexedFile, len(files))
	for i, f := range files {
		indexedFiles[i] = store.IndexedFile{
			ID:       uuid.New().String(),
			Remote:   remote,
			Path:     f.Path,
			Name:     f.Name,
			Size:     f.Size,
			ModTime:  f.ModTime,
			IsDir:    f.IsDir,
		}
	}

	if err := s.repo.SaveFiles(ctx, remote, indexedFiles); err != nil {
		s.repo.UpdateStatus(ctx, remote, "error", err.Error())
		return err
	}

	return s.repo.UpdateLastIndexed(ctx, remote)
}

func (s *SearchService) listRecursive(ctx context.Context, remote, path string, config store.RemoteIndexConfig) ([]engine.FileInfo, error) {
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

func (s *SearchService) Search(ctx context.Context, query string, limit, offset int) ([]store.IndexedFile, int, error) {
	return s.repo.Search(ctx, query, limit, offset)
}

func (s *SearchService) GetConfigs(ctx context.Context) ([]store.RemoteIndexConfig, error) {
	dbConfigs, err := s.repo.GetConfigs(ctx)
	if err != nil {
		return nil, err
	}

	remotes, err := s.storageEngine.ListRemotes(ctx)
	if err != nil {
		return dbConfigs, nil // Fallback to what we have in DB if engine fails
	}

	configMap := make(map[string]store.RemoteIndexConfig)
	for _, c := range dbConfigs {
		configMap[c.Remote] = c
	}

	var results []store.RemoteIndexConfig
	for _, r := range remotes {
		if cfg, ok := configMap[r.Name]; ok {
			results = append(results, cfg)
		} else {
			// Provide default config for discovered remote not yet in DB
			results = append(results, store.RemoteIndexConfig{
				Remote:               r.Name,
				AutoIndexIntervalMin: 0,
				Status:               "idle",
			})
		}
	}

	return results, nil
}

func (s *SearchService) UpdateConfig(ctx context.Context, remote string, interval int, excludedPatterns, includedExtensions string, minSize int64) error {
	return s.repo.SaveConfig(ctx, store.RemoteIndexConfig{
		Remote:               remote,
		AutoIndexIntervalMin: interval,
		Status:               "idle",
		ExcludedPatterns:     excludedPatterns,
		IncludedExtensions:   includedExtensions,
		MinSizeBytes:         minSize,
	})
}

func (s *SearchService) BatchUpdateConfig(ctx context.Context, configs map[string]store.RemoteIndexConfig) error {
	for remote, cfg := range configs {
		cfg.Remote = remote
		if err := s.repo.SaveConfig(ctx, cfg); err != nil {
			return err
		}
	}
	return nil
}
