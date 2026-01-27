package service

import (
	"context"
	"fmt"

	"gravity/internal/model"
	"gravity/internal/provider"
	"gravity/internal/store"

	"go.uber.org/zap"
)

type ProviderService struct {
	repo     *store.ProviderRepo
	registry *provider.Registry
	resolver *provider.Resolver
	logger   *zap.Logger
}

func NewProviderService(repo *store.ProviderRepo, registry *provider.Registry, l *zap.Logger) *ProviderService {
	return &ProviderService{
		repo:     repo,
		registry: registry,
		resolver: provider.NewResolver(registry),
		logger:   l.With(zap.String("service", "provider")),
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
			impl.Configure(ctx, p.Config)
		}
	}

	return nil
}

func (s *ProviderService) List(ctx context.Context) ([]model.ProviderSummary, error) {
	// Combine implementation info with stored config/status
	results := []model.ProviderSummary{}

	for _, impl := range s.registry.List() {
		summary := model.ProviderSummary{
			Name:        impl.Name(),
			DisplayName: impl.DisplayName(),
			Type:        impl.Type(),
			Priority:    impl.Priority(),
			Configured:  impl.IsConfigured(),
		}

		stored, err := s.repo.Get(ctx, impl.Name())
		if err == nil {
			summary.Enabled = stored.Enabled
			summary.Account = stored.CachedAccount
		} else {
			summary.Enabled = false
		}

		results = append(results, summary)
	}

	return results, nil
}

func (s *ProviderService) Configure(ctx context.Context, name string, config map[string]string, enabled bool) error {
	s.logger.Info("configuring provider", zap.String("name", name), zap.Bool("enabled", enabled))
	impl := s.registry.Get(name)
	if impl == nil {
		return fmt.Errorf("provider not found")
	}

	if err := impl.Configure(ctx, config); err != nil {
		s.logger.Error("failed to configure provider", zap.String("name", name), zap.Error(err))
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

func (s *ProviderService) Resolve(ctx context.Context, url string, headers map[string]string) (*provider.ResolveResult, string, error) {
	return s.resolver.Resolve(ctx, url, headers)
}

func (s *ProviderService) GetConfigSchema(name string) ([]provider.ConfigField, error) {
	impl := s.registry.Get(name)
	if impl == nil {
		return nil, fmt.Errorf("provider not found")
	}
	return impl.ConfigSchema(), nil
}

func (s *ProviderService) Delete(ctx context.Context, name string) error {
	// Reset config and disable
	return s.Configure(ctx, name, map[string]string{}, false)
}

func (s *ProviderService) GetStatus(ctx context.Context, name string) (*model.AccountInfo, error) {
	impl := s.registry.Get(name)
	if impl == nil {
		return nil, fmt.Errorf("provider not found")
	}
	// Force a test to get fresh status
	return impl.Test(ctx)
}

func (s *ProviderService) GetHosts(ctx context.Context, name string) ([]string, error) {
	impl := s.registry.Get(name)
	if impl == nil {
		return nil, fmt.Errorf("provider not found")
	}

	if debrid, ok := impl.(provider.DebridProvider); ok {
		return debrid.GetHosts(ctx)
	}

	return []string{}, nil
}
