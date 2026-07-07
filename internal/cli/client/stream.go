package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spacehz-lab/cal/internal/contract"
)

const (
	StreamEventProgress StreamEventName = "progress"
	StreamEventResult   StreamEventName = "result"
	StreamEventError    StreamEventName = "error"
)

// StreamEventName identifies one local daemon SSE event type.
type StreamEventName string

// StreamEvent is the decoded SSE envelope passed to CLI rendering.
type StreamEvent struct {
	Name StreamEventName
	Data json.RawMessage
}

// StreamHandler observes one decoded SSE envelope.
type StreamHandler func(context.Context, *StreamEvent) error

func (client *Client) stream(ctx context.Context, path string, body any, onEvent StreamHandler, target any) error {
	req, err := client.newRequest(ctx, http.MethodPost, path, nil, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.http.Do(req)
	if err != nil {
		return newError(0, contract.ErrorCaldUnavailable, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}
		return decodeErrorResponse(resp.StatusCode, data)
	}
	return client.readStream(ctx, resp.Body, onEvent, target)
}

func (client *Client) readStream(ctx context.Context, body io.Reader, onEvent StreamHandler, target any) error {
	scanner := bufio.NewScanner(body)
	var name StreamEventName
	var data json.RawMessage
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			done, err := handleStreamEvent(ctx, name, data, onEvent, target)
			if done || err != nil {
				return err
			}
			name = ""
			data = nil
			continue
		}
		if value, ok := strings.CutPrefix(line, "event:"); ok {
			name = StreamEventName(strings.TrimSpace(value))
			continue
		}
		if value, ok := strings.CutPrefix(line, "data:"); ok {
			data = json.RawMessage(strings.TrimSpace(value))
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stream: %w", err)
	}
	return fmt.Errorf("stream ended without result")
}

func handleStreamEvent(ctx context.Context, name StreamEventName, data json.RawMessage, onEvent StreamHandler, target any) (bool, error) {
	if name == "" {
		return false, nil
	}
	event := &StreamEvent{Name: name, Data: data}
	if onEvent != nil {
		if err := onEvent(ctx, event); err != nil {
			return true, err
		}
	}
	switch name {
	case StreamEventProgress:
		return false, nil
	case StreamEventResult:
		if target != nil {
			if err := json.Unmarshal(data, target); err != nil {
				return true, fmt.Errorf("decode stream result: %w", err)
			}
		}
		return true, nil
	case StreamEventError:
		var response contract.ErrorResponse
		if err := json.Unmarshal(data, &response); err != nil {
			return true, fmt.Errorf("decode stream error: %w", err)
		}
		return true, newError(0, response.Error.Code, response.Error.Message)
	default:
		return true, fmt.Errorf("unsupported stream event %q", name)
	}
}
