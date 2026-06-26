package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/spacehz-lab/cal/internal/cald/control"
)

type evalHandler struct {
	svc control.Service
}

func (h evalHandler) metrics(w http.ResponseWriter, _ *http.Request) {
	metrics, err := h.svc.Eval()
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (h evalHandler) trace(w http.ResponseWriter, r *http.Request) {
	trace, ok, err := h.svc.GetTrace(chi.URLParam(r, "trace_id"))
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeRecord(w, trace, ok)
}
