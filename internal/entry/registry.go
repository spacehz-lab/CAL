package entry

import (
	"context"
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

// ProviderStore reads and writes durable provider records.
type ProviderStore interface {
	ListProviders() ([]model.Provider, error)
	GetProvider(id string) (model.Provider, bool, error)
	SaveProvider(provider *model.Provider) error
}

// Registry owns explicit provider registration and loading.
type Registry struct {
	store ProviderStore
}

// NewRegistry creates a provider entry registry.
func NewRegistry(store ProviderStore) *Registry {
	return &Registry{store: store}
}

// Register resolves and stores one explicit provider path.
func (registry *Registry) Register(ctx context.Context, req *RegisterRequest) (*RegisterResult, error) {
	ctx = normalizeContext(ctx)
	if err := registry.validate(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, newError(CodeInvalidProviderPath, "provider_path is required")
	}

	provider, err := resolveProviderPath(req.ProviderPath)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	providers, err := registry.store.ListProviders()
	if err != nil {
		return nil, wrapError(CodeEntryStoreFailed, "list providers", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	result := &RegisterResult{Provider: provider, Created: true}
	for _, existing := range providers {
		if existing.ID == provider.ID {
			result.Created = false
			result.Updated = true
			break
		}
	}

	if err := registry.store.SaveProvider(&provider); err != nil {
		return nil, wrapError(CodeEntryStoreFailed, "save provider", err)
	}
	return result, nil
}

// Load returns one stored provider by id.
func (registry *Registry) Load(ctx context.Context, req *LoadRequest) (*LoadResult, error) {
	ctx = normalizeContext(ctx)
	if err := registry.validate(ctx); err != nil {
		return nil, err
	}
	if req == nil || strings.TrimSpace(req.ProviderID) == "" {
		return nil, newError(CodeProviderNotFound, "provider_id is required")
	}

	providerID := strings.TrimSpace(req.ProviderID)
	provider, ok, err := registry.store.GetProvider(providerID)
	if err != nil {
		return nil, wrapError(CodeEntryStoreFailed, "get provider", err)
	}
	if !ok {
		return nil, newError(CodeProviderNotFound, fmt.Sprintf("provider %q was not found", providerID))
	}
	return &LoadResult{Provider: provider}, nil
}

// List returns stored providers.
func (registry *Registry) List(ctx context.Context) ([]model.Provider, error) {
	ctx = normalizeContext(ctx)
	if err := registry.validate(ctx); err != nil {
		return nil, err
	}
	providers, err := registry.store.ListProviders()
	if err != nil {
		return nil, wrapError(CodeEntryStoreFailed, "list providers", err)
	}
	return providers, nil
}

// GetByPath returns a stored provider by normalized provider path.
func (registry *Registry) GetByPath(ctx context.Context, providerPath string) (model.Provider, bool, error) {
	ctx = normalizeContext(ctx)
	if err := registry.validate(ctx); err != nil {
		return model.Provider{}, false, err
	}
	normalizedPath, err := normalizeProviderPath(providerPath)
	if err != nil {
		return model.Provider{}, false, err
	}

	providers, err := registry.store.ListProviders()
	if err != nil {
		return model.Provider{}, false, wrapError(CodeEntryStoreFailed, "list providers", err)
	}
	for _, provider := range providers {
		if provider.Path == normalizedPath {
			return provider, true, nil
		}
	}
	return model.Provider{}, false, nil
}

func (registry *Registry) validate(ctx context.Context) error {
	if registry == nil || registry.store == nil {
		return newError(CodeEntryStoreFailed, "provider store is required")
	}
	return ctx.Err()
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
