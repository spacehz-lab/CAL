package httpapi

import (
	"net/http"

	"github.com/spacehz-lab/cal/internal/cald/control"
	caluse "github.com/spacehz-lab/cal/internal/use"
)

type useHandler struct {
	svc control.Service
}

func (h useHandler) use(w http.ResponseWriter, r *http.Request) {
	var req caluse.Request
	if !decodeRequest(w, r, &req) {
		return
	}
	result, err := h.svc.Use(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
