package model

import "time"

type ProviderType string

const (
	ProviderTypeDirect   ProviderType = "direct"
	ProviderTypeDebrid   ProviderType = "debrid"
	ProviderTypeFileHost ProviderType = "filehost"
)

type Provider struct {
	Name          string            `json:"name"`
	DisplayName   string            `json:"displayName"`
	Type          ProviderType      `json:"type"`
	Enabled       bool              `json:"enabled"`
	Priority      int               `json:"priority"`
	Config        map[string]string `json:"config,omitempty"`
	CachedHosts   []string          `json:"cachedHosts,omitempty"`
	CachedAccount *AccountInfo      `json:"cachedAccount,omitempty"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

type AccountInfo struct {
	Username  string     `json:"username,omitempty"`
	IsPremium bool       `json:"isPremium"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}
