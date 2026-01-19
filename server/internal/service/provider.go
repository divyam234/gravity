package service

import (
	"context"
	"fmt"

	"gravity/internal/model"
	"gravity/internal/provider"
	"gravity/internal/store"
)

type ProviderService struct {
	repo     *store.ProviderRepo
	registry *provider.Registry
	resolver *provider.Resolver
}

func NewProviderService(repo *store.ProviderRepo, registry *provider.Registry) *ProviderService {
	return &ProviderService{
		repo:     repo,
		registry: registry,
		resolver: provider.NewResolver(registry),
	}
}

func (s *ProviderService) Init(ctx context.Context) error {
	// Load configurations from DB and apply to providers in registry
	stored, err := s.repo.List(ctx)
	if err != nil {
		return err
	}

	for _, p := range stored {
		impl := s.registry.Get(p.Name)
		if impl != nil {
			impl.Configure(p.Config)
		}
	}

	return nil
}

func (s *ProviderService) List(ctx context.Context) ([]map[string]interface{}, error) {
	// Combine implementation info with stored config/status
	results := []map[string]interface{}{}

	for _, impl := range s.registry.List() {
		res := map[string]interface{}{
			"name":        impl.Name(),
			"displayName": impl.DisplayName(),
			"type":        impl.Type(),
			"priority":    impl.Priority(),
			"configured":  impl.IsConfigured(),
		}

		stored, err := s.repo.Get(ctx, impl.Name())
		if err == nil {
			res["enabled"] = stored.Enabled
			res["account"] = stored.CachedAccount
		} else {
			res["enabled"] = false
		}

		results = append(results, res)
	}

	return results, nil
}

func (s *ProviderService) Configure(ctx context.Context, name string, config map[string]string, enabled bool) error {
	impl := s.registry.Get(name)
	if impl == nil {
		return fmt.Errorf("provider not found")
	}

	if err := impl.Configure(config); err != nil {
		return err
	}

	// Test connection and cache account info
	account, _ := impl.Test(ctx)

	p := &model.Provider{
		Name:          name,
		Enabled:       enabled,
		Priority:      impl.Priority(),
		Config:        config,
		CachedAccount: account,
	}

	return s.repo.Save(ctx, p)
}

func (s *ProviderService) Resolve(ctx context.Context, url string) (*provider.ResolveResult, string, error) {
	return s.resolver.Resolve(ctx, url)
}

func (s *ProviderService) GetConfigSchema(name string) ([]provider.ConfigField, error) {
	impl := s.registry.Get(name)
	if impl == nil {
		return nil, fmt.Errorf("provider not found")
	}
	return impl.ConfigSchema(), nil
}
