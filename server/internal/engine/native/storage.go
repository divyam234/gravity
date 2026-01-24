package native

import (
	"context"
	"sync"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
)

type DynamicStorage struct {
	baseDir    string
	completion storage.PieceCompletion
	dirMap     sync.Map // map[string]string (HashHex -> Directory)
}

func NewDynamicStorage(baseDir, metadataDir string) *DynamicStorage {
	// Initialize global completion storage in metadata dir
	// This prevents .torrent.db files appearing in download directories
	pc, err := storage.NewDefaultPieceCompletionForDir(metadataDir)
	if err != nil {
		// Fallback to in-memory if DB fails
		pc = storage.NewMapPieceCompletion()
	}

	return &DynamicStorage{
		baseDir:    baseDir,
		completion: pc,
	}
}

// Register maps an infohash to a specific directory
func (s *DynamicStorage) Register(infoHash string, dir string) {
	if dir != "" {
		s.dirMap.Store(infoHash, dir)
	}
}

func (s *DynamicStorage) OpenTorrent(ctx context.Context, info *metainfo.Info, infoHash metainfo.Hash) (storage.TorrentImpl, error) {
	dir := s.baseDir

	// Check exact hex string (standard)
	if val, ok := s.dirMap.Load(infoHash.HexString()); ok {
		dir = val.(string)
	}

	// Use NewFileOpts to specify the content directory AND the shared completion storage
	opts := storage.NewFileClientOpts{
		ClientBaseDir:   dir,
		PieceCompletion: s.completion,
	}

	// OpenTorrent on the configured file client
	return storage.NewFileOpts(opts).OpenTorrent(ctx, info, infoHash)
}

func (s *DynamicStorage) Close() error {
	if s.completion != nil {
		return s.completion.Close()
	}
	return nil
}
