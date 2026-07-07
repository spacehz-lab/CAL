package cli

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestParseInputsJSON(t *testing.T) {
	inputs, err := parseInputsJSON(`{"text":"ok","count":2}`)
	if err != nil {
		t.Fatalf("parseInputsJSON() error = %v", err)
	}
	if inputs["text"] != "ok" || inputs["count"].(float64) != 2 {
		t.Fatalf("inputs = %#v, want parsed object", inputs)
	}
}

func TestParseInputsJSONRejectsNonObject(t *testing.T) {
	if _, err := parseInputsJSON(`["bad"]`); err == nil {
		t.Fatal("parseInputsJSON() error = nil, want non-object error")
	}
}

func TestParseMinVerifyLevel(t *testing.T) {
	level, err := parseMinVerifyLevel(" L2 ")
	if err != nil {
		t.Fatalf("parseMinVerifyLevel() error = %v", err)
	}
	if level != model.VerifyLevelL2 {
		t.Fatalf("level = %q, want L2", level)
	}

	level, err = parseMinVerifyLevel("")
	if err != nil {
		t.Fatalf("parseMinVerifyLevel(empty) error = %v", err)
	}
	if level != "" {
		t.Fatalf("empty level = %q, want zero value", level)
	}
}

func TestParseMinVerifyLevelRejectsUnknownLevel(t *testing.T) {
	_, err := parseMinVerifyLevel("L9")
	if err == nil {
		t.Fatal("parseMinVerifyLevel() error = nil, want invalid level error")
	}
}
