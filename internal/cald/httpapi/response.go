package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/spacehz-lab/cal/internal/cald/control"
	"github.com/spacehz-lab/cal/internal/discovery"
)

type errorResponse struct {
	Error control.APIError `json:"error"`
}

func decodeRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, control.NewAPIError("invalid_request", err.Error()))
		return false
	}
	return true
}

func writeRecord[T any](w http.ResponseWriter, record T, ok bool) {
	if !ok {
		writeError(w, http.StatusNotFound, control.NewAPIError("not_found", "record was not found"))
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func writeDomainError(w http.ResponseWriter, err error) {
	var apiErr control.APIError
	if errors.As(err, &apiErr) {
		writeError(w, http.StatusUnprocessableEntity, apiErr)
		return
	}
	var codedErr discovery.CodedError
	if errors.As(err, &codedErr) {
		writeError(w, http.StatusUnprocessableEntity, control.NewAPIError(codedErr.Code, codedErr.Message))
		return
	}
	writeInternalError(w, err)
}

func writeInternalError(w http.ResponseWriter, err error) {
	writeError(w, http.StatusInternalServerError, control.NewAPIError("internal_error", err.Error()))
}

func writeError(w http.ResponseWriter, status int, err control.APIError) {
	writeJSON(w, status, errorResponse{Error: err})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}
