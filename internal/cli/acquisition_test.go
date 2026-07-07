package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
)

func TestAcquisitionRunPostsRequest(t *testing.T) {
	cmd, stdout, _ := newTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/acquisitions" {
			t.Fatalf("request = %s %s, want POST /v1/acquisitions", r.Method, r.URL.Path)
		}
		req := decodeRequest[contract.AcquisitionRequest](t, r)
		if req.ProviderID != "provider_cli" || req.Hint != "convert a document" || req.ProposalPath != "/tmp/proposal.json" || req.Mode != contract.AcquisitionModeReplay {
			t.Fatalf("request = %#v, want provider replay request", req)
		}
		writeResponse(t, w, contract.AcquisitionResponse{TraceID: "trace_1"})
	})

	if err := execute(t, cmd, "acquisition", "run", "--provider-id", "provider_cli", "--hint", "convert a document", "--proposal-path", "/tmp/proposal.json", "--mode", "replay", "--json"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	response := decodeOutput[contract.AcquisitionResponse](t, stdout)
	if response.TraceID != "trace_1" {
		t.Fatalf("response = %#v, want trace_1", response)
	}
}

func TestAcquisitionRunStreamJSONOutputsJSONLines(t *testing.T) {
	cmd, stdout, _ := newTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/acquisitions/stream" {
			t.Fatalf("request = %s %s, want POST /v1/acquisitions/stream", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: progress\n")
		fmt.Fprint(w, `data: {"stage":"proposal","status":"started"}`+"\n\n")
		fmt.Fprint(w, "event: result\n")
		fmt.Fprint(w, `data: {"trace_id":"trace_1","capabilities_promoted":1}`+"\n\n")
	})

	if err := execute(t, cmd, "acquisition", "run", "--provider-id", "provider_cli", "--stream", "--json"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("output lines = %#v, want progress and result", lines)
	}
	var first struct {
		Event string `json:"event"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("decode first line: %v", err)
	}
	if first.Event != "progress" {
		t.Fatalf("first event = %q, want progress", first.Event)
	}
	var second struct {
		Event string `json:"event"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("decode second line: %v", err)
	}
	if second.Event != "result" {
		t.Fatalf("second event = %q, want result", second.Event)
	}
}
