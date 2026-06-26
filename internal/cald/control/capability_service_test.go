package control

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/runtime"
)

func TestCapabilityServiceListsAndGetsCapabilities(t *testing.T) {
	svc := newTestService(t)
	capability := testCapabilityRecord(t, "provider_test")
	if err := svc.store.PutCapability(capability); err != nil {
		t.Fatalf("PutCapability() error = %v", err)
	}

	list, err := svc.ListCapabilities(runtime.ListOptions{})
	if err != nil {
		t.Fatalf("ListCapabilities() error = %v", err)
	}
	if list.Count != 1 || list.Capabilities[0].ID != capability.ID {
		t.Fatalf("ListCapabilities() = %#v, want one capability", list)
	}

	got, ok, err := svc.GetCapability(capability.ID)
	if err != nil {
		t.Fatalf("GetCapability() error = %v", err)
	}
	if !ok || got.ID != capability.ID {
		t.Fatalf("GetCapability() = %#v ok %v, want capability", got, ok)
	}
}
