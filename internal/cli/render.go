package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/cli/client"
)

// RenderMode selects CLI output format.
type RenderMode string

const (
	RenderText RenderMode = "text"
	RenderJSON RenderMode = "json"
)

// RenderOptions controls one CLI render operation.
type RenderOptions struct {
	Mode RenderMode
}

func render(cmd *cobra.Command, opts RenderOptions, value any, text string) error {
	if opts.Mode == RenderJSON {
		return writeJSON(cmd.OutOrStdout(), value)
	}
	if text == "" {
		text = "ok"
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), text)
	return err
}

func renderMode(jsonOut bool) RenderMode {
	if jsonOut {
		return RenderJSON
	}
	return RenderText
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

type streamRenderer struct {
	cmd      *cobra.Command
	jsonOut  bool
	terminal bool
}

func newStreamRenderer(cmd *cobra.Command, jsonOut bool) *streamRenderer {
	return &streamRenderer{cmd: cmd, jsonOut: jsonOut}
}

func (renderer *streamRenderer) Handle(_ context.Context, event *client.StreamEvent) error {
	if event == nil {
		return nil
	}
	if event.Name == client.StreamEventResult || event.Name == client.StreamEventError {
		renderer.terminal = true
	}
	if renderer.jsonOut {
		return writeJSONLine(renderer.cmd.OutOrStdout(), streamEnvelope{Event: string(event.Name), Data: event.Data})
	}
	if event.Name != client.StreamEventProgress {
		return nil
	}
	text := progressText(event.Data)
	if text == "" {
		return nil
	}
	_, err := fmt.Fprintln(renderer.cmd.ErrOrStderr(), text)
	return err
}

func (renderer *streamRenderer) TerminalSeen() bool {
	return renderer != nil && renderer.terminal
}

type streamEnvelope struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

func writeJSONLine(out io.Writer, value any) error {
	return json.NewEncoder(out).Encode(value)
}

func progressText(data json.RawMessage) string {
	var event struct {
		Stage  string `json:"stage"`
		Step   string `json:"step"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Join([]string{event.Stage, event.Step, event.Status}, " "))
}
