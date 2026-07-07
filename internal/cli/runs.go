package cli

import (
	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/contract"
)

const (
	flagBindingID  = "binding-id"
	flagInputsJSON = "inputs-json"
	flagStrategy   = "strategy"
	flagVerify     = "verify"
	flagMinVerify  = "min-verify-level"
)

func (cli *CLI) newRunsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runs",
		Short: "Execute promoted capability bindings",
	}
	cmd.AddCommand(cli.newRunsCreateCommand())
	return cmd
}

func (cli *CLI) newRunsCreateCommand() *cobra.Command {
	var bindingID string
	var capabilityID string
	var inputsJSON string
	var jsonOut bool
	var minVerifyLevel string
	var providerID string
	var strategy string
	var stream bool
	var verify bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create one capability run",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := required(capabilityID, flagCapabilityID); err != nil {
				return commandError(cmd, jsonOut, invalidInput(err.Error()))
			}
			inputs, err := parseInputsJSON(inputsJSON)
			if err != nil {
				return commandError(cmd, jsonOut, invalidInput(err.Error()))
			}
			minLevel, err := parseMinVerifyLevel(minVerifyLevel)
			if err != nil {
				return commandError(cmd, jsonOut, invalidInput(err.Error()))
			}
			ctx, err := cli.commandContext()
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			req := &contract.RunRequest{
				CapabilityID:   capabilityID,
				BindingID:      bindingID,
				Inputs:         inputs,
				ProviderID:     providerID,
				Strategy:       contract.RunStrategy(strategy),
				Verify:         verify,
				MinVerifyLevel: minLevel,
			}
			if stream {
				renderer := newStreamRenderer(cmd, jsonOut)
				response, err := ctx.client.RunStream(cmd.Context(), req, renderer.Handle)
				if err != nil {
					return commandError(cmd, jsonOut && !renderer.TerminalSeen(), err)
				}
				if jsonOut {
					return nil
				}
				return render(cmd, RenderOptions{Mode: RenderText}, response, "run created")
			}
			response, err := ctx.client.Run(cmd.Context(), req)
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			return render(cmd, RenderOptions{Mode: renderMode(jsonOut)}, response, "run created")
		},
	}
	cmd.Flags().StringVar(&capabilityID, flagCapabilityID, "", "capability id")
	cmd.Flags().StringVar(&bindingID, flagBindingID, "", "binding id")
	cmd.Flags().StringVar(&providerID, flagProviderID, "", "provider id filter")
	cmd.Flags().StringVar(&inputsJSON, flagInputsJSON, "", "JSON object inputs")
	cmd.Flags().StringVar(&minVerifyLevel, flagMinVerify, "", "minimum verify level")
	cmd.Flags().StringVar(&strategy, flagStrategy, string(contract.RunStrategyDefault), "binding selection strategy")
	cmd.Flags().BoolVar(&stream, flagStream, false, "stream progress events")
	cmd.Flags().BoolVar(&verify, flagVerify, false, "verify run output")
	cmd.Flags().BoolVar(&jsonOut, flagJSON, false, "render machine-readable JSON")
	return cmd
}
