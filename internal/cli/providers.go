package cli

import (
	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/contract"
)

const (
	flagProviderPath = "provider-path"
)

func (cli *CLI) newProvidersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Manage registered providers",
	}
	cmd.AddCommand(cli.newProvidersAddCommand())
	cmd.AddCommand(cli.newProvidersListCommand())
	return cmd
}

func (cli *CLI) newProvidersAddCommand() *cobra.Command {
	var providerPath string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Register a provider path",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := required(providerPath, flagProviderPath); err != nil {
				return commandError(cmd, jsonOut, invalidInput(err.Error()))
			}
			ctx, err := cli.commandContext()
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			response, err := ctx.client.AddProvider(cmd.Context(), &contract.AddProviderRequest{ProviderPath: providerPath})
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			return render(cmd, RenderOptions{Mode: renderMode(jsonOut)}, response, "provider registered")
		},
	}
	cmd.Flags().StringVar(&providerPath, flagProviderPath, "", "provider executable path")
	cmd.Flags().BoolVar(&jsonOut, flagJSON, false, "render machine-readable JSON")
	return cmd
}

func (cli *CLI) newProvidersListCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered providers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, err := cli.commandContext()
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			response, err := ctx.client.ListProviders(cmd.Context())
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			return render(cmd, RenderOptions{Mode: renderMode(jsonOut)}, response, "providers listed")
		},
	}
	cmd.Flags().BoolVar(&jsonOut, flagJSON, false, "render machine-readable JSON")
	return cmd
}
