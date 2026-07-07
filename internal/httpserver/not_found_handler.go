package httpserver

import (
	"net/http"

	"github.com/spacehz-lab/cal/internal/contract"
)

func (server *Server) handleNotFound(w http.ResponseWriter, _ *http.Request) {
	writeTransportError(w, http.StatusNotFound, contract.ErrorNotFound, "route not found")
}
