package cli

import (
	"fmt"

	caldclient "github.com/spacehz-lab/cal/internal/cald/client"
	"github.com/spf13/cobra"
)

func newCapabilitiesCommand(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capabilities",
		Short: "List acquired capabilities",
	}
	cmd.AddCommand(newCapabilitiesListCommand(cfg))
	cmd.AddCommand(newCapabilitiesGetCommand(cfg))
	return cmd
}

func newCapabilitiesListCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	var providerID string
	var capabilityID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored capabilities",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			list, err := client.ListCapabilities(cmd.Context(), caldclient.CapabilityListOptions{
				CapabilityID: capabilityID,
				ProviderID:   providerID,
			})
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), list)
			}
			if list.Count == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "no capabilities")
				return err
			}
			for _, capability := range list.Capabilities {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), capability.ID); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	cmd.Flags().StringVar(&providerID, "provider-id", "", "provider_id query parameter")
	cmd.Flags().StringVar(&capabilityID, "capability-id", "", "capability_id query parameter")
	return cmd
}

func newCapabilitiesGetCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	var capabilityID string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get one stored capability",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if capabilityID == "" {
				return writeCommandError(cmd, jsonOut, newCommandError(commandErrorInvalidCapabilityInput, "capability_id is required"))
			}
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			capability, err := client.GetCapability(cmd.Context(), capabilityID)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), capability)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), capability.ID)
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	cmd.Flags().StringVar(&capabilityID, "capability-id", "", "capability_id path parameter")
	return cmd
}
