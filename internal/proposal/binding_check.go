package proposal

import (
	"fmt"
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
	bindingReasonMissingCLIArgs           = "missing_cli_args"
	bindingReasonInvalidCLIArgsType       = "invalid_cli_args_type"
	bindingReasonInvalidCLIArgItem        = "invalid_cli_arg_item"
	bindingReasonEmptyCLIArgs             = "empty_cli_args"
	bindingReasonInvalidCLIInputTemplate  = "invalid_cli_input_template"
	bindingReasonProviderExecutableInArgs = "provider_executable_in_args"
	bindingReasonMissingProbeInput        = "missing_probe_input"
	bindingReasonUnknownInputConstraint   = "unknown_input_constraint"
	bindingReasonInvalidInputConstraint   = "invalid_input_constraint"
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
	args, reason := cliExecutionArgsOrReason(candidate.Execution)
	if reason != "" {
		return reason
	}
	if argsIncludeProviderExecutable(args, req.Provider.Path) {
		return bindingReasonProviderExecutableInArgs
	}
	required, err := runtime.NewRunner(runtime.DefaultRegistry()).RequiredInputs(candidate.Execution)
	if err != nil {
		return bindingReasonWithDetail(bindingReasonInvalidCLIInputTemplate, err.Error())
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
		if !validInputConstraint(candidate.InputConstraints[input]) {
			return bindingReasonWithDetail(bindingReasonInvalidInputConstraint, input)
		}
	}
	return ""
}

func validInputConstraint(constraint any) bool {
	fields, ok := constraint.(map[string]any)
	if !ok {
		return false
	}
	if enum, ok := fields["enum"]; ok {
		switch enum.(type) {
		case []any, []string:
			return true
		default:
			return false
		}
	}
	return true
}

func bindingReasonWithDetail(reason, detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return reason
	}
	return reason + ":" + detail
}

func cliExecutionArgs(execution core.Execution) ([]string, bool) {
	args, reason := cliExecutionArgsOrReason(execution)
	return args, reason == ""
}

func cliExecutionArgsOrReason(execution core.Execution) ([]string, string) {
	value, ok := execution.Spec[core.ExecutionSpecArgs]
	if !ok {
		return nil, bindingReasonMissingCLIArgs
	}
	switch typed := value.(type) {
	case []string:
		if len(typed) == 0 {
			return nil, bindingReasonEmptyCLIArgs
		}
		return append([]string(nil), typed...), ""
	case []any:
		if len(typed) == 0 {
			return nil, bindingReasonEmptyCLIArgs
		}
		args := make([]string, len(typed))
		for index, item := range typed {
			arg, ok := item.(string)
			if !ok {
				return nil, bindingReasonWithDetail(bindingReasonInvalidCLIArgItem, fmt.Sprintf("%d:%T", index, item))
			}
			args[index] = arg
		}
		return args, ""
	default:
		return nil, bindingReasonWithDetail(bindingReasonInvalidCLIArgsType, fmt.Sprintf("%T", value))
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
