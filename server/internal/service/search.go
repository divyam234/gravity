package service

import (
	"context"
	"fmt"
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
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			s.checkAutoIndexing()
		}
	}()
}

func (s *SearchService) checkAutoIndexing() {
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

	// Recursive list from storage engine
	// Since StorageEngine interface doesn't have ListRecursive yet, we'll use a local helper
	// or update the interface. I'll update the interface.
	files, err := s.listRecursive(ctx, remote, "")
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
			Filename: f.Name,
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

func (s *SearchService) listRecursive(ctx context.Context, remote, path string) ([]engine.FileInfo, error) {
	// Root of remote
	virtualPath := "/" + remote + "/" + path
	items, err := s.storageEngine.List(ctx, virtualPath)
	if err != nil {
		return nil, err
	}

	var results []engine.FileInfo
	for _, item := range items {
		results = append(results, item)
		if item.IsDir {
			// Subdirectory listing
			// item.Path is like /remote/subfolder
			// we need subfolder part
			subPath := strings.TrimPrefix(item.Path, "/"+remote+"/")
			subItems, err := s.listRecursive(ctx, remote, subPath)
			if err == nil {
				results = append(results, subItems...)
			}
		}
	}
	return results, nil
}

func (s *SearchService) Search(ctx context.Context, query string) ([]store.IndexedFile, error) {
	return s.repo.Search(ctx, query)
}

func (s *SearchService) GetConfigs(ctx context.Context) ([]store.RemoteIndexConfig, error) {
	return s.repo.GetConfigs(ctx)
}

func (s *SearchService) UpdateConfig(ctx context.Context, remote string, interval int) error {
	return s.repo.SaveConfig(ctx, store.RemoteIndexConfig{
		Remote:               remote,
		AutoIndexIntervalMin: interval,
		Status:               "idle",
	})
}
