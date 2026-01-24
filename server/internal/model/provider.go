package model

import "time"

type ProviderType string

const (
	ProviderTypeDirect   ProviderType = "direct"
	ProviderTypeDebrid   ProviderType = "debrid"
	ProviderTypeFileHost ProviderType = "filehost"
)

type Provider struct {
	Name          string            `json:"name" gorm:"primaryKey"`
	DisplayName   string            `json:"displayName"`
	Type          ProviderType      `json:"type"`
	Enabled       bool              `json:"enabled"`
	Priority      int               `json:"priority"`
	Config        map[string]string `json:"config,omitempty" gorm:"serializer:json"`
	CachedHosts   []string          `json:"cachedHosts,omitempty" gorm:"serializer:json"`
	CachedAccount *AccountInfo      `json:"cachedAccount,omitempty" gorm:"serializer:json"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

type AccountInfo struct {
	Username  string     `json:"username,omitempty"`
	IsPremium bool       `json:"isPremium"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}
