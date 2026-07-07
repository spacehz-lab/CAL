package cli

import (
	"net/http"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
)

func TestCapabilitiesListPassesFilters(t *testing.T) {
	cmd, stdout, _ := newTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/capabilities" {
			t.Fatalf("request = %s %s, want GET /v1/capabilities", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("capability_id"); got != "document.convert" {
			t.Fatalf("capability_id = %q, want document.convert", got)
		}
		if got := r.URL.Query().Get("provider_id"); got != "provider_cli" {
			t.Fatalf("provider_id = %q, want provider_cli", got)
		}
		writeResponse(t, w, contract.CapabilityListResponse{Count: 1})
	})

	if err := execute(t, cmd, "capabilities", "list", "--capability-id", "document.convert", "--provider-id", "provider_cli", "--json"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	response := decodeOutput[contract.CapabilityListResponse](t, stdout)
	if response.Count != 1 {
		t.Fatalf("response = %#v, want count 1", response)
	}
}
