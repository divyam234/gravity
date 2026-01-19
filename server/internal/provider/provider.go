package provider

import (
	"context"

	"gravity/internal/model"
)

type ConfigField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"` // text, password, select
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type ResolveResult struct {
	URL      string            `json:"url"`
	Filename string            `json:"filename"`
	Size     int64             `json:"size"`
	Headers  map[string]string `json:"headers"`
	Error    string            `json:"error,omitempty"`
}

type Provider interface {
	// Metadata
	Name() string
	DisplayName() string
	Type() model.ProviderType

	// Configuration
	ConfigSchema() []ConfigField
	Configure(config map[string]string) error
	IsConfigured() bool

	// URL handling
	Supports(url string) bool
	Priority() int
	Resolve(ctx context.Context, url string) (*ResolveResult, error)

	// Health
	Test(ctx context.Context) (*model.AccountInfo, error)
}

type DebridProvider interface {
	Provider
	GetHosts(ctx context.Context) ([]string, error)
}

// MagnetProvider is implemented by providers that support magnet links with file selection
type MagnetProvider interface {
	Provider

	// CheckMagnet checks if a magnet is available/cached and returns file list
	// Returns nil, nil if magnet is not cached (for debrid providers)
	CheckMagnet(ctx context.Context, magnet string) (*model.MagnetInfo, error)

	// DeleteMagnet removes a magnet from user's account (debrid only)
	DeleteMagnet(ctx context.Context, magnetID string) error
}
