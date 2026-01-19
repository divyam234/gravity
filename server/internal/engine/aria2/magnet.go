package aria2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"gravity/internal/engine"
	"gravity/internal/model"
)

// GetMagnetFiles fetches metadata for a magnet and returns file list
func (e *Engine) GetMagnetFiles(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
	log.Printf("[aria2] Starting metadata fetch for magnet")
	// Add magnet with metadata-only mode
	res, err := e.client.Call(ctx, "aria2.addUri", []string{magnet}, map[string]interface{}{
		"bt-metadata-only": "true",
		"bt-save-metadata": "true",
	})
	if err != nil {
		return nil, err
	}

	var gid string
	if err := json.Unmarshal(res, &gid); err != nil {
		return nil, err
	}
	log.Printf("[aria2] Magnet added with GID: %s", gid)

	defer e.client.Call(context.Background(), "aria2.remove", gid) // Clean up after getting metadata

	// Wait for metadata to be fetched (poll status)
	var info *model.MagnetInfo
	for i := 0; i < 60; i++ { // Max 60 seconds
		log.Printf("[aria2] Polling metadata status (attempt %d/60)...", i+1)
		statusRes, err := e.client.Call(ctx, "aria2.tellStatus", gid)
		if err != nil {
			log.Printf("[aria2] tellStatus error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		var status struct {
			Status     string `json:"status"`
			BitTorrent struct {
				Info struct {
					Name string `json:"name"`
				} `json:"info"`
			} `json:"bittorrent"`
		}
		if err := json.Unmarshal(statusRes, &status); err != nil {
			log.Printf("[aria2] unmarshal error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("[aria2] Current status: %s, Torrent name: %s", status.Status, status.BitTorrent.Info.Name)

		// Check if we have bittorrent info
		if status.BitTorrent.Info.Name != "" {
			name := status.BitTorrent.Info.Name
			log.Printf("[aria2] Metadata retrieved! Name: %s", name)

			// Get files
			filesRes, err := e.client.Call(ctx, "aria2.getFiles", gid)
			if err != nil {
				return nil, err
			}

			var files []struct {
				Index  string `json:"index"`
				Path   string `json:"path"`
				Length string `json:"length"`
			}
			if err := json.Unmarshal(filesRes, &files); err != nil {
				return nil, err
			}

			log.Printf("[aria2] Found %d files", len(files))

			var magnetFiles []model.MagnetFile
			var totalSize int64

			for _, f := range files {
				size := parseSize(f.Length)

				// Extract relative path (remove download dir prefix)
				relPath := f.Path
				if idx := strings.Index(f.Path, name); idx >= 0 {
					relPath = f.Path[idx:]
				}

				magnetFiles = append(magnetFiles, model.MagnetFile{
					ID:       f.Index, // 1-indexed for --select-file
					Name:     extractFilename(relPath),
					Path:     relPath,
					Size:     size,
					IsFolder: false,
				})
				totalSize += size
			}

			// Extract hash from magnet
			hash := extractHashFromMagnet(magnet)

			info = &model.MagnetInfo{
				Source: "aria2",
				Cached: false,
				Name:   name,
				Hash:   hash,
				Size:   totalSize,
				Files:  magnetFiles,
			}
			break
		}

		// Check for errors
		if status.Status == "error" {
			log.Printf("[aria2] Metadata fetch failed with error status")
			return nil, fmt.Errorf("failed to fetch magnet metadata")
		}

		time.Sleep(1 * time.Second)
	}

	if info == nil {
		log.Printf("[aria2] Metadata fetch timed out after 60 seconds")
		return nil, fmt.Errorf("timeout fetching magnet metadata")
	}

	return info, nil
}

// AddMagnetWithSelection starts a magnet download with selected files only
func (e *Engine) AddMagnetWithSelection(ctx context.Context, magnet string, selectedIndexes []string, opts engine.DownloadOptions) (string, error) {
	// Build select-file string (comma-separated 1-indexed)
	selectFile := strings.Join(selectedIndexes, ",")

	options := map[string]interface{}{
		"select-file": selectFile,
	}

	if opts.Dir != "" {
		options["dir"] = opts.Dir
	}

	result, err := e.client.Call(ctx, "aria2.addUri", []string{magnet}, options)
	if err != nil {
		return "", err
	}

	var gid string
	if err := json.Unmarshal(result, &gid); err != nil {
		return "", err
	}
	return gid, nil
}

// GetTorrentFiles extracts metadata from a .torrent file (base64 encoded)
func (e *Engine) GetTorrentFiles(ctx context.Context, torrentBase64 string) (*model.MagnetInfo, error) {
	log.Printf("[aria2] Extracting metadata from .torrent file")
	res, err := e.client.Call(ctx, "aria2.addTorrent", torrentBase64, []interface{}{}, map[string]interface{}{
		"paused": "true",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add torrent: %w", err)
	}

	var gid string
	if err := json.Unmarshal(res, &gid); err != nil {
		return nil, err
	}

	defer e.client.Call(context.Background(), "aria2.remove", gid)

	statusRes, err := e.client.Call(ctx, "aria2.tellStatus", gid)
	if err != nil {
		return nil, err
	}

	var status struct {
		Status     string `json:"status"`
		InfoHash   string `json:"infoHash"`
		BitTorrent struct {
			Info struct {
				Name string `json:"name"`
			} `json:"info"`
		} `json:"bittorrent"`
	}
	if err := json.Unmarshal(statusRes, &status); err != nil {
		return nil, err
	}

	filesRes, err := e.client.Call(ctx, "aria2.getFiles", gid)
	if err != nil {
		return nil, err
	}

	var files []struct {
		Index  string `json:"index"`
		Path   string `json:"path"`
		Length string `json:"length"`
	}
	if err := json.Unmarshal(filesRes, &files); err != nil {
		return nil, err
	}

	var magnetFiles []model.MagnetFile
	var totalSize int64
	for _, f := range files {
		size := parseSize(f.Length)

		magnetFiles = append(magnetFiles, model.MagnetFile{
			ID:       f.Index,
			Name:     extractFilename(f.Path),
			Path:     f.Path,
			Size:     size,
			IsFolder: false,
		})
		totalSize += size
	}

	return &model.MagnetInfo{
		Source: "aria2",
		Cached: false,
		Name:   status.BitTorrent.Info.Name,
		Hash:   status.InfoHash,
		Size:   totalSize,
		Files:  magnetFiles,
	}, nil
}

func extractHashFromMagnet(magnet string) string {
	// Extract btih hash from magnet URI
	if idx := strings.Index(magnet, "btih:"); idx >= 0 {
		hash := magnet[idx+5:]
		if ampIdx := strings.Index(hash, "&"); ampIdx >= 0 {
			hash = hash[:ampIdx]
		}
		return strings.ToLower(hash)
	}
	return ""
}

func extractFilename(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func parseSize(v interface{}) int64 {
	switch val := v.(type) {
	case string:
		var size int64
		fmt.Sscanf(val, "%d", &size)
		return size
	case float64:
		return int64(val)
	case int64:
		return val
	}
	return 0
}
