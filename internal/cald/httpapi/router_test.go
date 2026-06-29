package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/cald/control"
	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/store"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestRouterDaemonStatusAndStop(t *testing.T) {
	stopped := make(chan struct{})
	router := NewRouter(RouterConfig{
		Status: control.Status{Running: true, Mode: "local", Endpoint: "http://127.0.0.1:1"},
		Stop:   func() { close(stopped) },
	})

	status := serveRequest(t, router, http.MethodGet, "/v1/daemon/status", "")
	if status.Code != http.StatusOK || !strings.Contains(status.Body.String(), `"running": true`) {
		t.Fatalf("status response = %d %s, want running status", status.Code, status.Body.String())
	}

	stop := serveRequest(t, router, http.MethodPost, "/v1/daemon/stop", "")
	if stop.Code != http.StatusOK || !strings.Contains(stop.Body.String(), `"stopping": true`) {
		t.Fatalf("stop response = %d %s, want stopping", stop.Code, stop.Body.String())
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("stop callback was not called")
	}
}

func TestRouterReadsStoredRecords(t *testing.T) {
	svc := newHTTPTestService(t)
	seedHTTPRecords(t, svc.Home())
	router := NewRouter(RouterConfig{Service: svc})

	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{name: "providers", path: "/v1/providers", want: "provider_test"},
		{name: "provider detail", path: "/v1/providers/provider_test", want: "test-provider"},
		{name: "capabilities", path: "/v1/capabilities", want: "document.export_pdf"},
		{name: "capability detail", path: "/v1/capabilities/document.export_pdf", want: "Export a document to PDF."},
		{name: "run detail", path: "/v1/runs/run_test", want: "run_test"},
		{name: "eval", path: "/v1/eval", want: `"providers": 1`},
		{name: "trace detail", path: "/v1/traces/trace_test", want: "trace_test"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := serveRequest(t, router, http.MethodGet, test.path, "")
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), test.want) {
				t.Fatalf("response = %d %s, want %q", response.Code, response.Body.String(), test.want)
			}
		})
	}

	missing := serveRequest(t, router, http.MethodGet, "/v1/providers/provider_missing", "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing response = %d %s, want 404", missing.Code, missing.Body.String())
	}
	assertErrorCode(t, missing, "not_found")
}

func TestRouterRejectsInvalidRequestShape(t *testing.T) {
	svc := newHTTPTestService(t)
	router := NewRouter(RouterConfig{Service: svc})

	response := serveRequest(t, router, http.MethodPost, "/v1/providers", `{"unexpected":true}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("response = %d %s, want 400", response.Code, response.Body.String())
	}
	assertErrorCode(t, response, "invalid_request")
}

func TestRouterDiscoveryRejectsMissingTarget(t *testing.T) {
	svc := newHTTPTestService(t)
	router := NewRouter(RouterConfig{Service: svc})

	response := serveRequest(t, router, http.MethodPost, "/v1/discovery", `{}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("response = %d %s, want 422", response.Code, response.Body.String())
	}
	assertErrorCode(t, response, "invalid_discovery_target")
}

func TestRouterDiscoveryRejectsProviderPathField(t *testing.T) {
	svc := newHTTPTestService(t)
	router := NewRouter(RouterConfig{Service: svc})

	response := serveRequest(t, router, http.MethodPost, "/v1/discovery", `{"provider_path":"/tmp/missing-provider"}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("response = %d %s, want 400 unknown field", response.Code, response.Body.String())
	}
	assertErrorCode(t, response, "invalid_request")
}

func TestRouterDoesNotExposeDiscoveryJobLookup(t *testing.T) {
	svc := newHTTPTestService(t)
	router := NewRouter(RouterConfig{Service: svc})

	response := serveRequest(t, router, http.MethodGet, "/v1/discovery/disc_123", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("response = %d %s, want 404", response.Code, response.Body.String())
	}
}

func TestRouterRunInvalidInput(t *testing.T) {
	svc := newHTTPTestService(t)
	router := NewRouter(RouterConfig{Service: svc})

	response := serveRequest(t, router, http.MethodPost, "/v1/runs", `{}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("response = %d %s, want 422", response.Code, response.Body.String())
	}
	assertErrorCode(t, response, "invalid_run_input")
}

func TestRouterUseInvalidInput(t *testing.T) {
	svc := newHTTPTestService(t)
	router := NewRouter(RouterConfig{Service: svc})

	response := serveRequest(t, router, http.MethodPost, "/v1/uses", `{}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("response = %d %s, want 422", response.Code, response.Body.String())
	}
	assertErrorCode(t, response, "invalid_use_input")
}

func newHTTPTestService(t *testing.T) control.Service {
	t.Helper()
	svc, err := control.NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return svc
}

func serveRequest(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response %q: %v", response.Body.String(), err)
	}
}

func assertErrorCode(t *testing.T, response *httptest.ResponseRecorder, code string) {
	t.Helper()
	var body struct {
		Error control.APIError `json:"error"`
	}
	decodeResponse(t, response, &body)
	if body.Error.Code != code {
		t.Fatalf("error code = %q, want %q in %s", body.Error.Code, code, response.Body.String())
	}
}

func seedHTTPRecords(t *testing.T, home string) {
	t.Helper()
	s, err := store.Open(home)
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	provider := core.Provider{
		ID:   "provider_test",
		Name: "test-provider",
		Kind: core.ProviderKindCLI,
		Path: "/tmp/provider-test",
	}
	capability := testHTTPCapability(t, provider.ID)
	run := core.Run{
		ID:           "run_test",
		CapabilityID: capability.ID,
		Status:       core.RunStatusSucceeded,
	}
	trace := caltrace.Trace{ID: "trace_test", Status: caltrace.StatusCompleted}
	if err := s.PutProvider(provider); err != nil {
		t.Fatalf("PutProvider() error = %v", err)
	}
	if err := s.PutCapability(capability); err != nil {
		t.Fatalf("PutCapability() error = %v", err)
	}
	if err := s.PutRun(run); err != nil {
		t.Fatalf("PutRun() error = %v", err)
	}
	if err := s.PutTrace(trace); err != nil {
		t.Fatalf("PutTrace() error = %v", err)
	}
}

func testHTTPCapability(t *testing.T, providerID string) core.Capability {
	t.Helper()
	capabilityID := "document.export_pdf"
	execution := core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{core.ExecutionSpecArgs: []string{"--version"}},
	}
	bindingID, err := core.BindingIDForExecution(capabilityID, providerID, execution)
	if err != nil {
		t.Fatalf("BindingIDForExecution() error = %v", err)
	}
	return core.Capability{
		ID:          capabilityID,
		Description: "Export a document to PDF.",
		Bindings: []core.Binding{{
			ID:           bindingID,
			CapabilityID: capabilityID,
			ProviderID:   providerID,
			Execution:    execution,
			Verify:       testHTTPVerifySpec(),
			Evidence:     []core.EvidenceRef{{ID: "evidence_test"}},
			State:        core.BindingStatePromoted,
		}},
	}
}

func testHTTPVerifySpec() *core.VerifySpec {
	return &core.VerifySpec{
		Level:  core.VerifyLevelL2,
		Method: core.VerifyMethodExecute,
		Checks: []core.VerifyCheck{{Subject: "target", Predicate: core.VerifyPredicateExists}},
	}
}
