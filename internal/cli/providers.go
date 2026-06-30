package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/core"
)

type providersOutput struct {
	Providers []core.Provider `json:"providers"`
}

func newProvidersCommand(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Inspect provider records",
	}
	cmd.AddCommand(newProvidersAddCommand(cfg))
	cmd.AddCommand(newProvidersListCommand(cfg))
	cmd.AddCommand(newProvidersGetCommand(cfg))
	return cmd
}

func newProvidersAddCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	var providerPath string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Register one provider path",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if providerPath == "" {
				return writeCommandError(cmd, jsonOut, newCommandError(commandErrorInvalidProviderInput, "provider_path is required"))
			}
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			provider, err := client.AddProvider(cmd.Context(), providerPath)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), provider)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", provider.ID, provider.Kind, provider.Name)
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	cmd.Flags().StringVar(&providerPath, "provider-path", "", "provider_path request field")
	return cmd
}

func newProvidersListCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored providers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			providers, err := client.ListProviders(cmd.Context())
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), providersOutput{Providers: providers})
			}
			for _, provider := range providers {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", provider.ID, provider.Kind, provider.Name); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	return cmd
}

func newProvidersGetCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	var providerID string
	var providerPath string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get one stored provider",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if (providerID == "") == (providerPath == "") {
				return writeCommandError(cmd, jsonOut, newCommandError(commandErrorInvalidProviderInput, "supply exactly one of provider_id or provider_path"))
			}
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			var provider core.Provider
			if providerPath != "" {
				provider, err = client.GetProviderByPath(cmd.Context(), providerPath)
			} else {
				provider, err = client.GetProvider(cmd.Context(), providerID)
			}
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), provider)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", provider.ID, provider.Kind, provider.Name)
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	cmd.Flags().StringVar(&providerID, "provider-id", "", "provider_id path parameter")
	cmd.Flags().StringVar(&providerPath, "provider-path", "", "provider_path query parameter")
	return cmd
}
