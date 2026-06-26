package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/spacehz-lab/cal/internal/cald/control"
	"github.com/spacehz-lab/cal/internal/runtime"
)

type capabilityHandler struct {
	svc control.Service
}

func (h capabilityHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListCapabilities(runtime.ListOptions{
		CapabilityID: r.URL.Query().Get("capability_id"),
		ProviderID:   r.URL.Query().Get("provider_id"),
	})
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h capabilityHandler) get(w http.ResponseWriter, r *http.Request) {
	capability, ok, err := h.svc.GetCapability(chi.URLParam(r, "capability_id"))
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeRecord(w, capability, ok)
}
