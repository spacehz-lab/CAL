package control

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/config"
	"github.com/spacehz-lab/cal/internal/proposalflow"
)

func TestDiscoverRejectsInvalidTargetShape(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Discover(context.Background(), DiscoveryRequest{})
	if err == nil {
		t.Fatal("Discover() error = nil, want invalid target")
	}
	var apiErr APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "invalid_discovery_target" {
		t.Fatalf("Discover() error = %#v, want invalid_discovery_target", err)
	}
}

func TestNewAcquisitionRunnerBuildsDefaultLLMRunner(t *testing.T) {
	t.Setenv(config.EnvLLMAPI, config.LLMAPIChatCompletions)
	t.Setenv(config.EnvLLMModel, "test-model")
	t.Setenv(config.EnvLLMAPIKey, "test-key")
	svc := newTestService(t)

	_, err := svc.newAcquisitionRunner(DiscoveryRequest{})
	if err != nil {
		t.Fatalf("newAcquisitionRunner(default) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(svc.Home(), proposalflow.PolicyFileName)); err != nil {
		t.Fatalf("proposal policy was not written: %v", err)
	}
}

func TestNewAcquisitionRunnerRejectsInvalidProposalPolicy(t *testing.T) {
	t.Setenv(config.EnvLLMAPI, config.LLMAPIChatCompletions)
	t.Setenv(config.EnvLLMModel, "test-model")
	t.Setenv(config.EnvLLMAPIKey, "test-key")
	svc := newTestService(t)
	if err := os.WriteFile(filepath.Join(svc.Home(), proposalflow.PolicyFileName), []byte(`{
  "surface": {
    "allowed_kinds": ["widget"],
    "skip_names": [],
    "skip_patterns": []
  },
  "capability": {
    "allowed_subjects": [],
    "blocked_subjects": []
  }
}`), 0o644); err != nil {
		t.Fatalf("write proposal policy: %v", err)
	}

	_, err := svc.newAcquisitionRunner(DiscoveryRequest{})
	var apiErr APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "invalid_proposal_policy" {
		t.Fatalf("newAcquisitionRunner() error = %#v, want invalid_proposal_policy", err)
	}
}
