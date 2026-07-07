package store

import "github.com/spacehz-lab/cal/internal/model"

// ListProviders reads all stored provider records.
func (store *Store) ListProviders() ([]model.Provider, error) {
	return listJSONRecords(store.root, providersDir, model.ValidateProvider, func(a, b model.Provider) bool {
		return a.ID < b.ID
	})
}

// GetProvider reads one provider by id.
func (store *Store) GetProvider(id string) (model.Provider, bool, error) {
	return getJSONRecord(store, providersDir, id, model.ValidateProvider)
}

// SaveProvider writes one provider record.
func (store *Store) SaveProvider(provider *model.Provider) error {
	if provider == nil {
		return errNilRecord("provider")
	}
	return saveJSONRecord(store, providersDir, provider.ID, *provider, model.ValidateProvider)
}
