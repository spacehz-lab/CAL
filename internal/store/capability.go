package store

import (
	"github.com/spacehz-lab/cal/internal/core"
)

// ListCapabilities reads all stored capability records.
func (s *Store) ListCapabilities() ([]core.Capability, error) {
	return listJSONRecords(s.home, capabilitiesDir, "capabilities", "capability", core.ValidateCapability, func(a, b core.Capability) bool {
		return a.ID < b.ID
	})
}

// GetCapability reads one capability by id.
func (s *Store) GetCapability(id string) (core.Capability, bool, error) {
	return getJSONRecord(s, capabilitiesDir, id, core.ValidateCapability)
}

// PutCapability writes one capability record.
func (s *Store) PutCapability(capability core.Capability) error {
	return putJSONRecord(s, capabilitiesDir, capability.ID, capability, core.ValidateCapability)
}
