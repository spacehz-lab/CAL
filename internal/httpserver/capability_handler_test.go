package httpserver

import (
	"net/http"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
)

func TestCapabilityQueryReturnsListResponse(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodGet, routeCapabilities+"?capability_id=document.echo&provider_id=provider_test", "")

	assertStatus(t, rec, http.StatusOK)
	response := decodeResponse[contract.CapabilityListResponse](t, rec)
	if response.Count != 0 || len(response.Capabilities) != 0 {
		t.Fatalf("response = %#v, want empty filtered list", response)
	}
}
