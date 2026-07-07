package httpserver

import (
	"net/http"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestUsePostDecodesRequestAndReturnsUseResponse(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodPost, routeUses, `{"intent":"resize image"}`)

	assertStatus(t, rec, http.StatusOK)
	response := decodeResponse[contract.UseResponse](t, rec)
	if response.Intent != "resize image" || response.Status != model.RunStatusFailed || response.Error == nil {
		t.Fatalf("response = %#v, want failed use response for requested intent", response)
	}
}
