package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/cald"
)

func newDaemonCommand(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Inspect or control the local CAL service",
	}

	cmd.AddCommand(newDaemonStartCommand(cfg))

	var jsonOut bool
	status := &cobra.Command{
		Use:   "status",
		Short: "Show local CAL service status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			status := cald.LocalStatus()
			if client, err := newCaldClient(cfg); err == nil {
				if remoteStatus, err := client.Status(cmd.Context()); err == nil {
					status = remoteStatus
				}
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), status)
			}
			if status.Running {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "cald running")
				return err
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "cald stopped")
			return err
		},
	}
	status.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")

	cmd.AddCommand(status)
	cmd.AddCommand(newDaemonStopCommand(cfg))
	return cmd
}

func newDaemonStopCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the local CAL service",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			if err := client.Stop(cmd.Context()); err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), map[string]bool{"stopping": true})
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "cald stopping")
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	return cmd
}

func newDaemonUnavailableCommand(use, short string, code commandErrorCode, message string) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if jsonOut {
				return writeJSONCommandError(cmd.OutOrStdout(), code, message)
			}
			return newCommandError(code, message)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	return cmd
}
