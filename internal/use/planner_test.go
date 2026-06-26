package use

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlannerFillsTemporaryTarget(t *testing.T) {
	now := time.Date(2026, 6, 26, 1, 2, 3, 0, time.UTC)
	resolution := Resolution{
		Selection: Selection{inputsPatch: map[string]any{"source": "input.txt"}},
		Required:  []string{"source", "target"},
	}

	inputs, err := NewPlanner("use_123", now).Plan(Request{}, resolution)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if inputs["source"] != "input.txt" {
		t.Fatalf("inputs = %#v, want patched source", inputs)
	}
	target, ok := inputs["target"].(string)
	if !ok || !strings.Contains(target, filepath.Join("cal", "artifacts", "2026-06-26", "use_123.out")) {
		t.Fatalf("target = %#v, want dated temporary artifact path", inputs["target"])
	}
}

func TestPlannerReportsMissingBusinessInputs(t *testing.T) {
	_, err := NewPlanner("use_123", time.Now()).Plan(Request{}, Resolution{Required: []string{"source", "target"}})
	if err == nil || err.Code != CodeMissingInputs || !strings.Contains(err.Message, "source") {
		t.Fatalf("Plan() error = %#v, want missing source", err)
	}
}

func TestPlannerRejectsPatchOverride(t *testing.T) {
	_, err := NewPlanner("use_123", time.Now()).Plan(Request{
		Inputs: map[string]any{"source": "caller.txt"},
	}, Resolution{
		Selection: Selection{inputsPatch: map[string]any{"source": "llm.txt"}},
		Required:  []string{"source"},
	})
	if err == nil || err.Code != CodeInvalidLLMSelection {
		t.Fatalf("Plan() error = %#v, want invalid llm selection", err)
	}
}

func TestPlannerDoesNotFillTargetWhenUnneeded(t *testing.T) {
	inputs, err := NewPlanner("use_123", time.Now()).Plan(Request{}, Resolution{})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if _, ok := inputs["target"]; ok {
		t.Fatalf("inputs = %#v, want no target", inputs)
	}
}
