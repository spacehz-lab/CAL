package cli

import (
	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/contract"
)

const (
	flagCapabilityID = "capability-id"
	flagHint         = "hint"
	flagMode         = "mode"
	flagProposalPath = "proposal-path"
	flagProviderID   = "provider-id"
)

func (cli *CLI) newAcquisitionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acquisition",
		Short: "Run capability acquisition",
	}
	cmd.AddCommand(cli.newAcquisitionRunCommand())
	return cmd
}

func (cli *CLI) newAcquisitionRunCommand() *cobra.Command {
	var hint string
	var jsonOut bool
	var mode string
	var proposalPath string
	var providerID string
	var stream bool
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run acquisition against registered providers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, err := cli.commandContext()
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			req := &contract.AcquisitionRequest{
				ProviderID:   providerID,
				Hint:         hint,
				ProposalPath: proposalPath,
				Mode:         contract.AcquisitionMode(mode),
			}
			if stream {
				renderer := newStreamRenderer(cmd, jsonOut)
				response, err := ctx.client.AcquireStream(cmd.Context(), req, renderer.Handle)
				if err != nil {
					return commandError(cmd, jsonOut && !renderer.TerminalSeen(), err)
				}
				if jsonOut {
					return nil
				}
				return render(cmd, RenderOptions{Mode: RenderText}, response, "acquisition completed")
			}
			response, err := ctx.client.Acquire(cmd.Context(), req)
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			return render(cmd, RenderOptions{Mode: renderMode(jsonOut)}, response, "acquisition completed")
		},
	}
	cmd.Flags().StringVar(&providerID, flagProviderID, "", "provider id filter")
	cmd.Flags().StringVar(&hint, flagHint, "", "natural-language acquisition hint")
	cmd.Flags().StringVar(&proposalPath, flagProposalPath, "", "proposal replay path")
	cmd.Flags().StringVar(&mode, flagMode, string(contract.AcquisitionModeLive), "acquisition mode")
	cmd.Flags().BoolVar(&stream, flagStream, false, "stream progress events")
	cmd.Flags().BoolVar(&jsonOut, flagJSON, false, "render machine-readable JSON")
	return cmd
}
