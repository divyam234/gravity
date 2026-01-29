package ytdlp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gravity/internal/client"
	"gravity/internal/model"
	"gravity/internal/provider"

	"github.com/rclone/rclone/lib/rest"
)

type YtDlpProvider struct {
	binaryPath     string
	extractors     map[string]bool
	extractorsOnce sync.Once
	client         *client.Client
}

func New() *YtDlpProvider {
	return &YtDlpProvider{
		binaryPath: "yt-dlp",
		extractors: make(map[string]bool),
		client:     client.New(context.Background(), "", client.WithTimeout(15*time.Second)),
	}
}

func (p *YtDlpProvider) Name() string        { return "ytdlp" }
func (p *YtDlpProvider) DisplayName() string { return "yt-dlp" }
func (p *YtDlpProvider) Type() model.ProviderType {
	return model.ProviderTypeFileHost
}

func (p *YtDlpProvider) ConfigSchema() []provider.ConfigField {
	return []provider.ConfigField{}
}

func (p *YtDlpProvider) Configure(ctx context.Context, config map[string]string) error {
	return nil
}

func (p *YtDlpProvider) IsConfigured() bool {
	return true
}

func (p *YtDlpProvider) loadExtractors() {
	p.extractorsOnce.Do(func() {
		cmd := exec.Command(p.binaryPath, "--list-extractors")
		output, err := cmd.Output()
		if err != nil {
			return
		}

		scanner := bufio.NewScanner(bytes.NewReader(output))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || line == "generic" || strings.Contains(strings.ToLower(line), "broken") {
				continue
			}

			// Handle namespaced extractors like "twitch:stream" -> "twitch"
			if idx := strings.Index(line, ":"); idx != -1 {
				line = line[:idx]
			}

			p.extractors[strings.ToLower(line)] = true
		}

		// Manual aliases for shorteners or mismatches
		p.extractors["youtu"] = true
	})
}

func (p *YtDlpProvider) Supports(rawURL string) bool {
	p.loadExtractors()

	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	hostname := strings.ToLower(u.Hostname())
	hostname = strings.TrimPrefix(hostname, "www.")

	parts := strings.FieldsFunc(hostname, func(r rune) bool {
		return r == '.' || r == '-'
	})

	for _, part := range parts {
		if p.extractors[part] {
			return true
		}
	}

	return false
}

func (p *YtDlpProvider) Priority() int {
	return 10
}

type YtDlpJSON struct {
	URL            string            `json:"url"`
	Title          string            `json:"title"`
	Ext            string            `json:"ext"`
	Filesize       *int64            `json:"filesize"`
	FilesizeApprox *int64            `json:"filesize_approx"`
	HttpHeaders    map[string]string `json:"http_headers"`
	Filename       string            `json:"_filename"`
	Cookies        string            `json:"cookies"`
}

func (p *YtDlpProvider) Resolve(ctx context.Context, rawURL string, headers map[string]string) (*provider.ResolveResult, error) {
	// We want the best format that is a single file (no merging required)
	// so that aria2 can handle it as a standard HTTP download.
	args := []string{
		"-j",
		"--no-playlist",
		"-f", "best",
		"-o", "%(title)s.%(ext)s", // Force filename format
	}

	// Pass input headers to yt-dlp
	for k, v := range headers {
		args = append(args, "--add-header", fmt.Sprintf("%s:%s", k, v))
	}

	args = append(args, rawURL)

	cmd := exec.CommandContext(ctx, p.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("yt-dlp failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("yt-dlp failed: %w", err)
	}

	var meta YtDlpJSON
	if err := json.Unmarshal(output, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse yt-dlp output: %w", err)
	}

	// Ensure headers map exists
	if meta.HttpHeaders == nil {
		meta.HttpHeaders = make(map[string]string)
	}

	// Move Cookies from JSON field to Header if present
	if meta.Cookies != "" {
		meta.HttpHeaders["Cookie"] = meta.Cookies
	}

	var size int64
	if meta.Filesize != nil {
		size = *meta.Filesize
	} else if meta.FilesizeApprox != nil {
		size = *meta.FilesizeApprox
	}

	// 1. Perform HEAD/Range check to get exact size and final URL
	// This ensures we have the final redirected URL and accurate size for the engine
	finalURL := meta.URL

	// Prepare rest.Opts for the request
	opts := rest.Opts{
		Method:       "GET",
		RootURL:      meta.URL,
		ExtraHeaders: make(map[string]string),
	}

	maps.Copy(opts.ExtraHeaders, meta.HttpHeaders)
	opts.ExtraHeaders["Range"] = "bytes=0-0"

	// Use p.client.Call
	resp, err := p.client.Call(ctx, &opts)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			finalURL = resp.Request.URL.String()

			// Parse size from Content-Range or Content-Length if size is still 0
			if size == 0 {
				if cr := resp.Header.Get("Content-Range"); cr != "" {
					parts := strings.Split(cr, "/")
					if len(parts) == 2 {
						fmt.Sscanf(parts[1], "%d", &size)
					}
				}
				if size == 0 {
					size = resp.ContentLength
				}
			}
		}
	}

	// Filename is now pre-formatted by yt-dlp via -o
	filename := meta.Filename
	if filename == "" {
		// Fallback just in case
		filename = meta.Title
		if meta.Ext != "" {
			filename = fmt.Sprintf("%s.%s", meta.Title, meta.Ext)
		}
	}

	// Basic sanitization is still good practice for FS safety
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")

	return &provider.ResolveResult{
		URL:     finalURL,
		Name:    filename,
		Size:    size,
		Headers: meta.HttpHeaders,
	}, nil
}

func (p *YtDlpProvider) Test(ctx context.Context) (*model.AccountInfo, error) {
	cmd := exec.CommandContext(ctx, p.binaryPath, "--version")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp not found: %w", err)
	}
	return &model.AccountInfo{Username: "System", IsPremium: true}, nil
}
