package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newTracesCommand(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "traces",
		Short: "Inspect acquisition traces",
	}
	cmd.AddCommand(newTracesGetCommand(cfg))
	return cmd
}

func newTracesGetCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	var traceID string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get one stored trace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if traceID == "" {
				return writeCommandError(cmd, jsonOut, newCommandError(commandErrorInvalidTraceInput, "trace_id is required"))
			}
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			trace, err := client.GetTrace(cmd.Context(), traceID)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), trace)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", trace.ID, trace.Status)
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	cmd.Flags().StringVar(&traceID, "trace-id", "", "trace_id path parameter")
	return cmd
}
