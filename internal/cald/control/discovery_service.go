package control

import (
	"context"
	"fmt"
	"strings"

	baselinerules "github.com/spacehz-lab/cal/internal/baseline/rules"
	"github.com/spacehz-lab/cal/internal/discovery"
	observecli "github.com/spacehz-lab/cal/internal/observe/cli"
	"github.com/spacehz-lab/cal/internal/proposalflow"
)

const (
	discoveryModeLLM   = "llm"
	discoveryModeRules = "rules"
)

// DiscoveryRequest identifies one synchronous discovery acquisition target.
type DiscoveryRequest struct {
	ProviderID   string `json:"provider_id,omitempty"`
	CapabilityID string `json:"capability_id,omitempty"`
	ProposalPath string `json:"proposal_path,omitempty"`
	Mode         string `json:"mode,omitempty"`
}

// Discover runs synchronous targeted acquisition for one stored provider.
func (svc Service) Discover(ctx context.Context, req DiscoveryRequest) (discovery.JobResult, error) {
	if strings.TrimSpace(req.ProviderID) == "" {
		return discovery.JobResult{}, NewAPIError("invalid_discovery_target", "provider_id is required")
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
	return result, nil
}

func (svc Service) newAcquisitionRunner(req DiscoveryRequest) (discovery.AcquisitionRunner, error) {
	if strings.TrimSpace(req.ProposalPath) != "" {
		replay, err := proposalflow.LoadReplayFile(req.ProposalPath)
		if err != nil {
			return discovery.AcquisitionRunner{}, NewAPIError("invalid_discovery_proposal", err.Error())
		}
		return discovery.NewAcquisitionRunner(observecli.Observer{}, replay), nil
	}
	switch discoveryMode(req.Mode) {
	case discoveryModeRules:
		return discovery.NewAcquisitionRunner(observecli.Observer{}, baselinerules.Proposer{}), nil
	case discoveryModeLLM:
		return discovery.AcquisitionRunner{}, NewAPIError("unsupported_discovery_mode", "llm discovery has not been migrated to Proposal")
	default:
		return discovery.AcquisitionRunner{}, NewAPIError("invalid_discovery_mode", fmt.Sprintf("unsupported discovery mode %q", req.Mode))
	}
}

func discoveryMode(mode string) string {
	if strings.TrimSpace(mode) == "" {
		return discoveryModeLLM
	}
	return mode
}
