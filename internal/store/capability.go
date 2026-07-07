package store

import "github.com/spacehz-lab/cal/internal/model"

// ListCapabilities reads all stored capability records.
func (store *Store) ListCapabilities() ([]model.Capability, error) {
	return listJSONRecords(store.root, capabilitiesDir, model.ValidateCapability, func(a, b model.Capability) bool {
		return a.ID < b.ID
	})
}

// GetCapability reads one capability by id.
func (store *Store) GetCapability(id string) (model.Capability, bool, error) {
	return getJSONRecord(store, capabilitiesDir, id, model.ValidateCapability)
}

// SaveCapability writes one capability record.
func (store *Store) SaveCapability(capability *model.Capability) error {
	if capability == nil {
		return errNilRecord("capability")
	}
	return saveJSONRecord(store, capabilitiesDir, capability.ID, *capability, model.ValidateCapability)
}
