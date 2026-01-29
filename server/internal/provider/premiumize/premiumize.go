package premiumize

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

const baseURL = "https://www.premiumize.me/api"

type PremiumizeProvider struct {
	apiKey   string
	proxyURL string
	client   *client.Client
	regexes  []*regexp.Regexp
}

func New() *PremiumizeProvider {
	return &PremiumizeProvider{}
}

func (p *PremiumizeProvider) Name() string        { return "premiumize" }
func (p *PremiumizeProvider) DisplayName() string { return "Premiumize" }
func (p *PremiumizeProvider) Type() model.ProviderType {
	return model.ProviderTypeDebrid
}

func (p *PremiumizeProvider) ConfigSchema() []provider.ConfigField {
	return []provider.ConfigField{
		{
			Key:         "api_key",
			Label:       "API Key",
			Type:        "password",
			Required:    true,
			Description: "Get your API Key from premiumize.me/account",
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

func (p *PremiumizeProvider) Configure(ctx context.Context, config map[string]string) error {
	p.apiKey = config["api_key"]
	p.proxyURL = config["proxy_url"]

	opts := []client.Option{
		client.WithTimeout(10 * time.Second),
	}

	if p.proxyURL != "" {
		opts = append(opts, client.WithProxy(p.proxyURL))
	}

	p.client = client.New(ctx, baseURL, opts...)

	// Fetch supported hosts/regexes
	tCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := p.fetchRegexes(tCtx); err != nil {
		fmt.Printf("Premiumize: failed to fetch supported hosts (continuing with fallbacks): %v\n", err)
	}

	return nil
}

func (p *PremiumizeProvider) IsConfigured() bool {
	return p.apiKey != ""
}

func (p *PremiumizeProvider) Supports(rawURL string) bool {
	if strings.HasPrefix(rawURL, "magnet:") {
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

	return strings.Contains(rawURL, "premiumize.me")
}

func (p *PremiumizeProvider) fetchRegexes(ctx context.Context) error {
	opts := rest.Opts{
		Method: "GET",
		Path:   "/services/list",
		Parameters: url.Values{
			"apikey": {p.apiKey},
		},
	}

	var result struct {
		Status        string   `json:"status"`
		DirectDL      []string `json:"directdl"`
		RegexPatterns []string `json:"regexpatterns"`
	}

	_, err := p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return err
	}

	var compiled []*regexp.Regexp
	for _, s := range result.RegexPatterns {
		re, err := regexp.Compile(s)
		if err == nil {
			compiled = append(compiled, re)
		}
	}

	if len(compiled) == 0 && len(result.DirectDL) > 0 {
		for _, domain := range result.DirectDL {
			pattern := `https?://(www\.)?` + regexp.QuoteMeta(domain)
			re, err := regexp.Compile(pattern)
			if err == nil {
				compiled = append(compiled, re)
			}
		}
	}

	p.regexes = compiled
	return nil
}

func (p *PremiumizeProvider) Priority() int {
	return 80
}

func (p *PremiumizeProvider) Resolve(ctx context.Context, rawURL string, headers map[string]string) (*provider.ResolveResult, error) {
	form := url.Values{}
	form.Set("src", rawURL)
	form.Set("apikey", p.apiKey)

	opts := rest.Opts{
		Method:      "POST",
		Path:        "/transfer/directdl",
		ContentType: "application/x-www-form-urlencoded",
		Body:        strings.NewReader(form.Encode()),
	}

	var result struct {
		Status   string `json:"status"`
		Message  string `json:"message"`
		Location string `json:"location"`
		Filename string `json:"filename"`
		Filesize int64  `json:"filesize"`
	}

	_, err := p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("premiumize error: %s", result.Message)
	}

	return &provider.ResolveResult{
		URL:  result.Location,
		Name: result.Filename,
		Size: result.Filesize,
	}, nil
}

func (p *PremiumizeProvider) Test(ctx context.Context) (*model.AccountInfo, error) {
	opts := rest.Opts{
		Method: "GET",
		Path:   "/account/info",
		Parameters: url.Values{
			"apikey": {p.apiKey},
		},
	}

	var result struct {
		Status       string  `json:"status"`
		Message      string  `json:"message"`
		PremiumUntil float64 `json:"premium_until"`
		CustomerId   string  `json:"customer_id"`
		LimitUsed    float64 `json:"limit_used"`
		LimitTotal   float64 `json:"limit_total"`
	}

	_, err := p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("premiumize error: %s", result.Message)
	}

	var expiresAt *time.Time
	if result.PremiumUntil > 0 {
		t := time.Unix(int64(result.PremiumUntil), 0)
		expiresAt = &t
	}

	return &model.AccountInfo{
		Username:  result.CustomerId,
		IsPremium: expiresAt != nil && expiresAt.After(time.Now()),
		ExpiresAt: expiresAt,
	}, nil
}

func (p *PremiumizeProvider) GetHosts(ctx context.Context) ([]string, error) {
	return []string{}, nil
}
