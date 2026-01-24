package aria2

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"gravity/internal/engine"
	"gravity/internal/model"

	"github.com/anacrolix/torrent/metainfo"
	"go.uber.org/zap"
)

// resolveMetadata uses the engine's persistent metadata client to fetch info
func (e *Engine) resolveMetadata(ctx context.Context, magnet string) (string, error) {
	if e.metadataClient == nil {
		return "", fmt.Errorf("metadata client not initialized")
	}

	t, err := e.metadataClient.AddMagnet(magnet)
	if err != nil {
		return "", err
	}
	defer t.Drop()

	timeout := 60 * time.Second
	e.mu.RLock()
	if e.settings != nil && e.settings.Download.ConnectTimeout > 0 {
		timeout = time.Duration(e.settings.Download.ConnectTimeout) * time.Second
	}
	e.mu.RUnlock()

	select {
	case <-t.GotInfo():
		// Success
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(timeout):
		return "", fmt.Errorf("metadata resolution timeout")
	}

	mi := t.Metainfo()

	// Ensure trackers are present (critical for Aria2 startup speed)
	if u, err := url.Parse(magnet); err == nil {
		trackers := u.Query()["tr"]
		if len(trackers) > 0 {
			e.logger.Debug("adding trackers from magnet link to .torrent", zap.Int("count", len(trackers)))
			// Merge with existing announce list if needed, or just append
			// Simple approach: if AnnounceList is empty, populate it
			if len(mi.AnnounceList) == 0 {
				for _, tr := range trackers {
					mi.AnnounceList = append(mi.AnnounceList, []string{tr})
				}
			}
			// Ensure Announce is set if empty
			if mi.Announce == "" && len(trackers) > 0 {
				mi.Announce = trackers[0]
			}
		}
	}

	var buf bytes.Buffer
	if err := mi.Write(&buf); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func (e *Engine) GetMagnetFiles(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
	e.logger.Debug("fetching magnet metadata using native lib")

	if e.metadataClient == nil {
		return nil, fmt.Errorf("metadata client not initialized")
	}

	t, err := e.metadataClient.AddMagnet(magnet)
	if err != nil {
		return nil, err
	}
	defer t.Drop()

	timeout := 60 * time.Second
	e.mu.RLock()
	if e.settings != nil && e.settings.Download.ConnectTimeout > 0 {
		timeout = time.Duration(e.settings.Download.ConnectTimeout) * time.Second
	}
	e.mu.RUnlock()

	select {
	case <-t.GotInfo():
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout fetching magnet metadata")
	}

	totalSize := t.Length()
	name := t.Name()
	hash := t.InfoHash().String()

	magnetInfo := &model.MagnetInfo{
		Source: "aria2",
		Name:   name,
		Hash:   hash,
		Size:   totalSize,
	}

	for i, f := range t.Files() {
		magnetInfo.Files = append(magnetInfo.Files, &model.MagnetFile{
			ID:       fmt.Sprintf("%d", i+1), // 1-based index for aria2
			Name:     filepath.Base(f.Path()),
			Path:     f.Path(),
			Size:     f.Length(),
			Index:    i + 1,
			IsFolder: false,
		})
	}

	return magnetInfo, nil
}

func (e *Engine) AddMagnetWithSelection(ctx context.Context, magnet string, selectedIndexes []string, opts engine.DownloadOptions) (string, error) {
	// Just forward to Add, which handles metadata resolution and file selection
	return e.Add(ctx, magnet, opts)
}

func (e *Engine) GetTorrentFiles(ctx context.Context, torrentBase64 string) (*model.MagnetInfo, error) {
	mi, err := metainfo.Load(strings.NewReader(torrentBase64))
	if err != nil {
		return nil, fmt.Errorf("failed to parse torrent data: %w", err)
	}

	info, err := mi.UnmarshalInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal torrent info: %w", err)
	}

	magnetInfo := &model.MagnetInfo{
		Source: "aria2",
		Name:   info.Name,
		Hash:   mi.HashInfoBytes().String(),
		Size:   info.TotalLength(),
	}

	for i, f := range info.UpvertedFiles() {
		path := strings.Join(f.Path, "/")
		magnetInfo.Files = append(magnetInfo.Files, &model.MagnetFile{
			ID:       fmt.Sprintf("%d", i+1),
			Name:     filepath.Base(path),
			Path:     path,
			Size:     f.Length,
			Index:    i + 1,
			IsFolder: false,
		})
	}

	return magnetInfo, nil
}
