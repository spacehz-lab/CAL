package cald

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/logging"
)

// CommandConfig carries process IO for the cald command tree.
type CommandConfig struct {
	Out  io.Writer
	Err  io.Writer
	Home string
}

// NewCommand builds the cald executable command tree.
func NewCommand(cfg CommandConfig) *cobra.Command {
	logging.Configure(logging.Options{
		Process: "cald",
		Home:    cfg.Home,
		Err:     cfg.Err,
	})
	root := &cobra.Command{
		Use:           "cald",
		Short:         "CAL local service",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	if cfg.Out != nil {
		root.SetOut(cfg.Out)
	}
	if cfg.Err != nil {
		root.SetErr(cfg.Err)
	}

	root.AddCommand(&cobra.Command{
		Use:   "serve",
		Short: "Start the local CAL service",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return serve(cmd.Context(), cfg.Home)
		},
	})

	return root
}
