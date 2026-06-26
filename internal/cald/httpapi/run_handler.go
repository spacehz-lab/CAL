package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/spacehz-lab/cal/internal/cald/control"
)

type runHandler struct {
	svc control.Service
}

func (h runHandler) run(w http.ResponseWriter, r *http.Request) {
	var req control.RunRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	run, err := h.svc.Run(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (h runHandler) get(w http.ResponseWriter, r *http.Request) {
	run, ok, err := h.svc.GetRun(chi.URLParam(r, "run_id"))
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeRecord(w, run, ok)
}
