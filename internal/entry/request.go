package entry

import "github.com/spacehz-lab/cal/internal/model"

// RegisterRequest registers one explicit provider path.
type RegisterRequest struct {
	ProviderPath string
}

// RegisterResult reports the stored provider and write action.
type RegisterResult struct {
	Provider model.Provider
	Created  bool
	Updated  bool
}

// LoadRequest loads one provider by id.
type LoadRequest struct {
	ProviderID string
}

// LoadResult reports the loaded provider.
type LoadResult struct {
	Provider model.Provider
}
