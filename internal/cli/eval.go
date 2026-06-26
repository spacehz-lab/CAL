package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newEvalCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Summarize discovery and reuse evidence",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			metrics, err := client.Eval(cmd.Context())
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), metrics)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "providers=%d capabilities=%d bindings=%d promoted_bindings=%d traces=%d runs=%d\n", metrics.Summary.Providers, metrics.Summary.Capabilities, metrics.Summary.Bindings, metrics.Summary.PromotedBindings, metrics.Summary.Traces, metrics.Summary.Runs)
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	return cmd
}
