package proposal

import (
	"fmt"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestSelectExistingCapabilityIDsReturnsSmallCatalog(t *testing.T) {
	capabilities := []core.Capability{
		{ID: "image.resize"},
		{ID: "document.export_pdf"},
		{ID: "bad"},
	}

	selected := SelectExistingCapabilityIDs(nil, capabilities, "image.resize", 30)
	want := []string{"image.resize", "document.export_pdf"}
	assertIDs(t, selected, want)
}

func TestSelectExistingCapabilityIDsScoresObservationTokens(t *testing.T) {
	capabilities := largeCapabilityCatalog()
	capabilities = append(capabilities,
		core.Capability{ID: "document.export_pdf"},
		core.Capability{ID: "image.resize"},
	)
	observations := []caltrace.Observation{{
		Type:   "cli_output",
		Source: "help",
		Content: map[string]any{
			"text": "Usage: tool make-pdf --in <input> --out <output.pdf>\nConverts text to PDF.",
		},
	}}

	selected := SelectExistingCapabilityIDs(observations, capabilities, "", 3)
	if len(selected) == 0 || selected[0] != "document.export_pdf" {
		t.Fatalf("selected = %#v, want document.export_pdf first", selected)
	}
}

func TestSelectExistingCapabilityIDsSupportsTreeLikeObservationText(t *testing.T) {
	capabilities := largeCapabilityCatalog()
	capabilities = append(capabilities,
		core.Capability{ID: "message.send"},
		core.Capability{ID: "document.export_pdf"},
	)
	observations := []caltrace.Observation{{
		Type:   "ax_tree",
		Source: "main_window",
		Content: map[string]any{
			"role":  "button",
			"title": "Send message",
			"path":  "Window > Composer > Send",
		},
	}}

	selected := SelectExistingCapabilityIDs(observations, capabilities, "", 2)
	if len(selected) == 0 || selected[0] != "message.send" {
		t.Fatalf("selected = %#v, want message.send first", selected)
	}
}

func TestSelectExistingCapabilityIDsPrioritizesHintInLargeCatalog(t *testing.T) {
	capabilities := largeCapabilityCatalog()
	capabilities = append(capabilities,
		core.Capability{ID: "document.export_pdf"},
		core.Capability{ID: "image.resize"},
	)

	selected := SelectExistingCapabilityIDs(nil, capabilities, "image.resize", 2)
	if len(selected) == 0 || selected[0] != "image.resize" {
		t.Fatalf("selected = %#v, want hint first", selected)
	}
}

func TestSelectExistingCapabilityIDsReturnsOnlyPositiveScoresForLargeCatalog(t *testing.T) {
	selected := SelectExistingCapabilityIDs(nil, largeCapabilityCatalog(), "", 10)
	if len(selected) != 0 {
		t.Fatalf("selected = %#v, want no candidates without matching evidence", selected)
	}
}

func largeCapabilityCatalog() []core.Capability {
	capabilities := make([]core.Capability, 0, 51)
	for i := 0; i < 51; i++ {
		capabilities = append(capabilities, core.Capability{ID: fmt.Sprintf("catalog.cap_%02d", i)})
	}
	return capabilities
}

func assertIDs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ids = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("ids = %#v, want %#v", got, want)
		}
	}
}
