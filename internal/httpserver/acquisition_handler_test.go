package httpserver

import (
	"net/http"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
)

func TestAcquisitionMapsMissingLLMToUnavailable(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodPost, routeAcquisitions, `{"provider_id":"provider_test"}`)

	assertStatus(t, rec, http.StatusServiceUnavailable)
	assertErrorCode(t, rec, contract.ErrorCaldUnavailable)
}

func TestAcquisitionMapsUnsupportedModeToInvalidRequest(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodPost, routeAcquisitions, `{"mode":"replay"}`)

	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorCode(t, rec, contract.ErrorInvalidRequest)
}
