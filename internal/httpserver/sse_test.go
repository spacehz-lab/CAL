package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestRunStreamEmitsProgressAndResult(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodPost, routeRunsStream, `{"capability_id":"missing.capability","inputs":{}}`)

	assertStatus(t, rec, http.StatusOK)
	if contentType := rec.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("content-type = %q, want text/event-stream", contentType)
	}
	events := decodeSSE(t, rec.Body.String())
	if len(events) < 2 {
		t.Fatalf("events = %#v, want progress and result", events)
	}
	if events[0].Name != sseEventProgress {
		t.Fatalf("first event = %s, want progress; body = %s", events[0].Name, rec.Body.String())
	}
	if events[len(events)-1].Name != sseEventResult {
		t.Fatalf("last event = %s, want result; body = %s", events[len(events)-1].Name, rec.Body.String())
	}
	var response contract.RunResponse
	if err := json.Unmarshal(events[len(events)-1].Data, &response); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if response.Run == nil || response.Run.Status != model.RunStatusFailed {
		t.Fatalf("response = %#v, want failed run result", response)
	}
}

func TestAcquisitionStreamEmitsErrorEvent(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodPost, routeAcquisitionsStream, `{"provider_id":"provider_test"}`)

	assertStatus(t, rec, http.StatusOK)
	events := decodeSSE(t, rec.Body.String())
	if len(events) != 1 || events[0].Name != sseEventError {
		t.Fatalf("events = %#v, want one error event; body = %s", events, rec.Body.String())
	}
	var response contract.ErrorResponse
	if err := json.Unmarshal(events[0].Data, &response); err != nil {
		t.Fatalf("decode error event: %v", err)
	}
	if response.Error.Code != contract.ErrorCaldUnavailable {
		t.Fatalf("error = %#v, want cald_unavailable", response.Error)
	}
}

type sseEvent struct {
	Name string
	Data json.RawMessage
}

func decodeSSE(t *testing.T, body string) []sseEvent {
	t.Helper()
	blocks := strings.Split(strings.TrimSpace(body), "\n\n")
	events := make([]sseEvent, 0, len(blocks))
	for _, block := range blocks {
		var event sseEvent
		for _, line := range strings.Split(block, "\n") {
			if value, ok := strings.CutPrefix(line, "event:"); ok {
				event.Name = strings.TrimSpace(value)
			}
			if value, ok := strings.CutPrefix(line, "data:"); ok {
				event.Data = json.RawMessage(strings.TrimSpace(value))
			}
		}
		if event.Name != "" {
			events = append(events, event)
		}
	}
	return events
}
