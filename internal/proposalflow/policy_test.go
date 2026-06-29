package proposalflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultPolicyValidates(t *testing.T) {
	if err := ValidatePolicy(DefaultPolicy()); err != nil {
		t.Fatalf("ValidatePolicy(DefaultPolicy()) error = %v", err)
	}
}

func TestEnsurePolicyFileWritesDefaultPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), PolicyFileName)
	policy, err := EnsurePolicyFile(path)
	if err != nil {
		t.Fatalf("EnsurePolicyFile() error = %v", err)
	}
	if len(policy.Surface.AllowedKinds) == 0 || len(policy.Surface.SkipNames) == 0 {
		t.Fatalf("policy = %#v, want default surface policy", policy)
	}
	loaded, err := LoadPolicyFile(path)
	if err != nil {
		t.Fatalf("LoadPolicyFile() error = %v", err)
	}
	if len(loaded.Surface.AllowedKinds) != len(policy.Surface.AllowedKinds) {
		t.Fatalf("loaded policy = %#v, want written default policy", loaded)
	}
}

func TestLoadPolicyFileReadsCompletePolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), PolicyFileName)
	if err := os.WriteFile(path, []byte(`{
  "surface": {
    "allowed_kinds": ["command", "option"],
    "skip_names": ["help"],
    "skip_patterns": ["^debug-"]
  },
  "capability": {
    "allowed_subjects": ["file"],
    "blocked_subjects": ["network"]
  }
}`), 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	policy, err := LoadPolicyFile(path)
	if err != nil {
		t.Fatalf("LoadPolicyFile() error = %v", err)
	}
	if len(policy.Surface.AllowedKinds) != 2 || policy.Surface.SkipPatterns[0] != "^debug-" {
		t.Fatalf("surface policy = %#v, want configured policy", policy.Surface)
	}
	if policy.Capability.AllowedSubjects[0] != "file" || policy.Capability.BlockedSubjects[0] != "network" {
		t.Fatalf("capability policy = %#v, want configured policy", policy.Capability)
	}
}

func TestLoadPolicyFileRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), PolicyFileName)
	if err := os.WriteFile(path, []byte(`{"surface":{"allowed_kinds":["command"]},"unexpected":true}`), 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	_, err := LoadPolicyFile(path)
	if err == nil || !strings.Contains(err.Error(), "decode proposal policy") {
		t.Fatalf("LoadPolicyFile() error = %v, want decode proposal policy error", err)
	}
}

func TestValidatePolicyRejectsInvalidSurfacePolicy(t *testing.T) {
	tests := []struct {
		name   string
		policy Policy
		want   string
	}{
		{
			name:   "empty allowed kinds",
			policy: Policy{Surface: SurfacePolicy{}},
			want:   "allowed_kinds",
		},
		{
			name: "unsupported kind",
			policy: Policy{Surface: SurfacePolicy{
				AllowedKinds: []string{"command", "widget"},
			}},
			want: "unsupported",
		},
		{
			name: "invalid regex",
			policy: Policy{Surface: SurfacePolicy{
				AllowedKinds: []string{"command"},
				SkipPatterns: []string{"["},
			}},
			want: "invalid proposal surface skip pattern",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidatePolicy(test.policy)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidatePolicy() error = %v, want %q", err, test.want)
			}
		})
	}
}
