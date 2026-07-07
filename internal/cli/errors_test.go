package cli

import (
	"errors"
	"net/http"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
)

func TestCommandErrorRendersStructuredJSON(t *testing.T) {
	cmd, stdout, _ := newTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeResponse(t, w, contract.ErrorResponse{Error: contract.Error{Code: contract.ErrorInvalidRequest, Message: "bad request"}})
	})

	err := execute(t, cmd, "providers", "list", "--json")
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != exitUsage {
		t.Fatalf("error = %T %[1]v, want usage ExitError", err)
	}
	response := decodeOutput[contract.ErrorResponse](t, stdout)
	if response.Error.Code != contract.ErrorInvalidRequest || response.Error.Message != "bad request" {
		t.Fatalf("response = %#v, want invalid_request", response)
	}
}
