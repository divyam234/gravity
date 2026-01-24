package alldebrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gravity/internal/model"
	"gravity/internal/provider"
)

const (
	baseURL = "https://api.alldebrid.com/v4.1"
	agent   = "gravity"
)

type AllDebridProvider struct {
	apiKey     string
	httpClient *http.Client
	hosts      []string
}

func New() *AllDebridProvider {
	return &AllDebridProvider{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
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
	}
}

func (p *AllDebridProvider) Configure(config map[string]string) error {
	p.apiKey = config["api_key"]
	if p.apiKey != "" {
		// Try to fetch hosts to verify key
		hosts, err := p.fetchHosts(context.Background())
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

func (p *AllDebridProvider) Resolve(ctx context.Context, rawURL string) (*provider.ResolveResult, error) {
	endpoint := fmt.Sprintf("%s/link/unlock?agent=%s&apikey=%s&link=%s",
		baseURL, agent, p.apiKey, url.QueryEscape(rawURL))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result LinkUnlockResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("alldebrid error: %s", result.Error.Message)
	}

	return &provider.ResolveResult{
		URL:      result.Data.Link,
		Filename: result.Data.Filename,
		Size:     result.Data.Filesize,
	}, nil
}

func (p *AllDebridProvider) Test(ctx context.Context) (*model.AccountInfo, error) {
	endpoint := fmt.Sprintf("%s/user?agent=%s&apikey=%s", baseURL, agent, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
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
	endpoint := fmt.Sprintf("%s/hosts?agent=%s", baseURL, agent)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result HostsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var hosts []string
	for _, h := range result.Data.Hosts {
		hosts = append(hosts, h.Domain)
	}

	return hosts, nil
}
