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
	URL           string                `json:"url"`
	Name          string                `json:"name"`
	Size          int64                 `json:"size"`
	Headers       map[string]string     `json:"headers"`
	Error         string                `json:"error,omitempty"`
	IsMagnet      bool                  `json:"isMagnet"`
	Hash          string                `json:"hash,omitempty"`
	Files         []*model.DownloadFile `json:"files,omitempty"`
	ExecutionMode model.ExecutionMode   `json:"executionMode"`
}

type Provider interface {
	// Metadata
	Name() string
	DisplayName() string
	Type() model.ProviderType

	// Configuration
	ConfigSchema() []ConfigField
	Configure(ctx context.Context, config map[string]string) error
	IsConfigured() bool

	// URL handling
	Supports(url string) bool
	Priority() int
	Resolve(ctx context.Context, url string, headers map[string]string) (*ResolveResult, error)

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

	// CheckMagnet checks if a magnet is available and returns file list
	// Returns nil, nil if magnet is not cached (for debrid providers)
	CheckMagnet(ctx context.Context, magnet string) (*model.MagnetInfo, error)

	// GetMagnetFiles returns file tree for a magnet (by ID for debrid, by hash for aria2)
	GetMagnetFiles(ctx context.Context, magnetID string) ([]*model.MagnetFile, error)

	// DeleteMagnet removes a magnet from user's account (debrid only)
	DeleteMagnet(ctx context.Context, magnetID string) error
}
