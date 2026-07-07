package cli

import (
	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/contract"
)

func (cli *CLI) newCapabilitiesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capabilities",
		Short: "Inspect reusable capabilities",
	}
	cmd.AddCommand(cli.newCapabilitiesListCommand())
	return cmd
}

func (cli *CLI) newCapabilitiesListCommand() *cobra.Command {
	var capabilityID string
	var jsonOut bool
	var providerID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List reusable capabilities",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, err := cli.commandContext()
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			response, err := ctx.client.ListCapabilities(cmd.Context(), &contract.CapabilityListRequest{
				CapabilityID: capabilityID,
				ProviderID:   providerID,
			})
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			return render(cmd, RenderOptions{Mode: renderMode(jsonOut)}, response, "capabilities listed")
		},
	}
	cmd.Flags().StringVar(&capabilityID, flagCapabilityID, "", "capability id filter")
	cmd.Flags().StringVar(&providerID, flagProviderID, "", "provider id filter")
	cmd.Flags().BoolVar(&jsonOut, flagJSON, false, "render machine-readable JSON")
	return cmd
}
