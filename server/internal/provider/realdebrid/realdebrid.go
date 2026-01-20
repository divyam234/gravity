package realdebrid

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

const baseURL = "https://api.real-debrid.com/rest/1.0"

type RealDebridProvider struct {
	apiKey     string
	httpClient *http.Client
}

func New() *RealDebridProvider {
	return &RealDebridProvider{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *RealDebridProvider) Name() string        { return "realdebrid" }
func (p *RealDebridProvider) DisplayName() string { return "Real-Debrid" }
func (p *RealDebridProvider) Type() model.ProviderType {
	return model.ProviderTypeDebrid
}

func (p *RealDebridProvider) ConfigSchema() []provider.ConfigField {
	return []provider.ConfigField{
		{
			Key:         "api_key",
			Label:       "API Key",
			Type:        "password",
			Required:    true,
			Description: "Get your API token from real-debrid.com/apitoken",
		},
	}
}

func (p *RealDebridProvider) Configure(config map[string]string) error {
	p.apiKey = config["api_key"]
	return nil
}

func (p *RealDebridProvider) IsConfigured() bool {
	return p.apiKey != ""
}

func (p *RealDebridProvider) Supports(rawURL string) bool {
	// Real-Debrid support is broad, but for now we'll just check common patterns or magnets
	return strings.HasPrefix(rawURL, "magnet:") || strings.Contains(rawURL, "1fichier.com") || strings.Contains(rawURL, "rapidgator.net")
}

func (p *RealDebridProvider) Priority() int {
	return 90 // Slightly lower than AllDebrid by default
}

func (p *RealDebridProvider) Resolve(ctx context.Context, rawURL string) (*provider.ResolveResult, error) {
	// 1. Unrestrict link
	form := url.Values{}
	form.Set("link", rawURL)

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/unrestrict/link", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result UnrestrictLinkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != "" {
		return nil, fmt.Errorf("realdebrid error: %s", result.Error)
	}

	return &provider.ResolveResult{
		URL:      result.Link,
		Filename: result.Filename,
		Size:     result.Filesize,
	}, nil
}

func (p *RealDebridProvider) Test(ctx context.Context) (*model.AccountInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if result.Expiration != "" {
		t, err := time.Parse(time.RFC3339, result.Expiration)
		if err == nil {
			expiresAt = &t
		}
	}

	return &model.AccountInfo{
		Username:  result.Username,
		IsPremium: result.Type == "premium",
		ExpiresAt: expiresAt,
	}, nil
}
