package main

import (
	"fmt"
	"os"

	"github.com/spacehz-lab/cal/internal/cald"
)

func main() {
	cmd, err := cald.NewCommand(cald.CommandOptions{
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Environ: os.Environ(),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
