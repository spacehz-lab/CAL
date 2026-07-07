package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/progress"
)

const (
	sseEventProgress = "progress"
	sseEventResult   = "result"
	sseEventError    = "error"
)

const progressBufferSize = 64

type streamMessage struct {
	event string
	data  any
	done  bool
}

func streamResult[T any](w http.ResponseWriter, r *http.Request, run func(context.Context) (*T, error)) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeTransportError(w, http.StatusInternalServerError, contract.ErrorInternal, "streaming is not supported")
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	messages := make(chan streamMessage, progressBufferSize)
	ctx = progress.WithHandler(ctx, func(ctx context.Context, event *model.ProgressEvent) {
		select {
		case messages <- streamMessage{event: sseEventProgress, data: event}:
		case <-ctx.Done():
		}
	})

	go func() {
		result, err := run(ctx)
		if err != nil {
			sendStreamMessage(ctx, messages, streamMessage{event: sseEventError, data: appErrorResponse(err), done: true})
			return
		}
		sendStreamMessage(ctx, messages, streamMessage{event: sseEventResult, data: result, done: true})
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	for {
		select {
		case <-r.Context().Done():
			return
		case message := <-messages:
			if err := writeSSE(w, message.event, message.data); err != nil {
				cancel()
				return
			}
			flusher.Flush()
			if message.done {
				return
			}
		}
	}
}

func writeSSE(w http.ResponseWriter, event string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return err
	}
	return nil
}

func sendStreamMessage(ctx context.Context, messages chan<- streamMessage, message streamMessage) {
	select {
	case messages <- message:
	case <-ctx.Done():
	}
}
