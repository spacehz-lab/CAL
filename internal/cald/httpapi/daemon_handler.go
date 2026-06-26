package httpapi

import (
	"net/http"

	"github.com/spacehz-lab/cal/internal/cald/control"
)

type daemonHandler struct {
	status control.Status
	stop   func()
}

func (h daemonHandler) statusHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.status)
}

func (h daemonHandler) stopHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"stopping": true})
	if h.stop != nil {
		go h.stop()
	}
}
