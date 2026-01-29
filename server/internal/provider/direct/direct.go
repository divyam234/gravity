package direct

import (
	"context"
	"fmt"
	"mime"
	"net/textproto"
	"path"
	"strings"
	"time"

	"gravity/internal/client"
	"gravity/internal/model"
	"gravity/internal/provider"

	"github.com/rclone/rclone/lib/rest"
)

type DirectProvider struct {
	client *client.Client
}

func New() *DirectProvider {
	return &DirectProvider{
		client: client.New(context.Background(), "", client.WithTimeout(30*time.Second)),
	}
}

func (p *DirectProvider) Name() string        { return "direct" }
func (p *DirectProvider) DisplayName() string { return "Direct Download" }
func (p *DirectProvider) Type() model.ProviderType {
	return model.ProviderTypeDirect
}

func (p *DirectProvider) ConfigSchema() []provider.ConfigField {
	return []provider.ConfigField{}
}

func (p *DirectProvider) Configure(ctx context.Context, config map[string]string) error {
	return nil
}

func (p *DirectProvider) IsConfigured() bool {
	return true
}

func (p *DirectProvider) Supports(rawURL string) bool {
	return strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") || strings.HasPrefix(rawURL, "magnet:")
}

func (p *DirectProvider) Priority() int {
	return 0 // Lowest priority
}

func (p *DirectProvider) Resolve(ctx context.Context, rawURL string, headers map[string]string) (*provider.ResolveResult, error) {
	if strings.HasPrefix(rawURL, "magnet:") {
		return &provider.ResolveResult{
			URL:  rawURL,
			Name: "",
		}, nil
	}

	// Mimic rclone copyURLFn logic: Perform GET request
	// We use Range: bytes=0-0 to get headers and size without full download
	opts := rest.Opts{
		Method:  "GET",
		RootURL: rawURL,
		ExtraHeaders: map[string]string{
			"Range": "bytes=0-0",
		},
	}

	// Use the persistent client
	resp, err := p.client.Call(ctx, &opts)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Resolve failed: %s", resp.Status)
	}

	// 1. Determine Filename (Logic from copyURLFn)
	var filename string

	// Check Content-Disposition
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if val, ok := params["filename"]; ok {

				filename = textproto.TrimString(path.Base(strings.ReplaceAll(val, "\\", "/")))
			}
		}
	}

	// Fallback to URL path
	if filename == "" {
		filename = path.Base(resp.Request.URL.Path)
		if filename == "." || filename == "/" {
			return nil, fmt.Errorf("Resolve failed: file name wasn't found in url")
		}
	}

	// 2. Determine Size
	// Since we used Range request, Content-Length is partial.
	// We must parse Content-Range for total size.
	// Format: bytes start-end/total
	var size int64 = -1
	if cr := resp.Header.Get("Content-Range"); cr != "" {
		parts := strings.Split(cr, "/")
		if len(parts) == 2 {
			fmt.Sscanf(parts[1], "%d", &size)
		}
	}

	// If Content-Range didn't give size (e.g. server ignored range), use Content-Length
	if size == -1 {
		size = resp.ContentLength
	}

	return &provider.ResolveResult{
		URL:  rawURL,
		Name: filename,
		Size: size,
	}, nil
}

func (p *DirectProvider) Test(ctx context.Context) (*model.AccountInfo, error) {
	return &model.AccountInfo{
		Username:  "Guest",
		IsPremium: false,
	}, nil
}
