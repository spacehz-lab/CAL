package plan

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunCopiesInputsAndMergesPatch(t *testing.T) {
	inputs := map[string]any{"source": "input.md"}
	result, err := NewRunner().Run(&Request{
		UseID:          "use_test",
		Inputs:         inputs,
		RequiredInputs: []string{"source", "format"},
		InputsPatch:    map[string]any{"format": "pdf"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Inputs["source"] != "input.md" || result.Inputs["format"] != "pdf" {
		t.Fatalf("inputs = %#v, want source and format", result.Inputs)
	}
	inputs["source"] = "changed.md"
	if result.Inputs["source"] != "input.md" {
		t.Fatalf("planned inputs changed after caller mutation: %#v", result.Inputs)
	}
}

func TestRunRejectsPatchOverwrite(t *testing.T) {
	_, err := NewRunner().Run(&Request{
		Inputs:      map[string]any{"source": "input.md"},
		InputsPatch: map[string]any{"source": "other.md"},
	})
	if !errors.Is(err, ErrInvalidInputsPatch) {
		t.Fatalf("Run() error = %v, want ErrInvalidInputsPatch", err)
	}
}

func TestRunGeneratesMissingTarget(t *testing.T) {
	now := time.Date(2026, 7, 4, 1, 2, 3, 0, time.UTC)
	result, err := NewRunner().Run(&Request{
		UseID:          "use_123",
		Now:            now,
		RequiredInputs: []string{"target"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	target, ok := result.Inputs[InputTarget].(string)
	if !ok || !strings.HasSuffix(target, filepath.Join("2026-07-04", "use_123.out")) {
		t.Fatalf("target = %#v, want generated path", result.Inputs[InputTarget])
	}
}

func TestRunReportsMissingInputs(t *testing.T) {
	_, err := NewRunner().Run(&Request{RequiredInputs: []string{"source"}})
	if !errors.Is(err, ErrMissingInputs) {
		t.Fatalf("Run() error = %v, want ErrMissingInputs", err)
	}
}
