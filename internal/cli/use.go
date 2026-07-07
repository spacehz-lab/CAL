package cli

import (
	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/contract"
)

func (cli *CLI) newUseCommand() *cobra.Command {
	var inputsJSON string
	var jsonOut bool
	var minVerifyLevel string
	var providerID string
	var strategy string
	var stream bool
	var verify bool
	cmd := &cobra.Command{
		Use:   "use <intent>",
		Short: "Execute an intent through reusable capabilities",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			req := &contract.UseRequest{
				Intent:         args[0],
				Inputs:         inputs,
				ProviderID:     providerID,
				Strategy:       contract.RunStrategy(strategy),
				Verify:         verify,
				MinVerifyLevel: minLevel,
			}
			if stream {
				renderer := newStreamRenderer(cmd, jsonOut)
				response, err := ctx.client.UseStream(cmd.Context(), req, renderer.Handle)
				if err != nil {
					return commandError(cmd, jsonOut && !renderer.TerminalSeen(), err)
				}
				if jsonOut {
					return nil
				}
				return render(cmd, RenderOptions{Mode: RenderText}, response, "use completed")
			}
			response, err := ctx.client.Use(cmd.Context(), req)
			if err != nil {
				return commandError(cmd, jsonOut, err)
			}
			return render(cmd, RenderOptions{Mode: renderMode(jsonOut)}, response, "use completed")
		},
	}
	cmd.Flags().StringVar(&providerID, flagProviderID, "", "provider id filter")
	cmd.Flags().StringVar(&inputsJSON, flagInputsJSON, "", "JSON object inputs")
	cmd.Flags().StringVar(&minVerifyLevel, flagMinVerify, "", "minimum verify level")
	cmd.Flags().StringVar(&strategy, flagStrategy, string(contract.RunStrategyDefault), "binding selection strategy")
	cmd.Flags().BoolVar(&stream, flagStream, false, "stream progress events")
	cmd.Flags().BoolVar(&verify, flagVerify, false, "verify run output")
	cmd.Flags().BoolVar(&jsonOut, flagJSON, false, "render machine-readable JSON")
	return cmd
}
