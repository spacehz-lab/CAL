package proposalflow

import (
	"strings"
	"testing"
)

func TestCLISurfacePromptRequiresBroadPrimaryCommandInventory(t *testing.T) {
	required := []string{
		"include documented primary commands broadly up to max_surface_items",
		"do not omit a primary command",
		`decision="defer"`,
		"enough to seed Capability planning",
		`kind="option" instead of subcommand`,
	}
	for _, phrase := range required {
		if !strings.Contains(cliSurfaceSystemPrompt, phrase) {
			t.Fatalf("cliSurfaceSystemPrompt missing %q", phrase)
		}
	}
}
