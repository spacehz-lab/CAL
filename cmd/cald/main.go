package main

import (
	"fmt"
	"os"

	"github.com/spacehz-lab/cal/internal/cald"
)

func main() {
	cmd := cald.NewCommand(cald.CommandConfig{
		Out: os.Stdout,
		Err: os.Stderr,
	})
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
