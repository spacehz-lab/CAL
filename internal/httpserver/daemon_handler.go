package httpserver

import (
	"net/http"

	"github.com/spacehz-lab/cal/internal/contract"
)

func (server *Server) handleDaemonStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	status := contract.DaemonStatus{Running: true}
	if server.daemon.Status != nil {
		status = server.daemon.Status()
	}
	writeJSON(w, http.StatusOK, status)
}

func (server *Server) handleDaemonStop(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if server.daemon.Stop == nil {
		writeTransportError(w, http.StatusServiceUnavailable, contract.ErrorCaldUnavailable, "daemon stop is not configured")
		return
	}
	writeJSON(w, http.StatusOK, contract.DaemonStopResponse{Stopping: true})
	go server.daemon.Stop()
}
