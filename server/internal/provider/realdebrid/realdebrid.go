package realdebrid

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rclone/rclone/lib/rest"
	"gravity/internal/client"
	"gravity/internal/model"
	"gravity/internal/provider"
)

const baseURL = "https://api.real-debrid.com/rest/1.0"

type RealDebridProvider struct {
	apiKey   string
	proxyURL string
	client   *client.Client
	regexes  []*regexp.Regexp
}

func New() *RealDebridProvider {
	return &RealDebridProvider{}
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
		{
			Key:         "proxy_url",
			Label:       "Proxy URL",
			Type:        "text",
			Required:    false,
			Description: "Optional proxy (http://user:pass@host:port)",
		},
	}
}

func (p *RealDebridProvider) Configure(ctx context.Context, config map[string]string) error {
	p.apiKey = config["api_key"]
	p.proxyURL = config["proxy_url"]

	opts := []client.Option{
		client.WithTimeout(10 * time.Second),
	}

	if p.proxyURL != "" {
		opts = append(opts, client.WithProxy(p.proxyURL))
	}

	// Re-initialize client with options
	p.client = client.New(ctx, baseURL, opts...)

	// Fetch supported hosts/regexes synchronously with timeout
	tCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := p.fetchRegexes(tCtx); err != nil {
		fmt.Printf("Real-Debrid: failed to fetch supported hosts: %v\n", err)
	}

	return nil
}

func (p *RealDebridProvider) IsConfigured() bool {
	return p.apiKey != ""
}

func (p *RealDebridProvider) Supports(rawURL string) bool {
	if strings.HasPrefix(rawURL, "magnet:") {
		return true
	}

	if strings.Contains(rawURL, "real-debrid.com") {
		return true
	}

	if len(p.regexes) > 0 {
		for _, re := range p.regexes {
			if re.MatchString(rawURL) {
				return true
			}
		}
		return false
	}

	// Fallback if regexes failed to load
	return strings.Contains(rawURL, "1fichier.com") || strings.Contains(rawURL, "rapidgator.net") || strings.Contains(rawURL, "mega.nz")
}

func (p *RealDebridProvider) fetchRegexes(ctx context.Context) error {
	opts := rest.Opts{
		Method: "GET",
		Path:   "/hosts/regex",
	}

	var regexStrings []string
	_, err := p.client.CallJSON(ctx, &opts, nil, &regexStrings)
	if err != nil {
		return err
	}

	var compiled []*regexp.Regexp
	for _, s := range regexStrings {
		re, err := regexp.Compile(s)
		if err == nil {
			compiled = append(compiled, re)
		}
	}

	p.regexes = compiled
	return nil
}

func (p *RealDebridProvider) Priority() int {
	return 90 // Slightly lower than AllDebrid by default
}

func (p *RealDebridProvider) Resolve(ctx context.Context, rawURL string, headers map[string]string) (*provider.ResolveResult, error) {
	// 1. Unrestrict link
	form := url.Values{}
	form.Set("link", rawURL)

	opts := rest.Opts{
		Method:      "POST",
		Path:        "/unrestrict/link",
		ContentType: "application/x-www-form-urlencoded",
		Body:        strings.NewReader(form.Encode()),
		ExtraHeaders: map[string]string{
			"Authorization": "Bearer " + p.apiKey,
		},
	}

	var result UnrestrictLinkResponse
	_, err := p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return nil, err
	}

	if result.Error != "" {
		return nil, fmt.Errorf("realdebrid error: %s", result.Error)
	}

	res := &provider.ResolveResult{
		URL:  result.Link,
		Name: result.Filename,
		Size: result.Filesize,
	}

	// Fetch ModTime
	if meta, err := provider.FetchMetadata(ctx, p.client, result.Link); err == nil {
		res.ModTime = meta.ModTime
	}

	return res, nil
}

func (p *RealDebridProvider) Test(ctx context.Context) (*model.AccountInfo, error) {
	opts := rest.Opts{
		Method: "GET",
		Path:   "/user",
		ExtraHeaders: map[string]string{
			"Authorization": "Bearer " + p.apiKey,
		},
	}

	var result UserResponse
	_, err := p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
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
