package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestCommandTreeUsesOnlyFirstVersionCommands(t *testing.T) {
	cmd, _, _ := newTestCLI(t, nil)

	if hasCommand(childCommand(t, cmd, "providers"), "get") {
		t.Fatal("providers get command exists, want deferred command omitted")
	}
	if hasCommand(cmd, "traces") {
		t.Fatal("traces command exists, want deferred command omitted")
	}
	for _, args := range [][]string{
		{"daemon", "status"},
		{"daemon", "stop"},
		{"providers", "add"},
		{"providers", "list"},
		{"acquisition", "run"},
		{"capabilities", "list"},
		{"runs", "create"},
		{"use"},
		{"eval"},
	} {
		if _, _, err := cmd.Find(args); err != nil {
			t.Fatalf("Find(%v) error = %v", args, err)
		}
	}
}

func TestNewResolvesHomeFromEnv(t *testing.T) {
	home := t.TempDir()
	app, err := New(Options{Environ: []string{envHome + "=" + home}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if app.home != home {
		t.Fatalf("home = %q, want %q", app.home, home)
	}
}

func childCommand(t *testing.T, cmd interface{ Commands() []*cobra.Command }, name string) *cobra.Command {
	t.Helper()
	for _, child := range cmd.Commands() {
		if child.Name() == name {
			return child
		}
	}
	t.Fatalf("command %q not found", name)
	return nil
}

func hasCommand(cmd interface{ Commands() []*cobra.Command }, name string) bool {
	for _, child := range cmd.Commands() {
		if child.Name() == name {
			return true
		}
	}
	return false
}
