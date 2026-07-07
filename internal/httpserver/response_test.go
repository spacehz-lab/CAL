package httpserver

import (
	"net/http"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
)

func TestMalformedJSONReturnsInvalidRequest(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodPost, routeUses, `{`)

	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorCode(t, rec, contract.ErrorInvalidRequest)
}

func TestUnknownJSONFieldReturnsInvalidRequest(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodPost, routeUses, `{"intent":"resize image","unknown":true}`)

	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorCode(t, rec, contract.ErrorInvalidRequest)
}

func TestEmptyJSONBodyReturnsInvalidRequest(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodPost, routeUses, "")

	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorCode(t, rec, contract.ErrorInvalidRequest)
}

func TestMultipleJSONValuesReturnInvalidRequest(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodPost, routeUses, `{"intent":"resize image"} {}`)

	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorCode(t, rec, contract.ErrorInvalidRequest)
}
