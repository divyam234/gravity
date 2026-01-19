package aria2

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gravity/internal/engine"
	"gravity/internal/model"
)

// GetMagnetFiles fetches metadata for a magnet and returns file list
func (e *Engine) GetMagnetFiles(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
	// Add magnet with metadata-only mode
	ariaOpts := map[string]interface{}{
		"bt-metadata-only": "true",
		"bt-save-metadata": "true",
	}

	res, err := e.client.Call(ctx, "aria2.addUri", []string{magnet}, ariaOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to add magnet: %w", err)
	}

	var gid string
	if err := json.Unmarshal(res, &gid); err != nil {
		return nil, fmt.Errorf("failed to parse gid: %w", err)
	}

	// Ensure cleanup when done
	defer func() {
		e.client.Call(context.Background(), "aria2.forceRemove", gid)
		e.client.Call(context.Background(), "aria2.removeDownloadResult", gid)
	}()

	// Wait for metadata to be fetched (poll status)
	var info *model.MagnetInfo
	for i := 0; i < 60; i++ { // Max 60 seconds
		statusRes, err := e.client.Call(ctx, "aria2.tellStatus", gid)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		var status struct {
			Gid        string `json:"gid"`
			Status     string `json:"status"`
			InfoHash   string `json:"infoHash"`
			Dir        string `json:"dir"`
			BitTorrent *struct {
				Info *struct {
					Name string `json:"name"`
				} `json:"info"`
			} `json:"bittorrent"`
			Files []struct {
				Index  string `json:"index"`
				Path   string `json:"path"`
				Length string `json:"length"`
			} `json:"files"`
			ErrorMessage string `json:"errorMessage"`
		}

		if err := json.Unmarshal(statusRes, &status); err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		// Check for errors
		if status.Status == "error" {
			return nil, fmt.Errorf("failed to fetch magnet metadata: %s", status.ErrorMessage)
		}

		// Check if we have bittorrent info
		if status.BitTorrent != nil && status.BitTorrent.Info != nil && len(status.Files) > 0 {
			name := status.BitTorrent.Info.Name

			var magnetFiles []model.MagnetFile
			var totalSize int64

			for _, f := range status.Files {
				size, _ := strconv.ParseInt(f.Length, 10, 64)
				index, _ := strconv.Atoi(f.Index)

				// Extract relative path (remove download dir prefix)
				relPath := f.Path
				if status.Dir != "" && strings.HasPrefix(f.Path, status.Dir) {
					relPath = strings.TrimPrefix(f.Path, status.Dir)
					relPath = strings.TrimPrefix(relPath, "/")
				}

				// If the file is directly in the torrent folder, use just the filename
				filename := filepath.Base(f.Path)

				magnetFiles = append(magnetFiles, model.MagnetFile{
					ID:    f.Index, // 1-indexed for --select-file
					Name:  filename,
					Path:  relPath,
					Size:  size,
					Index: index,
				})
				totalSize += size
			}

			// Extract hash from magnet
			hash := extractHashFromMagnet(magnet)
			if status.InfoHash != "" {
				hash = status.InfoHash
			}

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

		time.Sleep(1 * time.Second)
	}

	if info == nil {
		return nil, fmt.Errorf("timeout fetching magnet metadata")
	}

	return info, nil
}

// AddMagnetWithSelection starts a magnet download with selected files only
func (e *Engine) AddMagnetWithSelection(ctx context.Context, magnet string, selectedIndexes []string, opts engine.DownloadOptions) (string, error) {
	ariaOpts := make(map[string]interface{})

	// Build select-file string (comma-separated 1-indexed)
	if len(selectedIndexes) > 0 {
		ariaOpts["select-file"] = strings.Join(selectedIndexes, ",")
	}

	if opts.Dir != "" {
		ariaOpts["dir"] = opts.Dir
	}

	res, err := e.client.Call(ctx, "aria2.addUri", []string{magnet}, ariaOpts)
	if err != nil {
		return "", err
	}

	var gid string
	if err := json.Unmarshal(res, &gid); err != nil {
		return "", err
	}

	return gid, nil
}

// extractHashFromMagnet extracts btih hash from magnet URI
func extractHashFromMagnet(magnet string) string {
	// Look for btih: in the magnet URI
	if idx := strings.Index(strings.ToLower(magnet), "btih:"); idx >= 0 {
		hash := magnet[idx+5:]
		if ampIdx := strings.Index(hash, "&"); ampIdx >= 0 {
			hash = hash[:ampIdx]
		}
		return strings.ToLower(hash)
	}
	return ""
}
