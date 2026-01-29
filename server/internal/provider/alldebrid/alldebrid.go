package alldebrid

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

const (
	baseURL = "https://api.alldebrid.com/v4.1"
	agent   = "gravity"
)

type AllDebridProvider struct {
	apiKey   string
	proxyURL string
	client   *client.Client
	hosts    []string
}

func New() *AllDebridProvider {
	return &AllDebridProvider{}
}

func (p *AllDebridProvider) Name() string        { return "alldebrid" }
func (p *AllDebridProvider) DisplayName() string { return "AllDebrid" }
func (p *AllDebridProvider) Type() model.ProviderType {
	return model.ProviderTypeDebrid
}

func (p *AllDebridProvider) ConfigSchema() []provider.ConfigField {
	return []provider.ConfigField{
		{
			Key:         "api_key",
			Label:       "API Key",
			Type:        "password",
			Required:    true,
			Description: "Get your API key from alldebrid.com/apikeys",
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

func (p *AllDebridProvider) Configure(ctx context.Context, config map[string]string) error {
	p.apiKey = config["api_key"]
	p.proxyURL = config["proxy_url"]

	opts := []client.Option{
		client.WithTimeout(10 * time.Second),
	}

	if p.proxyURL != "" {
		opts = append(opts, client.WithProxy(p.proxyURL))
	}

	p.client = client.New(ctx, baseURL, opts...)

	if p.apiKey != "" {
		// Try to fetch hosts to verify key
		// Use a timeout context
		ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		hosts, err := p.fetchHosts(ctx)
		if err == nil {
			p.hosts = hosts
		}
	}
	return nil
}

func (p *AllDebridProvider) IsConfigured() bool {
	return p.apiKey != ""
}

func (p *AllDebridProvider) Supports(rawURL string) bool {
	if strings.HasPrefix(rawURL, "magnet:") {
		return true
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := strings.ToLower(u.Host)
	for _, h := range p.hosts {
		if strings.Contains(host, h) {
			return true
		}
	}

	return false
}

func (p *AllDebridProvider) Priority() int {
	return 100 // High priority
}

func (p *AllDebridProvider) Resolve(ctx context.Context, rawURL string, headers map[string]string) (*provider.ResolveResult, error) {
	// If it's already an AllDebrid direct link, we might not need to unlock it.
	// But we still want filename and size. Unlock is the best way to get metadata.

	params := url.Values{}
	params.Set("agent", agent)
	params.Set("apikey", p.apiKey)
	params.Set("link", rawURL)

	opts := rest.Opts{
		Method:     "GET",
		Path:       "/link/unlock",
		Parameters: params,
	}

	var result LinkUnlockResponse
	_, err := p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("alldebrid error: %s", result.Error.Message)
	}

	return &provider.ResolveResult{
		URL:  result.Data.Link,
		Name: result.Data.Filename,
		Size: result.Data.Filesize,
	}, nil
}

func (p *AllDebridProvider) Test(ctx context.Context) (*model.AccountInfo, error) {
	params := url.Values{}
	params.Set("agent", agent)
	params.Set("apikey", p.apiKey)

	opts := rest.Opts{
		Method:     "GET",
		Path:       "/user",
		Parameters: params,
	}

	var result UserResponse
	_, err := p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("failed to get account info")
	}

	expiresAt := time.Unix(result.Data.User.PremiumUntil, 0)
	return &model.AccountInfo{
		Username:  result.Data.User.Username,
		IsPremium: result.Data.User.IsPremium,
		ExpiresAt: &expiresAt,
	}, nil
}

func (p *AllDebridProvider) fetchHosts(ctx context.Context) ([]string, error) {
	params := url.Values{}
	params.Set("agent", agent)

	opts := rest.Opts{
		Method:     "GET",
		Path:       "/hosts",
		Parameters: params,
	}

	var result HostsResponse
	_, err := p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return nil, err
	}

	var hosts []string
	for _, h := range result.Data.Hosts {
		hosts = append(hosts, h.Domain)
	}

	return hosts, nil
}
