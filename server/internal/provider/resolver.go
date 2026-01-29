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

	var lastErr error
	for _, p := range providers {
		if p.IsConfigured() && p.Supports(url) {
			_, cb := r.registry.GetWithBreaker(p.Name())
			if cb != nil && !cb.Allow() {
				continue // Circuit open, skip this provider
			}

			// Rate Limit
			limiter := r.registry.GetLimiter(p.Name())
			if limiter != nil {
				if err := limiter.Wait(ctx); err != nil {
					// Context cancelled or timeout
					if lastErr == nil {
						lastErr = err
					}
					continue
				}
			}

			res, err := p.Resolve(ctx, url, headers)
			if err == nil && res != nil {
				if cb != nil {
					cb.RecordSuccess()
				}
				return res, p.Name(), nil
			}

			if cb != nil {
				cb.RecordFailure()
			}
			lastErr = err
		}
	}

	if lastErr != nil {
		return nil, "", lastErr
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
