package control

import (
	"context"
	"fmt"
	"strings"

	baselinerules "github.com/spacehz-lab/cal/internal/baseline/rules"
	"github.com/spacehz-lab/cal/internal/discovery"
	sharedllm "github.com/spacehz-lab/cal/internal/llm"
	observecli "github.com/spacehz-lab/cal/internal/observe/cli"
	"github.com/spacehz-lab/cal/internal/proposal"
	proposalllm "github.com/spacehz-lab/cal/internal/proposal/llm"
)

const (
	discoveryModeLLM   = "llm"
	discoveryModeRules = "rules"
)

// DiscoveryRequest identifies one synchronous discovery acquisition target.
type DiscoveryRequest struct {
	ProviderID   string `json:"provider_id,omitempty"`
	ProviderPath string `json:"provider_path,omitempty"`
	CapabilityID string `json:"capability_id,omitempty"`
	ProposalPath string `json:"proposal_path,omitempty"`
	Mode         string `json:"mode,omitempty"`
}

// Discover runs synchronous targeted acquisition for one provider or path.
func (svc Service) Discover(ctx context.Context, req DiscoveryRequest) (discovery.JobResult, error) {
	hasProvider := strings.TrimSpace(req.ProviderID) != ""
	hasProviderPath := strings.TrimSpace(req.ProviderPath) != ""
	if hasProvider == hasProviderPath {
		return discovery.JobResult{}, NewAPIError("invalid_discovery_target", "supply exactly one of provider_id or provider_path")
	}
	if hasProviderPath {
		providers, err := discovery.ScanEntries(ctx, discovery.EntryOptions{Entries: []string{req.ProviderPath}})
		if err != nil {
			return discovery.JobResult{}, err
		}
		if len(providers) == 0 {
			return discovery.JobResult{}, NewAPIError("target_provider_not_found", fmt.Sprintf("provider path %q did not resolve to a CLI executable or app bundle provider", req.ProviderPath))
		}
		if len(providers) > 1 {
			return discovery.JobResult{}, NewAPIError("ambiguous_target_provider", fmt.Sprintf("provider path %q resolved to %d providers", req.ProviderPath, len(providers)))
		}
		if _, _, err := svc.saveProviders(providers); err != nil {
			return discovery.JobResult{}, err
		}
		req.ProviderID = providers[0].ID
	}
	runner, err := svc.newAcquisitionRunner(req)
	if err != nil {
		return discovery.JobResult{}, err
	}
	result, err := runner.Run(ctx, svc.store, discovery.AcquisitionOptions{
		ProviderID:   req.ProviderID,
		CapabilityID: req.CapabilityID,
	})
	if err != nil {
		return discovery.JobResult{}, err
	}
	if hasProviderPath {
		result.Target.Type = discovery.TargetProviderPath
		result.Target.Value = req.ProviderPath
		result.Target.ProviderID = req.ProviderID
	}
	return result, nil
}

func (svc Service) newAcquisitionRunner(req DiscoveryRequest) (discovery.AcquisitionRunner, error) {
	if strings.TrimSpace(req.ProposalPath) != "" {
		replay, err := proposal.LoadFile(req.ProposalPath)
		if err != nil {
			return discovery.AcquisitionRunner{}, NewAPIError("invalid_discovery_proposal", err.Error())
		}
		return discovery.NewAcquisitionRunner(observecli.Observer{}, replay, replay), nil
	}
	switch discoveryMode(req.Mode) {
	case discoveryModeRules:
		return discovery.NewAcquisitionRunner(observecli.Observer{}, baselinerules.Proposer{}, baselinerules.NewProbePlanner()), nil
	case discoveryModeLLM:
	default:
		return discovery.AcquisitionRunner{}, NewAPIError("invalid_discovery_mode", fmt.Sprintf("unsupported discovery mode %q", req.Mode))
	}
	cfg, err := svc.cfg.Load()
	if err != nil {
		return discovery.AcquisitionRunner{}, err
	}
	llmConfig, err := cfg.RuntimeLLMConfig()
	if err != nil {
		return discovery.AcquisitionRunner{}, NewAPIError("invalid_llm_config", err.Error())
	}
	client, err := sharedllm.NewClient(llmConfig)
	if err != nil {
		return discovery.AcquisitionRunner{}, NewAPIError("invalid_llm_config", err.Error())
	}
	proposer := proposalllm.NewProposer(client)
	return discovery.NewAcquisitionRunner(observecli.Observer{}, proposer, proposer), nil
}

func discoveryMode(mode string) string {
	if strings.TrimSpace(mode) == "" {
		return discoveryModeLLM
	}
	return mode
}
