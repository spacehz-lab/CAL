package cli

import (
	"net/http"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestProvidersAddPostsProviderPath(t *testing.T) {
	cmd, stdout, _ := newTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/providers" {
			t.Fatalf("request = %s %s, want POST /v1/providers", r.Method, r.URL.Path)
		}
		req := decodeRequest[contract.AddProviderRequest](t, r)
		if req.ProviderPath != "/bin/echo" {
			t.Fatalf("provider path = %q, want /bin/echo", req.ProviderPath)
		}
		writeResponse(t, w, contract.ProviderListResponse{Providers: []model.Provider{{ID: "provider_cli", Path: "/bin/echo"}}})
	})

	if err := execute(t, cmd, "providers", "add", "--provider-path", "/bin/echo", "--json"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	response := decodeOutput[contract.ProviderListResponse](t, stdout)
	if len(response.Providers) != 1 || response.Providers[0].ID != "provider_cli" {
		t.Fatalf("response = %#v, want provider_cli", response)
	}
}

func TestProvidersListCallsClient(t *testing.T) {
	cmd, stdout, _ := newTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/providers" {
			t.Fatalf("request = %s %s, want GET /v1/providers", r.Method, r.URL.Path)
		}
		writeResponse(t, w, contract.ProviderListResponse{Providers: []model.Provider{{ID: "provider_cli"}}})
	})

	if err := execute(t, cmd, "providers", "list", "--json"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	response := decodeOutput[contract.ProviderListResponse](t, stdout)
	if len(response.Providers) != 1 {
		t.Fatalf("providers len = %d, want 1", len(response.Providers))
	}
}
