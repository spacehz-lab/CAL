package store

import (
	"github.com/spacehz-lab/cal/internal/core"
)

// ListProviders reads all stored provider records.
func (s *Store) ListProviders() ([]core.Provider, error) {
	return listJSONRecords(s.home, providersDir, "providers", "provider", core.ValidateProvider, func(a, b core.Provider) bool {
		return a.ID < b.ID
	})
}

// GetProvider reads one provider by id.
func (s *Store) GetProvider(id string) (core.Provider, bool, error) {
	return getJSONRecord(s, providersDir, id, core.ValidateProvider)
}

// PutProvider writes one provider record.
func (s *Store) PutProvider(provider core.Provider) error {
	return putJSONRecord(s, providersDir, provider.ID, provider, core.ValidateProvider)
}
