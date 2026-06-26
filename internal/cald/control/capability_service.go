package control

import (
	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/runtime"
)

// ListCapabilities returns the service capability catalog.
func (svc Service) ListCapabilities(opts runtime.ListOptions) (runtime.CapabilityList, error) {
	capabilities, err := svc.store.ListCapabilities()
	if err != nil {
		return runtime.CapabilityList{}, err
	}
	return runtime.NewCatalog().List(capabilities, opts), nil
}

// GetCapability returns one stored Capability record.
func (svc Service) GetCapability(id string) (core.Capability, bool, error) {
	return svc.store.GetCapability(id)
}
