package direct

import (
	"context"
	"path/filepath"
	"strings"

	"gravity/internal/model"
	"gravity/internal/provider"
)

type DirectProvider struct{}

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

func (p *DirectProvider) Configure(config map[string]string) error {
	return nil
}

func (p *DirectProvider) IsConfigured() bool {
	return true
}

func (p *DirectProvider) Supports(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "magnet:")
}

func (p *DirectProvider) Priority() int {
	return 0 // Lowest priority
}

func (p *DirectProvider) Resolve(ctx context.Context, url string) (*provider.ResolveResult, error) {
	filename := filepath.Base(url)
	if strings.HasPrefix(url, "magnet:") {
		filename = "" // Will be determined by engine
	} else if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}

	return &provider.ResolveResult{
		URL:      url,
		Filename: filename,
	}, nil
}

func (p *DirectProvider) Test(ctx context.Context) (*model.AccountInfo, error) {
	return &model.AccountInfo{
		Username:  "Guest",
		IsPremium: false,
	}, nil
}
