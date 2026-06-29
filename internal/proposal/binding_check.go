package proposal

import (
	"path/filepath"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/runtime"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

const (
	bindingReasonMissingProbeMaterial     = "missing_probe_material"
	bindingReasonDuplicateProbeMaterial   = "duplicate_probe_material"
	bindingReasonChangedCapabilityID      = "changed_capability_id"
	bindingReasonInvalidProviderID        = "invalid_provider_id"
	bindingReasonMissingDescription       = "missing_description"
	bindingReasonUnsupportedExecutionKind = "unsupported_execution_kind"
	bindingReasonInvalidCLIArgs           = "invalid_cli_args"
	bindingReasonProviderExecutableInArgs = "provider_executable_in_args"
	bindingReasonMissingProbeInput        = "missing_probe_input"
	bindingReasonUnknownInputConstraint   = "unknown_input_constraint"
	bindingReasonCandidateLimit           = "candidate_limit"
)

func bindingCandidateSkipReason(req Request, capability capabilityPlan, candidate caltrace.Candidate, material probeMaterial) string {
	if candidate.ProviderID != "" && candidate.ProviderID != req.Provider.ID {
		return bindingReasonInvalidProviderID
	}
	if candidate.CapabilityID != "" && candidate.CapabilityID != capability.CapabilityID {
		return bindingReasonChangedCapabilityID
	}
	if candidate.Description == "" && strings.TrimSpace(capability.Description) == "" {
		return bindingReasonMissingDescription
	}
	if candidate.Execution.Kind != core.ExecutionKindCLI {
		return bindingReasonUnsupportedExecutionKind
	}
	args, ok := cliExecutionArgs(candidate.Execution)
	if !ok || len(args) == 0 {
		return bindingReasonInvalidCLIArgs
	}
	if argsIncludeProviderExecutable(args, req.Provider.Path) {
		return bindingReasonProviderExecutableInArgs
	}
	required, err := runtime.NewRunner(runtime.DefaultRegistry()).RequiredInputs(candidate.Execution)
	if err != nil {
		return bindingReasonInvalidCLIArgs
	}
	available := probeInputSet(material)
	for _, input := range required {
		if _, ok := available[input]; !ok {
			return bindingReasonWithDetail(bindingReasonMissingProbeInput, input)
		}
	}
	requiredSet := stringSet(required)
	for input := range candidate.InputConstraints {
		if _, ok := requiredSet[input]; !ok {
			return bindingReasonWithDetail(bindingReasonUnknownInputConstraint, input)
		}
	}
	return ""
}

func bindingReasonWithDetail(reason, detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return reason
	}
	return reason + ":" + detail
}

func cliExecutionArgs(execution core.Execution) ([]string, bool) {
	value, ok := execution.Spec[core.ExecutionSpecArgs]
	if !ok {
		return nil, false
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), true
	case []any:
		args := make([]string, len(typed))
		for index, item := range typed {
			arg, ok := item.(string)
			if !ok {
				return nil, false
			}
			args[index] = arg
		}
		return args, true
	default:
		return nil, false
	}
}

func argsIncludeProviderExecutable(args []string, providerPath string) bool {
	providerPath = strings.TrimSpace(providerPath)
	providerName := strings.TrimSpace(filepath.Base(providerPath))
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if providerPath != "" && arg == providerPath {
			return true
		}
		if providerName != "" && providerName != "." && arg == providerName {
			return true
		}
	}
	return false
}

func probeInputSet(material probeMaterial) map[string]struct{} {
	inputs := make(map[string]struct{}, len(material.Inputs)+len(material.Fixtures))
	for key := range material.Inputs {
		key = strings.TrimSpace(key)
		if key != "" {
			inputs[key] = struct{}{}
		}
	}
	for _, fixture := range material.Fixtures {
		input := strings.TrimSpace(fixture.Input)
		if input != "" {
			inputs[input] = struct{}{}
		}
	}
	return inputs
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}
