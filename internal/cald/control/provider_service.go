package control

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/discovery"
)

// AddProvider registers one explicit provider entry path.
func (svc Service) AddProvider(ctx context.Context, providerPath string) (core.Provider, error) {
	provider, err := providerFromPath(ctx, providerPath)
	if err != nil {
		return core.Provider{}, err
	}
	_, _, err = svc.saveProviders([]core.Provider{provider})
	return provider, err
}

// ListProviders returns stored Provider records.
func (svc Service) ListProviders() ([]core.Provider, error) {
	return svc.store.ListProviders()
}

// GetProvider returns one stored Provider record.
func (svc Service) GetProvider(id string) (core.Provider, bool, error) {
	return svc.store.GetProvider(id)
}

// GetProviderByPath resolves one explicit provider entry path and returns its stored Provider record.
func (svc Service) GetProviderByPath(providerPath string) (core.Provider, bool, error) {
	providerPath = strings.TrimSpace(providerPath)
	if providerPath == "" {
		return core.Provider{}, false, NewAPIError("invalid_provider_path", "provider_path is required")
	}
	cleanPath, err := filepath.Abs(filepath.Clean(os.ExpandEnv(providerPath)))
	if err != nil {
		return core.Provider{}, false, err
	}
	providers, err := svc.store.ListProviders()
	if err != nil {
		return core.Provider{}, false, err
	}
	for _, provider := range providers {
		if provider.Path == cleanPath {
			return provider, true, nil
		}
	}
	return core.Provider{}, false, nil
}

func providerFromPath(ctx context.Context, providerPath string) (core.Provider, error) {
	providerPath = strings.TrimSpace(providerPath)
	if providerPath == "" {
		return core.Provider{}, NewAPIError("invalid_provider_path", "provider_path is required")
	}
	providers, err := discovery.ScanEntries(ctx, discovery.EntryOptions{Entries: []string{providerPath}})
	if err != nil {
		return core.Provider{}, err
	}
	if len(providers) == 0 {
		return core.Provider{}, NewAPIError("target_provider_not_found", fmt.Sprintf("provider path %q did not resolve to a CLI executable or app bundle provider", providerPath))
	}
	if len(providers) > 1 {
		return core.Provider{}, NewAPIError("ambiguous_target_provider", fmt.Sprintf("provider path %q resolved to %d providers", providerPath, len(providers)))
	}
	return providers[0], nil
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
