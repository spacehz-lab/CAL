package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/cald/control"
)

type discoveryRunOptions struct {
	jsonOut      bool
	providerID   string
	capabilityID string
	proposalPath string
	mode         string
}

func newDiscoveryCommand(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discovery",
		Short: "Run capability discovery acquisition",
	}
	cmd.AddCommand(newDiscoveryRunCommand(cfg))
	return cmd
}

func newDiscoveryRunCommand(cfg Config) *cobra.Command {
	opts := discoveryRunOptions{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run discovery acquisition for one provider",
		RunE: func(cmd *cobra.Command, _ []string) error {
			req, err := buildDiscoveryRunRequest(opts)
			if err != nil {
				return writeCommandError(cmd, opts.jsonOut, err)
			}
			client, err := newCaldClient(cfg)
			if err != nil {
				return writeCommandError(cmd, opts.jsonOut, err)
			}
			result, err := client.Discover(cmd.Context(), req)
			if err != nil {
				return writeCommandError(cmd, opts.jsonOut, err)
			}
			if opts.jsonOut {
				return writeJSON(cmd.OutOrStdout(), result)
			}
			for _, provider := range result.Providers {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", provider.ID, provider.Name); err != nil {
					return err
				}
			}
			return nil
		},
	}
	opts.bind(cmd)
	return cmd
}

func (opts *discoveryRunOptions) bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "render machine-readable JSON")
	cmd.Flags().StringVar(&opts.providerID, "provider-id", "", "provider_id request field")
	cmd.Flags().StringVar(&opts.capabilityID, "capability-id", "", "optional capability_id debug filter")
	cmd.Flags().StringVar(&opts.proposalPath, "proposal-path", "", "proposal_path replay file")
	cmd.Flags().StringVar(&opts.mode, "mode", "llm", "candidate proposal mode: llm or rules")
	_ = cmd.Flags().MarkHidden("mode")
}

func buildDiscoveryRunRequest(opts discoveryRunOptions) (control.DiscoveryRequest, error) {
	if opts.providerID == "" {
		return control.DiscoveryRequest{}, newCommandError(commandErrorInvalidDiscoveryTarget, "provider_id is required")
	}
	return control.DiscoveryRequest{
		ProviderID:   opts.providerID,
		CapabilityID: opts.capabilityID,
		ProposalPath: opts.proposalPath,
		Mode:         opts.mode,
	}, nil
}
