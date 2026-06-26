package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	caldclient "github.com/spacehz-lab/cal/internal/cald/client"
	"github.com/spacehz-lab/cal/internal/logging"
)

// Config carries dependencies and IO for calctl commands.
type Config struct {
	In   io.Reader
	Out  io.Writer
	Err  io.Writer
	Home string
}

// NewRootCommand builds the calctl command tree.
func NewRootCommand(cfg Config) *cobra.Command {
	if cfg.In == nil {
		cfg.In = os.Stdin
	}
	if cfg.Out == nil {
		cfg.Out = os.Stdout
	}
	if cfg.Err == nil {
		cfg.Err = os.Stderr
	}
	logging.Configure(logging.Options{
		Process: "calctl",
		Home:    cfg.Home,
		Err:     cfg.Err,
	})

	root := &cobra.Command{
		Use:           "calctl",
		Short:         "Capability Acquisition Loop",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetIn(cfg.In)
	root.SetOut(cfg.Out)
	root.SetErr(cfg.Err)
	root.AddCommand(newDaemonCommand(cfg))
	root.AddCommand(newProvidersCommand(cfg))
	root.AddCommand(newDiscoveryCommand(cfg))
	root.AddCommand(newCapabilitiesCommand(cfg))
	root.AddCommand(newRunsCommand(cfg))
	root.AddCommand(newUseCommand(cfg))
	root.AddCommand(newEvalCommand(cfg))
	root.AddCommand(newTracesCommand(cfg))
	return root
}

func newCaldClient(cfg Config) (caldclient.Client, error) {
	return caldclient.New(cfg.Home)
}
