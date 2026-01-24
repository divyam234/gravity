package torbox

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rclone/rclone/lib/rest"
	"gravity/internal/client"
	"gravity/internal/model"
	"gravity/internal/provider"
)

const baseURL = "https://api.torbox.app/v1"

type TorBoxProvider struct {
	apiKey   string
	proxyURL string
	client   *client.Client
}

func New() *TorBoxProvider {
	return &TorBoxProvider{}
}

func (p *TorBoxProvider) Name() string        { return "torbox" }
func (p *TorBoxProvider) DisplayName() string { return "TorBox" }
func (p *TorBoxProvider) Type() model.ProviderType {
	return model.ProviderTypeDebrid
}

func (p *TorBoxProvider) ConfigSchema() []provider.ConfigField {
	return []provider.ConfigField{
		{
			Key:         "api_key",
			Label:       "API Key",
			Type:        "password",
			Required:    true,
			Description: "Get your API Key from torbox.app/settings",
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

func (p *TorBoxProvider) Configure(ctx context.Context, config map[string]string) error {
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

func (p *TorBoxProvider) IsConfigured() bool {
	return p.apiKey != ""
}

func (p *TorBoxProvider) Supports(rawURL string) bool {
	return strings.HasPrefix(rawURL, "magnet:")
}

func (p *TorBoxProvider) Priority() int {
	return 70
}

func (p *TorBoxProvider) Resolve(ctx context.Context, rawURL string, headers map[string]string) (*provider.ResolveResult, error) {
	// TorBox is mainly for torrents. If this is a direct link (not magnet), it might not work unless they have a link unrestrictor.
	// Documentation says they have /usenet/requestdl and /web/requestdl (for debrid links).
	// For now, if it's not a magnet, we might return error or try /web/requestdl if we knew the format.
	// But Supports only checks for magnet: so this method might not be called for http links.
	return nil, fmt.Errorf("torbox only supports magnets for now")
}

func (p *TorBoxProvider) Test(ctx context.Context) (*model.AccountInfo, error) {
	opts := rest.Opts{
		Method: "GET",
		Path:   "/user/me",
		ExtraHeaders: map[string]string{
			"Authorization": "Bearer " + p.apiKey,
		},
	}

	var result struct {
		Success bool        `json:"success"`
		Error   interface{} `json:"error"`
		Data    struct {
			Email      string `json:"email"`
			Plan       int    `json:"plan"`       // 0=free, >0=premium?
			Expiration string `json:"expiration"` // Date string or null
		} `json:"data"`
	}

	_, err := p.client.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("torbox error: %v", result.Error)
	}

	// Parse Expiration if needed, format usually ISO?
	// Plan > 0 is usually Premium.
	return &model.AccountInfo{
		Username:  result.Data.Email,
		IsPremium: result.Data.Plan > 0,
	}, nil
}

func (p *TorBoxProvider) GetHosts(ctx context.Context) ([]string, error) {
	return []string{}, nil
}
