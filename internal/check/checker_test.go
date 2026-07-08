package check

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestCheckerRulesExposeSupportedContract(t *testing.T) {
	rules := NewChecker().Rules()
	if len(rules) != 4 {
		t.Fatalf("Rules() len = %d, want 4", len(rules))
	}

	want := map[model.VerifySubjectType][]model.VerifyPredicate{
		model.VerifySubjectFile: {
			model.VerifyPredicateExists,
			model.VerifyPredicateNonEmpty,
			model.VerifyPredicateFormat,
			model.VerifyPredicateContains,
			model.VerifyPredicateContainsAny,
			model.VerifyPredicateRegex,
			model.VerifyPredicateBytesEqualTransform,
			model.VerifyPredicateHashLineMatches,
			model.VerifyPredicateArchiveContainsInput,
			model.VerifyPredicateJSONQueryMatches,
		},
		model.VerifySubjectStdout: {
			model.VerifyPredicateEquals,
			model.VerifyPredicateNotEquals,
			model.VerifyPredicateNonEmpty,
			model.VerifyPredicateContains,
			model.VerifyPredicateContainsAny,
			model.VerifyPredicateRegex,
			model.VerifyPredicateHashLineMatches,
			model.VerifyPredicateJSONQueryMatches,
		},
		model.VerifySubjectStderr: {
			model.VerifyPredicateEquals,
			model.VerifyPredicateNotEquals,
			model.VerifyPredicateNonEmpty,
			model.VerifyPredicateContains,
			model.VerifyPredicateContainsAny,
			model.VerifyPredicateRegex,
			model.VerifyPredicateHashLineMatches,
			model.VerifyPredicateJSONQueryMatches,
		},
		model.VerifySubjectExitCode: {
			model.VerifyPredicateEquals,
			model.VerifyPredicateNotEquals,
		},
	}

	for _, rule := range rules {
		if !reflect.DeepEqual(rule.AllowedPredicates, want[rule.Type]) {
			t.Fatalf("Rules()[%s] predicates = %#v, want %#v", rule.Type, rule.AllowedPredicates, want[rule.Type])
		}
	}
}

func TestCheckerValidateRejectsInvalidVerifySpecs(t *testing.T) {
	checker := NewChecker()
	tests := []struct {
		name string
		spec model.VerifySpec
	}{
		{
			name: "contract above L1",
			spec: model.VerifySpec{Level: model.VerifyLevelL2, Method: model.VerifyMethodContract},
		},
		{
			name: "execute missing checks",
			spec: model.VerifySpec{Level: model.VerifyLevelL1, Method: model.VerifyMethodExecute},
		},
		{
			name: "missing predicate params",
			spec: model.VerifySpec{
				Level:  model.VerifyLevelL2,
				Method: model.VerifyMethodExecute,
				Checks: []model.VerifyCheck{{Subject: model.VerifySubject{Type: model.VerifySubjectExitCode}, Predicate: model.VerifyPredicateEquals}},
			},
		},
		{
			name: "invalid regex",
			spec: model.VerifySpec{
				Level:  model.VerifyLevelL2,
				Method: model.VerifyMethodExecute,
				Checks: []model.VerifyCheck{{Subject: model.VerifySubject{Type: model.VerifySubjectStdout}, Predicate: model.VerifyPredicateRegex, Params: map[string]any{paramPattern: "["}}},
			},
		},
		{
			name: "unsupported param value",
			spec: model.VerifySpec{
				Level:  model.VerifyLevelL3,
				Method: model.VerifyMethodExecute,
				Checks: []model.VerifyCheck{{
					Subject:   model.VerifySubject{Type: model.VerifySubjectFile, Input: "target"},
					Predicate: model.VerifyPredicateBytesEqualTransform,
					Params:    map[string]any{paramSource: "source", paramTransform: "identity"},
				}},
			},
		},
		{
			name: "invalid subject predicate",
			spec: model.VerifySpec{
				Level:  model.VerifyLevelL2,
				Method: model.VerifyMethodExecute,
				Checks: []model.VerifyCheck{{
					Subject:   model.VerifySubject{Type: model.VerifySubjectFile, Input: "target"},
					Predicate: model.VerifyPredicateEquals,
					Params:    map[string]any{paramValue: "content"},
				}},
			},
		},
		{
			name: "missing file input",
			spec: model.VerifySpec{
				Level:  model.VerifyLevelL2,
				Method: model.VerifyMethodExecute,
				Checks: []model.VerifyCheck{{Subject: model.VerifySubject{Type: model.VerifySubjectFile}, Predicate: model.VerifyPredicateExists}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := checker.Validate(&test.spec); err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
		})
	}
}

func TestCheckerValidateAllowsSupportedVerifySpecs(t *testing.T) {
	checker := NewChecker()
	specs := []model.VerifySpec{
		{
			Level:  model.VerifyLevelL1,
			Method: model.VerifyMethodContract,
			Checks: []model.VerifyCheck{{Subject: model.VerifySubject{Type: model.VerifySubjectStdout}, Predicate: model.VerifyPredicateNonEmpty}},
		},
		{
			Level:  model.VerifyLevelL0,
			Method: model.VerifyMethodExecute,
		},
		{
			Level:  model.VerifyLevelL2,
			Method: model.VerifyMethodExecute,
			Checks: []model.VerifyCheck{{Subject: model.VerifySubject{Type: model.VerifySubjectStderr}, Predicate: model.VerifyPredicateEquals, Params: map[string]any{paramValue: ""}}},
		},
	}

	for _, spec := range specs {
		if err := checker.Validate(&spec); err != nil {
			t.Fatalf("Validate(%#v) error = %v", spec, err)
		}
	}
}

func TestCheckerRunRejectsNonExecutableSpecs(t *testing.T) {
	checker := NewChecker()
	contractSpec := model.VerifySpec{Level: model.VerifyLevelL1, Method: model.VerifyMethodContract}
	if _, err := checker.Run(context.Background(), &Request{Spec: &contractSpec}); err == nil {
		t.Fatal("Run(contract) error = nil, want error")
	}

	l0Spec := model.VerifySpec{Level: model.VerifyLevelL0, Method: model.VerifyMethodExecute}
	if _, err := checker.Run(context.Background(), &Request{Spec: &l0Spec}); err == nil {
		t.Fatal("Run(L0 execute) error = nil, want error")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	spec := model.VerifySpec{Level: model.VerifyLevelL2, Method: model.VerifyMethodExecute}
	if _, err := checker.Run(ctx, &Request{Spec: &spec}); err == nil {
		t.Fatal("Run(canceled) error = nil, want error")
	}
}

func TestCheckerRunReturnsEvidenceAndOutputs(t *testing.T) {
	target := writeTempFile(t, "target.txt", "ready\n")
	spec := model.VerifySpec{
		Level:  model.VerifyLevelL2,
		Method: model.VerifyMethodExecute,
		Checks: []model.VerifyCheck{
			{Subject: model.VerifySubject{Type: model.VerifySubjectFile, Input: "target"}, Predicate: model.VerifyPredicateExists},
			{Subject: model.VerifySubject{Type: model.VerifySubjectStdout}, Predicate: model.VerifyPredicateContains, Params: map[string]any{paramValue: "ok"}},
			{Subject: model.VerifySubject{Type: model.VerifySubjectStderr}, Predicate: model.VerifyPredicateNonEmpty},
			{Subject: model.VerifySubject{Type: model.VerifySubjectExitCode}, Predicate: model.VerifyPredicateEquals, Params: map[string]any{paramValue: 0}},
		},
	}

	result, err := NewChecker().Run(context.Background(), &Request{
		Spec:     &spec,
		Inputs:   map[string]any{"target": target},
		Stdout:   "ok\n",
		Stderr:   "warning\n",
		ExitCode: 0,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Evidence) != 4 {
		t.Fatalf("Evidence len = %d, want 4", len(result.Evidence))
	}
	if !strings.HasPrefix(result.Evidence[0].ID, "check_1_target_exists") {
		t.Fatalf("Evidence[0].ID = %q", result.Evidence[0].ID)
	}
	if result.Outputs["target"] != target || result.Outputs[string(model.VerifySubjectStdout)] != "ok\n" || result.Outputs[string(model.VerifySubjectStderr)] != "warning\n" || result.Outputs[string(model.VerifySubjectExitCode)] != 0 {
		t.Fatalf("Outputs = %#v", result.Outputs)
	}
}
