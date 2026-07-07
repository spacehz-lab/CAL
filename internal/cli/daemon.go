package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/cli/client"
	"github.com/spacehz-lab/cal/internal/contract"
)

func (cli *CLI) newDaemonCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Inspect or control the local CAL daemon",
	}
	cmd.AddCommand(cli.newDaemonStartCommand())
	cmd.AddCommand(cli.newDaemonStatusCommand())
	cmd.AddCommand(cli.newDaemonStopCommand())
	return cmd
}

func (cli *CLI) newDaemonStartCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the local CAL daemon in the background",
		RunE: func(cmd *cobra.Command, _ []string) error {
			status, err := cli.newDaemonStarter().Start(cmd.Context())
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			return render(cmd, RenderOptions{Mode: renderMode(jsonOut)}, status, "cald running")
		},
	}
	cmd.Flags().BoolVar(&jsonOut, flagJSON, false, "render machine-readable JSON")
	return cmd
}

func (cli *CLI) newDaemonStatusCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show local CAL daemon status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, err := cli.commandContext()
			if err != nil {
				var clientErr *client.Error
				if errors.As(err, &clientErr) && clientErr.Code == contract.ErrorCaldUnavailable {
					return render(cmd, RenderOptions{Mode: renderMode(jsonOut)}, contract.DaemonStatus{Running: false}, "cald stopped")
				}
				return commandError(cmd, jsonOut, err)
			}
			status, err := ctx.client.Status(cmd.Context())
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			text := "cald stopped"
			if status.Running {
				text = "cald running"
			}
			return render(cmd, RenderOptions{Mode: renderMode(jsonOut)}, status, text)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, flagJSON, false, "render machine-readable JSON")
	return cmd
}

func (cli *CLI) newDaemonStopCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the local CAL daemon",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, err := cli.commandContext()
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			response, err := ctx.client.Stop(cmd.Context())
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			return render(cmd, RenderOptions{Mode: renderMode(jsonOut)}, response, "cald stopping")
		},
	}
	cmd.Flags().BoolVar(&jsonOut, flagJSON, false, "render machine-readable JSON")
	return cmd
}
