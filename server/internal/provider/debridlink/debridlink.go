package debridlink

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rclone/rclone/lib/rest"
	"gravity/internal/client"
	"gravity/internal/model"
	"gravity/internal/provider"
)

const baseURL = "https://debrid-link.com/api/v2"

type DebridLinkProvider struct {
	apiKey   string
	proxyURL string
	client   *client.Client
}

func New() *DebridLinkProvider {
	return &DebridLinkProvider{}
}

func (p *DebridLinkProvider) Name() string        { return "debridlink" }
func (p *DebridLinkProvider) DisplayName() string { return "Debrid-Link" }
func (p *DebridLinkProvider) Type() model.ProviderType {
	return model.ProviderTypeDebrid
}

func (p *DebridLinkProvider) ConfigSchema() []provider.ConfigField {
	return []provider.ConfigField{
		{
			Key:         "api_key",
			Label:       "API Key",
			Type:        "password",
			Required:    true,
			Description: "Get your API Key from debrid-link.com/webapp/apikey",
		},
		{
			Key:         "proxy_url",
			Label:       "Proxy URL",
			Type:        "text",
			Required:    false,
			Description: "Optional proxy (http://user:pass@host:port)",
		},
	}
}

func (p *DebridLinkProvider) Configure(ctx context.Context, config map[string]string) error {
	p.apiKey = config["api_key"]
	p.proxyURL = config["proxy_url"]

	opts := []client.Option{
		client.WithTimeout(10 * time.Second),
	}

	if p.proxyURL != "" {
		opts = append(opts, client.WithProxy(p.proxyURL))
	}

	p.client = client.New(ctx, baseURL, opts...)
	return nil
}

func (p *DebridLinkProvider) IsConfigured() bool {
	return p.apiKey != ""
}

func (p *DebridLinkProvider) Supports(rawURL string) bool {
	return strings.HasPrefix(rawURL, "magnet:") || strings.Contains(rawURL, "debrid-link.com")
}

func (p *DebridLinkProvider) Priority() int {
	return 85
}

func (p *DebridLinkProvider) Resolve(ctx context.Context, rawURL string, headers map[string]string) (*provider.ResolveResult, error) {
	// Try /downloader/add for hoster links
	form := url.Values{}
	form.Set("url", rawURL)

	opts := rest.Opts{
		Method:      "POST",
		Path:        "/downloader/add",
		ContentType: "application/x-www-form-urlencoded",
		Body:        strings.NewReader(form.Encode()),
		ExtraHeaders: map[string]string{
			"Authorization": "Bearer " + p.apiKey,
		},
	}

	var result struct {
		Success bool
		Value   struct {
			DownloadLink string
			Name         string
			Size         int64
		}
		Error string
	}

	_, err := p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("debrid-link error: %s", result.Error)
	}

	return &provider.ResolveResult{
		URL:  result.Value.DownloadLink,
		Name: result.Value.Name,
		Size: result.Value.Size,
	}, nil
}

func (p *DebridLinkProvider) Test(ctx context.Context) (*model.AccountInfo, error) {
	opts := rest.Opts{
		Method: "GET",
		Path:   "/account/infos",
		ExtraHeaders: map[string]string{
			"Authorization": "Bearer " + p.apiKey,
		},
	}

	var result struct {
		Success bool
		Value   struct {
			Pseudo      string
			AccountType int // 1 = premium?
			Expiration  int64
		}
	}

	_, err := p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("debrid-link error")
	}

	expiresAt := time.Unix(result.Value.Expiration, 0)
	return &model.AccountInfo{
		Username:  result.Value.Pseudo,
		IsPremium: result.Value.AccountType == 1, // Assuming 1 is premium
		ExpiresAt: &expiresAt,
	}, nil
}

func (p *DebridLinkProvider) GetHosts(ctx context.Context) ([]string, error) {
	return []string{}, nil
}
