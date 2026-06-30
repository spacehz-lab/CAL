package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/core"
	caluse "github.com/spacehz-lab/cal/internal/use"
)

func newUseCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	var intent string
	var inputsJSON string
	var inputsFile string
	var providerID string
	var strategy string
	var verifyRun bool
	var minVerifyLevel string
	cmd := &cobra.Command{
		Use:   "use [intent]",
		Short: "Route an intent to a promoted capability",
		RunE: func(cmd *cobra.Command, args []string) error {
			if intent != "" && len(args) > 0 {
				return writeCommandError(cmd, jsonOut, newCommandError(commandErrorInvalidUseInput, "supply intent either as an argument or --intent, not both"))
			}
			if intent == "" {
				intent = strings.Join(args, " ")
			}
			inputs, err := readOptionalRunInputs(inputsJSON, inputsFile)
			if err != nil {
				return writeCommandError(cmd, jsonOut, newCommandError(commandErrorInvalidUseInput, err.Error()))
			}
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			result, err := client.Use(cmd.Context(), caluse.Request{
				Intent:         intent,
				Inputs:         inputs,
				ProviderID:     providerID,
				Strategy:       strategy,
				Verify:         verifyRun,
				MinVerifyLevel: core.VerifyLevel(minVerifyLevel),
			})
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			failed := result.Status == "failed"
			failureCode := ""
			if result.Error != nil {
				failureCode = result.Error.Code
			}
			if jsonOut {
				if err := writeJSON(cmd.OutOrStdout(), result); err != nil {
					return err
				}
				return runExitError(failed, failureCode)
			}
			if result.Selection == nil || result.Run == nil {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", result.Status)
				return runExitError(failed, failureCode)
			}
			if _, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", result.Status, result.Selection.CapabilityID, result.Selection.BindingID, result.Run.ID); err != nil {
				return err
			}
			return runExitError(failed, failureCode)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	cmd.Flags().StringVar(&intent, "intent", "", "intent request field")
	cmd.Flags().StringVar(&inputsJSON, "inputs-json", "", "inputs request object as JSON")
	cmd.Flags().StringVar(&inputsFile, "inputs-file", "", "path to a JSON file containing the inputs request object")
	cmd.Flags().StringVar(&providerID, "provider-id", "", "provider_id request field")
	cmd.Flags().StringVar(&strategy, "strategy", "default", "binding resolution strategy")
	cmd.Flags().BoolVar(&verifyRun, "verify", false, "verify the outcome after execution")
	cmd.Flags().StringVar(&minVerifyLevel, "min-verify-level", "", "minimum verification level: L1, L2, or L3")
	return cmd
}
