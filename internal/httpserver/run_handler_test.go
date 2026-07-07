package httpserver

import (
	"net/http"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestRunPostDecodesRequestAndReturnsRun(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodPost, routeRuns, `{"capability_id":"missing.capability","inputs":{}}`)

	assertStatus(t, rec, http.StatusOK)
	response := decodeResponse[contract.RunResponse](t, rec)
	if response.Run == nil || response.Run.CapabilityID != "missing.capability" || response.Run.Status != model.RunStatusFailed {
		t.Fatalf("response = %#v, want failed run for requested capability", response)
	}
}
