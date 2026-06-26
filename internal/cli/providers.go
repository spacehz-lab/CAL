package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/cald/control"
	"github.com/spacehz-lab/cal/internal/config"
	"github.com/spacehz-lab/cal/internal/core"
)

type providersOutput struct {
	Providers []core.Provider `json:"providers"`
}

func newProvidersCommand(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Inspect provider records and sources",
	}
	cmd.AddCommand(newProviderSourcesCommand(cfg))
	cmd.AddCommand(newProvidersFindCommand(cfg))
	cmd.AddCommand(newProvidersListCommand(cfg))
	cmd.AddCommand(newProvidersGetCommand(cfg))
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
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get one stored provider",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if providerID == "" {
				return writeCommandError(cmd, jsonOut, newCommandError(commandErrorInvalidProviderInput, "provider_id is required"))
			}
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			provider, err := client.GetProvider(cmd.Context(), providerID)
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
	return cmd
}

func newProvidersFindCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	var kind string
	cmd := &cobra.Command{
		Use:   "find",
		Short: "Find providers from configured sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			result, err := client.FindProviders(cmd.Context(), control.ProviderFindRequest{Kind: kind})
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), result)
			}
			for _, provider := range result.Providers {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", provider.ID, provider.Kind, provider.Name); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	cmd.Flags().StringVar(&kind, "kind", "", "provider kind filter")
	return cmd
}

type providerSourcesOutput struct {
	Sources []config.ProviderSource `json:"sources"`
}

type providerSourceMutationOutput struct {
	Changed bool                    `json:"changed"`
	Sources []config.ProviderSource `json:"sources"`
}

type providerSourceOptions struct {
	jsonOut bool
	kind    string
	value   string
}

func newProviderSourcesCommand(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sources",
		Short: "Manage provider sources",
	}
	cmd.AddCommand(newProviderSourcesListCommand(cfg))
	cmd.AddCommand(newProviderSourcesAddCommand(cfg))
	cmd.AddCommand(newProviderSourcesRemoveCommand(cfg))
	return cmd
}

func newProviderSourcesListCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List provider sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			sources, err := client.ListSources(cmd.Context())
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), providerSourcesOutput{Sources: sources})
			}
			for _, source := range sources {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", source.Kind, source.Value); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	return cmd
}

func newProviderSourcesAddCommand(cfg Config) *cobra.Command {
	opts := providerSourceOptions{}
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add one provider source",
		RunE: func(cmd *cobra.Command, _ []string) error {
			source, err := opts.source()
			if err != nil {
				return writeCommandError(cmd, opts.jsonOut, err)
			}
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, opts.jsonOut, err)
			}
			result, err := client.AddSource(cmd.Context(), source)
			if err != nil {
				return writeCommandError(cmd, opts.jsonOut, err)
			}
			if opts.jsonOut {
				return writeJSON(cmd.OutOrStdout(), providerSourceMutationOutput{Changed: result.Changed, Sources: result.Sources})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", source.Kind, source.Value)
			return err
		},
	}
	opts.bind(cmd)
	return cmd
}

func newProviderSourcesRemoveCommand(cfg Config) *cobra.Command {
	opts := providerSourceOptions{}
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove one provider source",
		RunE: func(cmd *cobra.Command, _ []string) error {
			source, err := opts.source()
			if err != nil {
				return writeCommandError(cmd, opts.jsonOut, err)
			}
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, opts.jsonOut, err)
			}
			result, err := client.RemoveSource(cmd.Context(), source)
			if err != nil {
				return writeCommandError(cmd, opts.jsonOut, err)
			}
			if opts.jsonOut {
				return writeJSON(cmd.OutOrStdout(), providerSourceMutationOutput{Changed: result.Changed, Sources: result.Sources})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", source.Kind, source.Value)
			return err
		},
	}
	opts.bind(cmd)
	return cmd
}

func (opts *providerSourceOptions) bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "render machine-readable JSON")
	cmd.Flags().StringVar(&opts.kind, "kind", "", "provider source kind")
	cmd.Flags().StringVar(&opts.value, "value", "", "provider source value")
}

func (opts providerSourceOptions) source() (config.ProviderSource, error) {
	if opts.kind == "" {
		return config.ProviderSource{}, newCommandError(commandErrorInvalidProviderInput, "kind is required")
	}
	if opts.value == "" {
		return config.ProviderSource{}, newCommandError(commandErrorInvalidProviderInput, "value is required")
	}
	return config.ProviderSource{Kind: config.ProviderSourceKind(opts.kind), Value: opts.value}, nil
}
