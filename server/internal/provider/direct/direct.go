package direct

import (
	"context"
	"strings"
	"time"

	"gravity/internal/client"
	"gravity/internal/model"
	"gravity/internal/provider"
)

type DirectProvider struct {
}

func New() *DirectProvider {
	return &DirectProvider{}
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
	c := client.New(ctx, "", client.WithTimeout(30*time.Second))
	if strings.HasPrefix(rawURL, "magnet:") {
		return &provider.ResolveResult{
			URL:  rawURL,
			Name: "",
		}, nil
	}

	return provider.FetchMetadata(ctx, c, rawURL)
}

func (p *DirectProvider) Test(ctx context.Context) (*model.AccountInfo, error) {
	return &model.AccountInfo{
		Username:  "Guest",
		IsPremium: false,
	}, nil
}
