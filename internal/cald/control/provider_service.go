package control

import (
	"context"
	"strings"

	"github.com/spacehz-lab/cal/internal/config"
	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/discovery"
)

// ProviderFindRequest filters provider entry scans.
type ProviderFindRequest struct {
	Kind string `json:"kind,omitempty"`
}

// ProviderFindResponse reports provider records created or updated by a scan.
type ProviderFindResponse struct {
	ProvidersCreated int             `json:"providers_created"`
	ProvidersUpdated int             `json:"providers_updated"`
	Providers        []core.Provider `json:"providers"`
}

// ListSources returns configured provider sources.
func (svc Service) ListSources() ([]config.ProviderSource, error) {
	cfg, err := svc.cfg.Ensure()
	if err != nil {
		return nil, err
	}
	return cfg.ProviderSources, nil
}

// AddPathSource adds a path provider source.
func (svc Service) AddPathSource(path string) (config.Config, bool, error) {
	return svc.cfg.AddProviderPath(path)
}

// RemovePathSource removes a path provider source.
func (svc Service) RemovePathSource(path string) (config.Config, bool, error) {
	return svc.cfg.RemoveProviderPath(path)
}

// FindProviders scans configured sources and writes Provider records.
func (svc Service) FindProviders(ctx context.Context, req ProviderFindRequest) (ProviderFindResponse, error) {
	if err := validateProviderKindFilter(req.Kind); err != nil {
		return ProviderFindResponse{}, err
	}
	cfg, err := svc.cfg.Ensure()
	if err != nil {
		return ProviderFindResponse{}, err
	}
	providers, err := discovery.ScanEntries(ctx, discovery.EntryOptions{Paths: cfg.PathSources()})
	if err != nil {
		return ProviderFindResponse{}, err
	}
	providers = filterProviders(providers, req.Kind)
	created, updated, err := svc.saveProviders(providers)
	if err != nil {
		return ProviderFindResponse{}, err
	}
	return ProviderFindResponse{
		ProvidersCreated: created,
		ProvidersUpdated: updated,
		Providers:        providers,
	}, nil
}

// ListProviders returns stored Provider records.
func (svc Service) ListProviders() ([]core.Provider, error) {
	return svc.store.ListProviders()
}

// GetProvider returns one stored Provider record.
func (svc Service) GetProvider(id string) (core.Provider, bool, error) {
	return svc.store.GetProvider(id)
}

func (svc Service) saveProviders(providers []core.Provider) (int, int, error) {
	existingProviders, err := svc.store.ListProviders()
	if err != nil {
		return 0, 0, err
	}
	existing := make(map[string]struct{}, len(existingProviders))
	for _, provider := range existingProviders {
		existing[provider.ID] = struct{}{}
	}
	created := 0
	updated := 0
	for _, provider := range providers {
		if _, ok := existing[provider.ID]; ok {
			updated++
		} else {
			created++
		}
		if err := svc.store.PutProvider(provider); err != nil {
			return 0, 0, err
		}
	}
	return created, updated, nil
}

func filterProviders(providers []core.Provider, kind string) []core.Provider {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return providers
	}
	filtered := make([]core.Provider, 0, len(providers))
	for _, provider := range providers {
		if string(provider.Kind) == kind {
			filtered = append(filtered, provider)
		}
	}
	return filtered
}

func validateProviderKindFilter(kind string) error {
	switch strings.TrimSpace(kind) {
	case "", string(core.ProviderKindCLI), string(core.ProviderKindApp):
		return nil
	default:
		return NewAPIError("unsupported_provider_kind", "provider kind must be cli or app")
	}
}
