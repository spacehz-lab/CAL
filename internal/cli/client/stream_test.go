package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestRunStreamDecodesProgressAndResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != routeRunsStream {
			t.Fatalf("request = %s %s, want POST %s", r.Method, r.URL.Path, routeRunsStream)
		}
		if accept := r.Header.Get("Accept"); accept != "text/event-stream" {
			t.Fatalf("accept = %q, want text/event-stream", accept)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: progress\n")
		fmt.Fprint(w, `data: {"stage":"execute","status":"started"}`+"\n\n")
		fmt.Fprint(w, "event: result\n")
		fmt.Fprint(w, `data: {"run":{"id":"run_1","status":"succeeded"}}`+"\n\n")
	}))
	defer server.Close()
	client := newTestClient(t, server)

	var events []StreamEventName
	response, err := client.RunStream(context.Background(), &contract.RunRequest{CapabilityID: "document.convert"}, func(_ context.Context, event *StreamEvent) error {
		events = append(events, event.Name)
		return nil
	})
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if response.Run == nil || response.Run.ID != "run_1" || response.Run.Status != model.RunStatusSucceeded {
		t.Fatalf("response = %#v, want succeeded run", response)
	}
	want := []StreamEventName{StreamEventProgress, StreamEventResult}
	if len(events) != len(want) {
		t.Fatalf("events = %#v, want %#v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("events = %#v, want %#v", events, want)
		}
	}
}

func TestAcquireStreamDecodesErrorEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: error\n")
		fmt.Fprint(w, `data: {"error":{"code":"cald_unavailable","message":"llm is not configured"}}`+"\n\n")
	}))
	defer server.Close()
	client := newTestClient(t, server)

	_, err := client.AcquireStream(context.Background(), &contract.AcquisitionRequest{}, nil)
	assertClientError(t, err, 0, contract.ErrorCaldUnavailable)
}

func TestStreamCallbackErrorStopsStream(t *testing.T) {
	callbackErr := errors.New("stop stream")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: progress\n")
		fmt.Fprint(w, `data: {"stage":"execute","status":"started"}`+"\n\n")
	}))
	defer server.Close()
	client := newTestClient(t, server)

	_, err := client.RunStream(context.Background(), &contract.RunRequest{}, func(context.Context, *StreamEvent) error {
		return callbackErr
	})
	if !errors.Is(err, callbackErr) {
		t.Fatalf("error = %v, want callback error", err)
	}
}
