package alldebrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"gravity/internal/model"
)

// CheckMagnet uploads magnet to AllDebrid and checks if cached
func (p *AllDebridProvider) CheckMagnet(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
	if !p.IsConfigured() {
		return nil, fmt.Errorf("alldebrid not configured")
	}

	// POST /v4/magnet/upload
	endpoint := fmt.Sprintf("%s/magnet/upload?agent=%s&apikey=%s", baseURL, agent, p.apiKey)

	form := url.Values{}
	form.Set("magnets[]", magnet)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		if result.Error != nil {
			return nil, fmt.Errorf("alldebrid error: %s", result.Error.Message)
		}
		return nil, fmt.Errorf("failed to check magnet")
	}

	if len(result.Data.Magnets) == 0 {
		return nil, fmt.Errorf("no magnet response")
	}

	m := result.Data.Magnets[0]
	if m.Error != nil {
		return nil, fmt.Errorf("alldebrid error: %s", m.Error.Message)
	}

	if !m.Ready {
		// Not cached - caller should fall back to aria2
		return nil, nil
	}

	// Get files
	files, err := p.getMagnetFiles(ctx, fmt.Sprintf("%d", m.ID))
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

// getMagnetFiles retrieves file tree for a magnet
func (p *AllDebridProvider) getMagnetFiles(ctx context.Context, magnetID string) ([]model.MagnetFile, error) {
	// POST /v4/magnet/files
	endpoint := fmt.Sprintf("%s/magnet/files?agent=%s&apikey=%s", baseURL, agent, p.apiKey)

	form := url.Values{}
	form.Set("id[]", magnetID)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
		Data   struct {
			Magnets []struct {
				ID    string            `json:"id"`
				Files []json.RawMessage `json:"files"`
				Error *struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error,omitempty"`
			} `json:"magnets"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" || len(result.Data.Magnets) == 0 {
		return nil, fmt.Errorf("failed to get magnet files")
	}

	if result.Data.Magnets[0].Error != nil {
		return nil, fmt.Errorf("alldebrid error: %s", result.Data.Magnets[0].Error.Message)
	}

	// Parse AllDebrid's nested file format
	return p.parseFiles(result.Data.Magnets[0].Files, "", 1)
}

// parseFiles converts AllDebrid's nested format to MagnetFile array
// AllDebrid format: { n: "name", s: size, l: "link" } for files
//
//	{ n: "name", e: [...] } for folders
func (p *AllDebridProvider) parseFiles(rawFiles []json.RawMessage, parentPath string, startIndex int) ([]model.MagnetFile, error) {
	var files []model.MagnetFile
	index := startIndex

	for _, raw := range rawFiles {
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
			children, err := p.parseFiles(node.E, path, index)
			if err != nil {
				continue
			}

			// Calculate folder size from children
			var folderSize int64
			for _, child := range children {
				folderSize += child.Size
			}

			// Update index based on children count
			for range children {
				if !children[0].IsFolder {
					index++
				}
			}

			files = append(files, model.MagnetFile{
				ID:       fmt.Sprintf("folder_%s", path),
				Name:     node.N,
				Path:     path,
				Size:     folderSize,
				IsFolder: true,
				Children: children,
			})
		} else {
			// File
			files = append(files, model.MagnetFile{
				ID:    node.L, // Use link as ID for easy lookup
				Name:  node.N,
				Path:  path,
				Size:  node.S,
				Link:  node.L,
				Index: index,
			})
			index++
		}
	}

	return files, nil
}

// DeleteMagnet removes a magnet from user's AllDebrid account
func (p *AllDebridProvider) DeleteMagnet(ctx context.Context, magnetID string) error {
	if !p.IsConfigured() {
		return fmt.Errorf("alldebrid not configured")
	}

	endpoint := fmt.Sprintf("%s/magnet/delete?agent=%s&apikey=%s", baseURL, agent, p.apiKey)

	form := url.Values{}
	form.Set("id", magnetID)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
