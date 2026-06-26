package httpapi

import (
	"net/http"

	"github.com/spacehz-lab/cal/internal/cald/control"
)

type discoveryHandler struct {
	svc control.Service
}

func (h discoveryHandler) run(w http.ResponseWriter, r *http.Request) {
	var req control.DiscoveryRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	result, err := h.svc.Discover(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
