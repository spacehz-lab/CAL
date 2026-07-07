package httpserver

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
)

func TestProviderListAndAdd(t *testing.T) {
	server := newTestServer(t, DaemonControl{})
	path := writeProviderScript(t)

	add := serve(t, server, http.MethodPost, routeProviders, fmt.Sprintf(`{"provider_path":%q}`, path))
	assertStatus(t, add, http.StatusOK)
	added := decodeResponse[contract.ProviderListResponse](t, add)
	if len(added.Providers) != 1 || added.Providers[0].Path != path {
		t.Fatalf("added providers = %#v, want registered path %s", added.Providers, path)
	}

	list := serve(t, server, http.MethodGet, routeProviders, "")
	assertStatus(t, list, http.StatusOK)
	response := decodeResponse[contract.ProviderListResponse](t, list)
	if len(response.Providers) != 1 || response.Providers[0].ID != added.Providers[0].ID {
		t.Fatalf("providers = %#v, want registered provider", response.Providers)
	}
}
