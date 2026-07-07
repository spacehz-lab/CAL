package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestListCapabilitiesEncodesQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != routeCapabilities {
			t.Fatalf("request = %s %s, want GET %s", r.Method, r.URL.Path, routeCapabilities)
		}
		if got := r.URL.Query().Get(queryCapabilityID); got != "document.convert" {
			t.Fatalf("capability query = %q, want document.convert", got)
		}
		if got := r.URL.Query().Get(queryProviderID); got != "provider_cli" {
			t.Fatalf("provider query = %q, want provider_cli", got)
		}
		writeJSON(t, w, contract.CapabilityListResponse{Count: 1})
	}))
	defer server.Close()
	client := newTestClient(t, server)

	response, err := client.ListCapabilities(context.Background(), &contract.CapabilityListRequest{
		CapabilityID: "document.convert",
		ProviderID:   "provider_cli",
	})
	if err != nil {
		t.Fatalf("ListCapabilities() error = %v", err)
	}
	if response.Count != 1 {
		t.Fatalf("response = %#v, want count 1", response)
	}
}

func TestEvalEncodesQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != routeEval {
			t.Fatalf("request = %s %s, want GET %s", r.Method, r.URL.Path, routeEval)
		}
		if got := r.URL.Query().Get(queryCapabilityID); got != "document.convert" {
			t.Fatalf("capability query = %q, want document.convert", got)
		}
		if got := r.URL.Query().Get(queryProviderID); got != "provider_cli" {
			t.Fatalf("provider query = %q, want provider_cli", got)
		}
		writeJSON(t, w, contract.EvalResponse{Capability: contract.CapabilityMetrics{Capabilities: 1}})
	}))
	defer server.Close()
	client := newTestClient(t, server)

	response, err := client.Eval(context.Background(), &contract.EvalRequest{
		CapabilityID: "document.convert",
		ProviderID:   "provider_cli",
	})
	if err != nil {
		t.Fatalf("Eval() error = %v", err)
	}
	if response.Capability.Capabilities != 1 {
		t.Fatalf("response = %#v, want one capability", response)
	}
}

func TestPostMethodsEncodeRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case routeDaemonStop:
			assertMethod(t, r, http.MethodPost)
			writeJSON(t, w, contract.DaemonStopResponse{Stopping: true})
		case routeProviders:
			assertMethod(t, r, http.MethodPost)
			var req contract.AddProviderRequest
			decodeJSON(t, r, &req)
			if req.ProviderPath != "/bin/echo" {
				t.Fatalf("provider path = %q, want /bin/echo", req.ProviderPath)
			}
			writeJSON(t, w, contract.ProviderListResponse{Providers: []model.Provider{{ID: "provider_cli"}}})
		case routeAcquisitions:
			assertMethod(t, r, http.MethodPost)
			var req contract.AcquisitionRequest
			decodeJSON(t, r, &req)
			if req.ProviderID != "provider_cli" {
				t.Fatalf("provider id = %q, want provider_cli", req.ProviderID)
			}
			writeJSON(t, w, contract.AcquisitionResponse{TraceID: "trace_1"})
		case routeRuns:
			assertMethod(t, r, http.MethodPost)
			var req contract.RunRequest
			decodeJSON(t, r, &req)
			if req.CapabilityID != "document.convert" {
				t.Fatalf("capability id = %q, want document.convert", req.CapabilityID)
			}
			writeJSON(t, w, contract.RunResponse{Run: &model.Run{ID: "run_1", Status: model.RunStatusSucceeded}})
		case routeUses:
			assertMethod(t, r, http.MethodPost)
			var req contract.UseRequest
			decodeJSON(t, r, &req)
			if req.Intent != "convert" {
				t.Fatalf("intent = %q, want convert", req.Intent)
			}
			writeJSON(t, w, contract.UseResponse{ID: "use_1", Status: model.RunStatusSucceeded})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := newTestClient(t, server)

	stop, err := client.Stop(context.Background())
	if err != nil || !stop.Stopping {
		t.Fatalf("Stop() = %#v, %v; want stopping response", stop, err)
	}
	providers, err := client.AddProvider(context.Background(), &contract.AddProviderRequest{ProviderPath: "/bin/echo"})
	if err != nil || len(providers.Providers) != 1 {
		t.Fatalf("AddProvider() = %#v, %v; want one provider", providers, err)
	}
	acquisition, err := client.Acquire(context.Background(), &contract.AcquisitionRequest{ProviderID: "provider_cli"})
	if err != nil || acquisition.TraceID != "trace_1" {
		t.Fatalf("Acquire() = %#v, %v; want trace_1", acquisition, err)
	}
	run, err := client.Run(context.Background(), &contract.RunRequest{CapabilityID: "document.convert"})
	if err != nil || run.Run == nil || run.Run.ID != "run_1" {
		t.Fatalf("Run() = %#v, %v; want run_1", run, err)
	}
	use, err := client.Use(context.Background(), &contract.UseRequest{Intent: "convert"})
	if err != nil || use.ID != "use_1" {
		t.Fatalf("Use() = %#v, %v; want use_1", use, err)
	}
}

func TestListProvidersCallsGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != routeProviders {
			t.Fatalf("request = %s %s, want GET %s", r.Method, r.URL.Path, routeProviders)
		}
		writeJSON(t, w, contract.ProviderListResponse{Providers: []model.Provider{{ID: "provider_cli"}}})
	}))
	defer server.Close()
	client := newTestClient(t, server)

	response, err := client.ListProviders(context.Background())
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}
	if len(response.Providers) != 1 || response.Providers[0].ID != "provider_cli" {
		t.Fatalf("response = %#v, want provider_cli", response)
	}
}

func assertMethod(t *testing.T, r *http.Request, method string) {
	t.Helper()
	if r.Method != method {
		t.Fatalf("method = %s, want %s", r.Method, method)
	}
}

func decodeJSON(t *testing.T, r *http.Request, target any) {
	t.Helper()
	if got := r.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
}
