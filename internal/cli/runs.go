package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/cald/control"
)

func newRunsCommand(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runs",
		Short: "Create and inspect capability runs",
	}
	cmd.AddCommand(newRunsCreateCommand(cfg))
	cmd.AddCommand(newRunsGetCommand(cfg))
	return cmd
}

func newRunsCreateCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	var inputsJSON string
	var inputsFile string
	var capabilityID string
	var bindingID string
	var providerID string
	var strategy string
	var verifyRun bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Execute a promoted capability",
		RunE: func(cmd *cobra.Command, _ []string) error {
			inputs, err := readRunInputs(inputsJSON, inputsFile)
			if err != nil {
				return writeCommandError(cmd, jsonOut, newCommandError(commandErrorInvalidRunInput, err.Error()))
			}
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			run, err := client.Run(cmd.Context(), control.RunRequest{
				CapabilityID: capabilityID,
				BindingID:    bindingID,
				Inputs:       inputs,
				ProviderID:   providerID,
				Strategy:     strategy,
				Verify:       verifyRun,
			})
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			failed := run.Status == "failed"
			failureCode := ""
			if run.Error != nil {
				failureCode = run.Error.Code
			}
			if jsonOut {
				if err := writeJSON(cmd.OutOrStdout(), run); err != nil {
					return err
				}
				return runExitError(failed, failureCode)
			}
			if _, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", run.Status, run.BindingID); err != nil {
				return err
			}
			return runExitError(failed, failureCode)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	cmd.Flags().StringVar(&capabilityID, "capability-id", "", "capability_id request field")
	cmd.Flags().StringVar(&bindingID, "binding-id", "", "binding_id request field")
	cmd.Flags().StringVar(&inputsJSON, "inputs-json", "", "inputs request object as JSON")
	cmd.Flags().StringVar(&inputsFile, "inputs-file", "", "path to a JSON file containing the inputs request object")
	cmd.Flags().StringVar(&providerID, "provider-id", "", "provider_id request field")
	cmd.Flags().StringVar(&strategy, "strategy", "default", "binding resolution strategy")
	cmd.Flags().BoolVar(&verifyRun, "verify", false, "verify the outcome after execution")
	return cmd
}

func newRunsGetCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	var runID string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get one stored run",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if runID == "" {
				return writeCommandError(cmd, jsonOut, newCommandError(commandErrorInvalidRunInput, "run_id is required"))
			}
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			run, err := client.GetRun(cmd.Context(), runID)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), run)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", run.ID, run.Status)
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	cmd.Flags().StringVar(&runID, "run-id", "", "run_id path parameter")
	return cmd
}

func readRunInputs(withJSON, withFile string) (map[string]any, error) {
	if withJSON == "" && withFile == "" {
		return nil, fmt.Errorf("supply exactly one of --inputs-json or --inputs-file")
	}
	if withJSON != "" && withFile != "" {
		return nil, fmt.Errorf("supply exactly one of --inputs-json or --inputs-file")
	}
	payload := []byte(withJSON)
	if withFile != "" {
		content, err := os.ReadFile(withFile)
		if err != nil {
			return nil, fmt.Errorf("read input file: %w", err)
		}
		payload = content
	}

	var inputs map[string]any
	if err := json.Unmarshal(payload, &inputs); err != nil {
		return nil, fmt.Errorf("decode input JSON: %w", err)
	}
	if inputs == nil {
		return nil, fmt.Errorf("input JSON must be an object")
	}
	return inputs, nil
}

func readOptionalRunInputs(withJSON, withFile string) (map[string]any, error) {
	if withJSON == "" && withFile == "" {
		return map[string]any{}, nil
	}
	return readRunInputs(withJSON, withFile)
}

func runExitError(failed bool, code string) error {
	if !failed {
		return nil
	}
	if code == "" {
		return fmt.Errorf("run failed")
	}
	return fmt.Errorf("run failed: %s", code)
}
