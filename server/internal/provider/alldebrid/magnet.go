package alldebrid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/rclone/rclone/lib/rest"
	"gravity/internal/model"
)

// CheckMagnet uploads magnet to AllDebrid and checks if cached
func (p *AllDebridProvider) CheckMagnet(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
	// POST /v4/magnet/upload
	
	form := url.Values{}
	form.Set("magnets[]", magnet)

	params := url.Values{}
	params.Set("agent", agent)
	params.Set("apikey", p.apiKey)

	opts := rest.Opts{
		Method:      "POST",
		Path:        "/magnet/upload",
		Parameters:  params,
		ContentType: "application/x-www-form-urlencoded",
		Body:        strings.NewReader(form.Encode()),
	}

	return p.handleUploadResponse(ctx, &opts)
}

// CheckTorrentFile uploads .torrent file to AllDebrid and checks status
func (p *AllDebridProvider) CheckTorrentFile(ctx context.Context, fileData []byte) (*model.MagnetInfo, error) {
	// POST /v4/magnet/upload/file
	
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files[]", "upload.torrent")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(fileData); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("agent", agent)
	params.Set("apikey", p.apiKey)

	opts := rest.Opts{
		Method:      "POST",
		Path:        "/magnet/upload/file",
		Parameters:  params,
		ContentType: writer.FormDataContentType(),
		Body:        body,
	}

	return p.handleUploadResponse(ctx, &opts)
}

func (p *AllDebridProvider) handleUploadResponse(ctx context.Context, opts *rest.Opts) (*model.MagnetInfo, error) {
	var result struct {
		Status string `json:"status"`
		Data   struct {
			Magnets []struct {
				Magnet string `json:"magnet"`
				Hash   string `json:"hash"`
				Name   string `json:"name"`
				Size   int64  `json:"size"`
				Ready  bool   `json:"ready"`
				ID     int64  `json:"id"`
				Error  *struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error,omitempty"`
			} `json:"magnets"`
		} `json:"data"`
	}

	_, err := p.client.CallJSON(ctx, opts, nil, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" || len(result.Data.Magnets) == 0 {
		return nil, fmt.Errorf("failed to upload/check")
	}

	m := result.Data.Magnets[0]
	if m.Error != nil {
		return nil, fmt.Errorf("alldebrid error: %s", m.Error.Message)
	}

	if !m.Ready {
		// Not cached
		return &model.MagnetInfo{
			Source:   "alldebrid",
			Cached:   false,
			MagnetID: fmt.Sprintf("%d", m.ID),
			Name:     m.Name,
			Hash:     m.Hash,
			Size:     m.Size,
		}, nil
	}

	// Get files
	files, err := p.GetMagnetFiles(ctx, fmt.Sprintf("%d", m.ID))
	if err != nil {
		return nil, err
	}

	return &model.MagnetInfo{
		Source:   "alldebrid",
		Cached:   true,
		MagnetID: fmt.Sprintf("%d", m.ID),
		Name:     m.Name,
		Hash:     m.Hash,
		Size:     m.Size,
		Files:    files,
	}, nil
}

// GetMagnetFiles retrieves file tree for a magnet
func (p *AllDebridProvider) GetMagnetFiles(ctx context.Context, magnetID string) ([]*model.MagnetFile, error) {
	// GET /v4/magnet/status
	params := url.Values{}
	params.Set("agent", agent)
	params.Set("apikey", p.apiKey)
	params.Set("id", magnetID)

	opts := rest.Opts{
		Method:     "GET",
		Path:       "/magnet/status",
		Parameters: params,
	}

	var result struct {
		Status string `json:"status"`
		Data   struct {
			Magnets struct {
				ID    int               `json:"id"`
				Files []json.RawMessage `json:"files"`
			} `json:"magnets"`
		} `json:"data"`
	}

	_, err := p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("failed to get magnet status")
	}

	// Parse AllDebrid's nested file format
	return p.parseFiles(result.Data.Magnets.Files, "")
}

// parseFiles converts AllDebrid's nested format to flat MagnetFile array
func (p *AllDebridProvider) parseFiles(rawFiles []json.RawMessage, parentPath string) ([]*model.MagnetFile, error) {
	var files []*model.MagnetFile

	for i, raw := range rawFiles {
		var node struct {
			N string            `json:"n"` // name
			S int64             `json:"s"` // size (file only)
			L string            `json:"l"` // link (file only)
			E []json.RawMessage `json:"e"` // children (folder only)
		}

		if err := json.Unmarshal(raw, &node); err != nil {
			continue
		}

		path := node.N
		if parentPath != "" {
			path = parentPath + "/" + node.N
		}

		if len(node.E) > 0 {
			// Folder
			children, err := p.parseFiles(node.E, path)
			if err != nil {
				continue
			}

			// Calculate folder size
			var folderSize int64
			for _, child := range children {
				folderSize += child.Size
			}

			files = append(files, &model.MagnetFile{
				ID:       fmt.Sprintf("folder_%d_%s", i, path),
				Name:     node.N,
				Path:     path,
				Size:     folderSize,
				IsFolder: true,
				Children: children,
			})
		} else {
			// File
			files = append(files, &model.MagnetFile{
				ID:       node.L, // Use link as ID for easy lookup
				Name:     node.N,
				Path:     path,
				Size:     node.S,
				Link:     node.L,
				IsFolder: false,
			})
		}
	}

	return files, nil
}

// DeleteMagnet removes a magnet from user's AllDebrid account
func (p *AllDebridProvider) DeleteMagnet(ctx context.Context, magnetID string) error {
	params := url.Values{}
	params.Set("agent", agent)
	params.Set("apikey", p.apiKey)
	
	form := url.Values{}
	form.Set("id", magnetID)

	opts := rest.Opts{
		Method:      "POST",
		Path:        "/magnet/delete",
		Parameters:  params,
		ContentType: "application/x-www-form-urlencoded",
		Body:        strings.NewReader(form.Encode()),
	}

	_, err := p.client.CallJSON(ctx, &opts, nil, nil)
	return err
}
