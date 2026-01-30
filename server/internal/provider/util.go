package provider

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/textproto"
	"path"
	"strings"
	"time"

	"gravity/internal/client"

	"github.com/rclone/rclone/lib/rest"
)

func FetchMetadata(ctx context.Context, c *client.Client, rawURL string) (*ResolveResult, error) {
	headOpts := rest.Opts{
		Method:  "HEAD",
		RootURL: rawURL,
	}

	resp, err := c.Call(ctx, &headOpts)
	if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		defer resp.Body.Close()
		return parseMetadataResponse(resp, rawURL), nil
	}
	if resp != nil {
		resp.Body.Close()
	}

	getOpts := rest.Opts{
		Method:  "GET",
		RootURL: rawURL,
		ExtraHeaders: map[string]string{
			"Range": "bytes=0-0",
		},
	}

	resp, err = c.Call(ctx, &getOpts)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("metadata fetch failed: %s", resp.Status)
	}

	return parseMetadataResponse(resp, rawURL), nil
}

func parseMetadataResponse(resp *http.Response, rawURL string) *ResolveResult {
	var filename string
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if val, ok := params["filename"]; ok {
				filename = textproto.TrimString(path.Base(strings.ReplaceAll(val, "\\", "/")))
			}
		}
	}
	if filename == "" {
		filename = path.Base(resp.Request.URL.Path)
	}

	size := rest.ParseSizeFromHeaders(resp.Header)

	var modTime time.Time
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		if t, err := http.ParseTime(lm); err == nil {
			modTime = t
		}
	}

	return &ResolveResult{
		URL:     rawURL,
		Name:    filename,
		Size:    size,
		ModTime: modTime,
	}
}
