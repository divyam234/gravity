package provider

import (
	"context"
	"fmt"
	"sort"
)

type Resolver struct {
	registry *Registry
}

func NewResolver(registry *Registry) *Resolver {
	return &Resolver{registry: registry}
}

func (r *Resolver) Resolve(ctx context.Context, url string, headers map[string]string) (*ResolveResult, string, error) {
	providers := r.registry.List()

	// Sort by priority descending
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Priority() > providers[j].Priority()
	})

	for _, p := range providers {
		if p.IsConfigured() && p.Supports(url) {
			res, err := p.Resolve(ctx, url, headers)
			if err == nil && res != nil {
				return res, p.Name(), nil
			}
		}
	}

	return nil, "", fmt.Errorf("no provider could resolve the URL")
}

func (r *Resolver) IsSupported(url string) bool {
	providers := r.registry.List()
	for _, p := range providers {
		if p.IsConfigured() && p.Supports(url) {
			return true
		}
	}
	return false
}
