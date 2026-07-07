package cli

import (
	"errors"
	"net/http"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestRunsCreatePostsInputs(t *testing.T) {
	cmd, stdout, _ := newTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/runs" {
			t.Fatalf("request = %s %s, want POST /v1/runs", r.Method, r.URL.Path)
		}
		req := decodeRequest[contract.RunRequest](t, r)
		if req.CapabilityID != "document.convert" || req.Inputs["target"] != "out.pdf" || req.MinVerifyLevel != model.VerifyLevelL1 {
			t.Fatalf("request = %#v, want run request with inputs", req)
		}
		writeResponse(t, w, contract.RunResponse{Run: &model.Run{ID: "run_1", Status: model.RunStatusSucceeded}})
	})

	if err := execute(t, cmd, "runs", "create", "--capability-id", "document.convert", "--inputs-json", `{"target":"out.pdf"}`, "--min-verify-level", "L1", "--json"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	response := decodeOutput[contract.RunResponse](t, stdout)
	if response.Run == nil || response.Run.ID != "run_1" {
		t.Fatalf("response = %#v, want run_1", response)
	}
}

func TestRunsCreateRequiresCapabilityID(t *testing.T) {
	cmd, _, _ := newTestCLI(t, nil)

	err := execute(t, cmd, "runs", "create")
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != exitUsage {
		t.Fatalf("error = %T %[1]v, want usage ExitError", err)
	}
}

func TestRunsCreateRejectsInvalidMinVerifyLevel(t *testing.T) {
	cmd, _, _ := newTestCLI(t, nil)

	err := execute(t, cmd, "runs", "create", "--capability-id", "document.convert", "--min-verify-level", "L9")
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != exitUsage {
		t.Fatalf("error = %T %[1]v, want usage ExitError", err)
	}
}
