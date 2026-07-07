package cli

import (
	"net/http"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestUsePostsIntentAndInputs(t *testing.T) {
	cmd, stdout, _ := newTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/uses" {
			t.Fatalf("request = %s %s, want POST /v1/uses", r.Method, r.URL.Path)
		}
		req := decodeRequest[contract.UseRequest](t, r)
		if req.Intent != "convert document" || req.Inputs["target"] != "out.pdf" || req.MinVerifyLevel != model.VerifyLevelL1 {
			t.Fatalf("request = %#v, want use request with inputs", req)
		}
		writeResponse(t, w, contract.UseResponse{ID: "use_1", Status: model.RunStatusSucceeded})
	})

	if err := execute(t, cmd, "use", "convert document", "--inputs-json", `{"target":"out.pdf"}`, "--min-verify-level", "L1", "--json"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	response := decodeOutput[contract.UseResponse](t, stdout)
	if response.ID != "use_1" {
		t.Fatalf("response = %#v, want use_1", response)
	}
}
