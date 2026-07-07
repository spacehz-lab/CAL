package httpserver

import (
	"net/http"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
)

func TestEvalQueryReturnsMetrics(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodGet, routeEval+"?provider_id=provider_test&capability_id=document.echo", "")

	assertStatus(t, rec, http.StatusOK)
	response := decodeResponse[contract.EvalResponse](t, rec)
	if response.Acquisition.Traces.Total != 0 || response.Reuse.Runs.Total != 0 || response.Capability.Capabilities != 0 {
		t.Fatalf("response = %#v, want empty metrics", response)
	}
}
